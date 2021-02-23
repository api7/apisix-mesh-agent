package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	proto2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/id"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestFileProvisionerGenerateEvents(t *testing.T) {
	p := &xdsFileProvisioner{
		logger: log.DefaultLogger,
		state:  make(map[string]*manifest),
	}
	rm := &manifest{
		Routes: []*apisix.Route{
			{
				Id: &apisix.ID{
					OneofId: &apisix.ID_StrVal{
						StrVal: "1",
					},
				},
			},
			{
				Id: &apisix.ID{
					OneofId: &apisix.ID_StrVal{
						StrVal: "2",
					},
				},
			},
		},
	}
	events := p.generateEvents("null", nil, rm)
	assert.Len(t, events, 2)
	assert.Equal(t, events[0].Object, rm.Routes[0])
	assert.Equal(t, events[0].Type, types.EventAdd)
	assert.Equal(t, events[1].Object, rm.Routes[1])
	assert.Equal(t, events[1].Type, types.EventAdd)
	assert.Equal(t, p.state["null"], rm)

	events = p.generateEvents("null", rm, nil)
	assert.Len(t, events, 2)
	assert.Equal(t, events[0].Tombstone, rm.Routes[0])
	assert.Nil(t, events[0].Object)
	assert.Equal(t, events[0].Type, types.EventDelete)
	assert.Equal(t, events[1].Tombstone, rm.Routes[1])
	assert.Nil(t, events[1].Object)
	assert.Equal(t, p.state["null"], (*manifest)(nil))

	rmo := &manifest{
		Routes: []*apisix.Route{
			{
				Id: &apisix.ID{
					OneofId: &apisix.ID_StrVal{
						StrVal: "1",
					},
				},
				Name: "old town",
			},
			{
				Id: &apisix.ID{
					OneofId: &apisix.ID_StrVal{
						StrVal: "3",
					},
				},
			},
		},
	}

	events = p.generateEvents("null", rmo, rm)
	assert.Len(t, events, 3)
	assert.Equal(t, events[0].Type, types.EventAdd)
	assert.Equal(t, events[0].Object, rm.Routes[1])
	assert.Equal(t, events[1].Type, types.EventDelete)
	assert.Equal(t, events[1].Tombstone, rmo.Routes[1])
	assert.Nil(t, events[1].Object)
	assert.Equal(t, events[2].Type, types.EventUpdate)
	assert.Equal(t, events[2].Object, rm.Routes[0])
	assert.Equal(t, p.state["null"], rm)
}

func TestFileProvisionerGenerateEventsFromDiscoveryResponse(t *testing.T) {
	rc := &routev3.RouteConfiguration{
		Name: "rc1",
		VirtualHosts: []*routev3.VirtualHost{
			{
				Name: "vhost1",
				Domains: []string{
					"*.apache.org",
					"apisix.apache.org",
				},
				Routes: []*routev3.Route{
					{
						Name: "route1",
						Match: &routev3.RouteMatch{
							CaseSensitive: &wrappers.BoolValue{
								Value: true,
							},
							PathSpecifier: &routev3.RouteMatch_Path{
								Path: "/foo",
							},
						},
						Action: &routev3.Route_Route{
							Route: &routev3.RouteAction{
								ClusterSpecifier: &routev3.RouteAction_Cluster{
									Cluster: "kubernetes.default.svc.cluster.local",
								},
							},
						},
					},
				},
			},
		},
	}
	c := &clusterv3.Cluster{
		Name: "httpbin.default.svc.cluster.local",
		ClusterDiscoveryType: &clusterv3.Cluster_Type{
			Type: clusterv3.Cluster_EDS,
		},
		LbPolicy: clusterv3.Cluster_ROUND_ROBIN,
	}
	ep := &endpointv3.ClusterLoadAssignment{
		ClusterName: "httpbin.default.svc.cluster.local",
		Endpoints: []*endpointv3.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpointv3.LbEndpoint{
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Protocol: corev3.SocketAddress_TCP,
											Address:  "10.0.3.11",
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: 8000,
											},
										},
									},
								},
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: 100,
						},
					},
				},
			},
		},
	}
	var (
		opaque  any.Any
		opaque2 any.Any
		opaque3 any.Any
	)
	opaque.TypeUrl = "type.googleapis.com/" + string(rc.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque, rc, proto2.MarshalOptions{}))
	opaque2.TypeUrl = "type.googleapis.com/" + string(c.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque2, c, proto2.MarshalOptions{}))
	opaque3.TypeUrl = "type.googleapis.com/" + string(ep.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque3, ep, proto2.MarshalOptions{}))

	s1, _ := protojson.Marshal(&opaque2)
	t.Log(string(s1))
	s2, _ := protojson.Marshal(&opaque3)
	t.Log(string(s2))

	dr := &discoveryv3.DiscoveryResponse{
		VersionInfo: "0",
		Resources:   []*any.Any{&opaque, &opaque2, &opaque3},
	}

	cfg := &config.Config{
		LogLevel:  "debug",
		LogOutput: "stderr",
	}
	adaptor, err := xdsv3.NewAdaptor(cfg)
	assert.Nil(t, err)
	p := &xdsFileProvisioner{
		logger:        log.DefaultLogger,
		v3Adaptor:     adaptor,
		state:         make(map[string]*manifest),
		upstreamCache: make(map[string]*apisix.Upstream),
	}
	events := p.generateEventsFromDiscoveryResponseV3("null", dr)
	assert.Len(t, events, 2)
	assert.Equal(t, events[0].Type, types.EventAdd)
	assert.Equal(t, events[0].Object.(*apisix.Route).Name, "route1.vhost1.rc1")
	assert.Nil(t, events[0].Tombstone)

	assert.Equal(t, events[1].Type, types.EventAdd)
	assert.Equal(t, events[1].Object.(*apisix.Upstream).Name, "httpbin.default.svc.cluster.local")
	assert.Len(t, events[1].Object.(*apisix.Upstream).Nodes, 1)
	assert.Equal(t, events[1].Object.(*apisix.Upstream).Nodes[0].Host, "10.0.3.11")
	assert.Equal(t, events[1].Object.(*apisix.Upstream).Nodes[0].Port, int32(8000))
	assert.Equal(t, events[1].Object.(*apisix.Upstream).Nodes[0].Weight, int32(100))
	assert.Nil(t, events[1].Tombstone)
}

func TestFileProvisionerHandleFileEvent(t *testing.T) {
	cfg := &config.Config{
		LogLevel:              "debug",
		LogOutput:             "stderr",
		UseXDSFileProvisioner: true,
		XDSWatchFiles:         []string{"./testdata"},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, err, "creating xds file provisioner")
	stopCh := make(chan struct{})
	evCh := p.Channel()
	go func() {
		err := p.Run(stopCh)
		assert.Nil(t, err, "launching provisioner")
	}()
	var events []types.Event
	select {
	case events = <-evCh:
		break
	case <-time.After(2 * time.Second):
		t.Fatal("no event arrived in time")
	}
	assert.Len(t, events, 1)
	assert.Equal(t, events[0].Type, types.EventAdd)
	assert.Nil(t, events[0].Tombstone)

	switch obj := events[0].Object.(type) {
	case *apisix.Route:
		assert.Equal(t, obj.Uris[0], "/foo")
		assert.Equal(t, obj.Name, "route1.vhost1.rc1")
		assert.Equal(t, obj.UpstreamId, id.GenID("kubernetes.default.svc.cluster.local"))
		assert.Equal(t, obj.Status, apisix.Route_Enable)
	case *apisix.Upstream:
		assert.Len(t, obj.Nodes, 0)
		assert.Equal(t, obj.Name, "httpbin.default.svc.cluster.local")
	}

	select {
	case events = <-evCh:
		break
	case <-time.After(2 * time.Second):
		t.Fatal("no event arrived in time")
	}
	assert.Len(t, events, 1)
	assert.Equal(t, events[0].Type, types.EventAdd)
	assert.Nil(t, events[0].Tombstone)

	switch obj := events[0].Object.(type) {
	case *apisix.Route:
		assert.Equal(t, obj.Uris[0], "/foo")
		assert.Equal(t, obj.Name, "route1.vhost1.rc1")
		assert.Equal(t, obj.UpstreamId.GetStrVal(), id.GenID("kubernetes.default.svc.cluster.local"))
		assert.Equal(t, obj.Status, apisix.Route_Enable)
	case *apisix.Upstream:
		assert.Len(t, obj.Nodes, 0)
		assert.Equal(t, obj.Name, "httpbin.default.svc.cluster.local")
	}

	eds := `
{
  "versionInfo": "1111",
  "resources": [
    {
      "@type": "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment",
      "clusterName": "httpbin.default.svc.cluster.local",
      "endpoints": [
        {
          "lbEndpoints": [
            {
              "endpoint": {
                "address": {
                  "socketAddress": {
                    "address": "10.0.3.11",
                    "portValue": 8000
                  }
                }
              },
              "loadBalancingWeight": 100
            }
          ]
        }
      ]
    }
  ]
}   
`
	filename := fmt.Sprintf("testdata/eds-%d.json", time.Now().Nanosecond())
	assert.Nil(t, ioutil.WriteFile(filename, []byte(eds), 0644))
	defer os.Remove(filename)

	select {
	case events = <-evCh:
		break
	case <-time.After(2 * time.Second):
		t.Fatal("no event arrived in time")
	}
	assert.Len(t, events, 1)
	assert.Equal(t, events[0].Type, types.EventUpdate)
	assert.Len(t, events[0].Object.(*apisix.Upstream).Nodes, 1)
	assert.Equal(t, events[0].Object.(*apisix.Upstream).Nodes[0].Host, "10.0.3.11")
	assert.Equal(t, events[0].Object.(*apisix.Upstream).Nodes[0].Port, int32(8000))
	assert.Equal(t, events[0].Object.(*apisix.Upstream).Nodes[0].Weight, int32(100))
	assert.Nil(t, events[0].Tombstone)

	assert.Nil(t, os.Remove(filename))
	select {
	case events = <-evCh:
		break
	case <-time.After(2 * time.Second):
		t.Fatal("no event arrived in time")
	}
	assert.Len(t, events, 1)
	assert.Equal(t, events[0].Type, types.EventUpdate)
	assert.Nil(t, events[0].Tombstone)
	assert.Len(t, events[0].Object.(*apisix.Upstream).Nodes, 0)

	close(stopCh)
	_, ok := <-evCh
	assert.Equal(t, ok, false)
}

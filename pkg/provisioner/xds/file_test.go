package xds

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	"github.com/api7/apisix-mesh-agent/pkg/config"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
	proto2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestFileProvisionerGenerateEvents(t *testing.T) {
	p := &xdsFileProvisioner{
		logger: log.DefaultLogger,
		state:  make(map[string]*resourceManifest),
	}
	rm := &resourceManifest{
		routes: []*apisix.Route{
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
	assert.Equal(t, events[0].Object, rm.routes[0])
	assert.Equal(t, events[0].Type, types.EventAdd)
	assert.Equal(t, events[1].Object, rm.routes[1])
	assert.Equal(t, events[1].Type, types.EventAdd)
	assert.Equal(t, p.state["null"], rm)

	events = p.generateEvents("null", rm, nil)
	assert.Len(t, events, 2)
	assert.Equal(t, events[0].Tombstone, rm.routes[0])
	assert.Nil(t, events[0].Object)
	assert.Equal(t, events[0].Type, types.EventDelete)
	assert.Equal(t, events[1].Tombstone, rm.routes[1])
	assert.Nil(t, events[1].Object)
	assert.Equal(t, p.state["null"], (*resourceManifest)(nil))

	rmo := &resourceManifest{
		routes: []*apisix.Route{
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
	assert.Equal(t, events[0].Object, rm.routes[1])
	assert.Equal(t, events[1].Type, types.EventDelete)
	assert.Equal(t, events[1].Tombstone, rmo.routes[1])
	assert.Nil(t, events[1].Object)
	assert.Equal(t, events[2].Type, types.EventUpdate)
	assert.Equal(t, events[2].Object, rm.routes[0])
	assert.Equal(t, p.state["null"], rm)
}

func TestFileProvisionerProcessRouteConfigurationV3(t *testing.T) {
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
	cfg := &config.Config{
		LogLevel:  "debug",
		LogOutput: "stderr",
	}
	adaptor, err := xdsv3.NewAdaptor(cfg)
	assert.Nil(t, err)
	p := &xdsFileProvisioner{
		logger:    log.DefaultLogger,
		v3Adaptor: adaptor,
	}
	var opaque any.Any
	opaque.TypeUrl = "type.googleapis.com/" + string(rc.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque, rc, proto2.MarshalOptions{}))
	routes := p.processRouteConfigurationV3(&opaque)
	assert.Len(t, routes, 1)
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
	var opaque any.Any
	opaque.TypeUrl = "type.googleapis.com/" + string(rc.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque, rc, proto2.MarshalOptions{}))
	dr := &discoveryv3.DiscoveryResponse{
		VersionInfo: "0",
		Resources:   []*any.Any{&opaque},
	}

	cfg := &config.Config{
		LogLevel:  "debug",
		LogOutput: "stderr",
	}
	adaptor, err := xdsv3.NewAdaptor(cfg)
	assert.Nil(t, err)
	p := &xdsFileProvisioner{
		logger:    log.DefaultLogger,
		v3Adaptor: adaptor,
		state:     make(map[string]*resourceManifest),
	}
	events := p.generateEventsFromDiscoveryResponseV3("null", dr)
	assert.Len(t, events, 1)
}

func TestFileProvisionerHandleFileEvent(t *testing.T) {
	cfg := &config.Config{
		LogLevel:              "debug",
		LogOutput:             "stderr",
		UseXDSFileProvisioner: true,
		XDSWatchFiles:         []string{"./testdata"},
	}
	p, err := NewXDSProvisionerFromFiles(cfg)
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
	close(stopCh)
}

package grpc

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/api7/apisix-mesh-agent/pkg/provisioner/util"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/nettest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
	"github.com/api7/apisix-mesh-agent/pkg/version"
)

func TestNewXDSProvisioner(t *testing.T) {
	cfg := &config.Config{
		RunId:           "12345",
		LogLevel:        "info",
		LogOutput:       "stderr",
		Provisioner:     "xds-v3-grpc",
		XDSConfigSource: "abc",
		RunningContext: &config.RunningContext{
			PodNamespace: "default",
			IPAddress:    "1.1.1.1",
		},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, p)
	assert.Equal(t, err.Error(), "bad xds config source")

	cfg.XDSConfigSource = "grpc://127.0.0.1:11111"
	p, err = NewXDSProvisioner(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, p.Channel())

	gp := p.(*grpcProvisioner)
	assert.Equal(t, gp.node.Id, util.GenNodeId(cfg.RunId, "1.1.1.1", "default.svc.cluster.local"))
	assert.Equal(t, gp.node.UserAgentName, "apisix-mesh-agent/"+version.Short())
}

func TestFirstSend(t *testing.T) {
	cfg := &config.Config{
		RunId:           "12345",
		LogLevel:        "info",
		LogOutput:       "stderr",
		Provisioner:     "xds-v3-grpc",
		XDSConfigSource: "grpc://127.0.0.1:11111",
		RunningContext: &config.RunningContext{
			PodNamespace: "default",
			IPAddress:    "1.1.1.1",
		},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, err)
	gp := p.(*grpcProvisioner)

	go func() {
		gp.firstSend()
	}()

	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "DiscoveryRequest is not sent in time")
	case dr := <-gp.sendCh:
		assert.Equal(t, dr.TypeUrl, types.RouteConfigurationUrl)
	}
	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "DiscoveryRequest is not sent in time")
	case dr := <-gp.sendCh:
		assert.Equal(t, dr.TypeUrl, types.ClusterUrl)
	}
}

type fakeClient struct {
	ctx    context.Context
	sendCh chan *discoveryv3.DiscoveryRequest
	recvCh chan *discoveryv3.DiscoveryResponse
}

func (f *fakeClient) Send(r *discoveryv3.DiscoveryRequest) error {
	select {
	case <-time.After(time.Second):
		return errors.New("timed out")
	case f.sendCh <- r:
		return nil
	}
}

func (f *fakeClient) Recv() (*discoveryv3.DiscoveryResponse, error) {
	resp := <-f.recvCh
	return resp, nil
}

func (f *fakeClient) Header() (metadata.MD, error) {
	return nil, nil
}
func (f *fakeClient) Trailer() metadata.MD {
	return nil
}
func (f *fakeClient) CloseSend() error {
	return nil
}
func (f *fakeClient) SendMsg(_ interface{}) error {
	return nil
}
func (f *fakeClient) RecvMsg(_ interface{}) error {
	return nil
}
func (f *fakeClient) Context() context.Context {
	return f.ctx
}

var _ discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesClient = new(fakeClient)

func TestSendLoop(t *testing.T) {
	cfg := &config.Config{
		RunId:           "12345",
		LogLevel:        "info",
		LogOutput:       "stderr",
		Provisioner:     "xds-v3-grpc",
		XDSConfigSource: "grpc://127.0.0.1:11111",
		RunningContext: &config.RunningContext{
			PodNamespace: "default",
			IPAddress:    "1.1.1.1",
		},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, err)
	gp := p.(*grpcProvisioner)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &fakeClient{
		sendCh: make(chan *discoveryv3.DiscoveryRequest),
		recvCh: make(chan *discoveryv3.DiscoveryResponse),
	}

	go func() {
		gp.sendLoop(ctx, client)
	}()

	r := &discoveryv3.DiscoveryRequest{
		VersionInfo: "111",
		Node:        gp.node,
		TypeUrl:     types.RouteConfigurationUrl,
	}
	gp.sendCh <- r
	rr := <-client.sendCh

	assert.Equal(t, rr.VersionInfo, "111")
	assert.Equal(t, rr.TypeUrl, types.RouteConfigurationUrl)
}

func TestRecvLoop(t *testing.T) {
	cfg := &config.Config{
		RunId:           "12345",
		LogLevel:        "info",
		LogOutput:       "stderr",
		Provisioner:     "xds-v3-grpc",
		XDSConfigSource: "grpc://127.0.0.1:11111",
		RunningContext: &config.RunningContext{
			PodNamespace: "default",
			IPAddress:    "1.1.1.1",
		},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, err)
	gp := p.(*grpcProvisioner)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &fakeClient{
		sendCh: make(chan *discoveryv3.DiscoveryRequest),
		recvCh: make(chan *discoveryv3.DiscoveryResponse),
	}

	go func() {
		gp.recvLoop(ctx, client)
	}()

	cluster := &clusterv3.Cluster{
		Name: "httpbin.default.svc.cluster.local",
	}
	val, err := proto.Marshal(cluster)
	assert.Nil(t, err)
	resp := &discoveryv3.DiscoveryResponse{
		VersionInfo: "111",
		TypeUrl:     types.ClusterUrl,
		Resources: []*any.Any{
			{
				TypeUrl: types.ClusterUrl,
				Value:   val,
			},
		},
	}
	client.recvCh <- resp
	resp2 := <-gp.recvCh
	assert.Equal(t, resp2.TypeUrl, types.ClusterUrl)
	assert.Len(t, resp2.Resources, 1)
}

func TestTranslateLoop(t *testing.T) {
	cfg := &config.Config{
		RunId:           "12345",
		LogLevel:        "info",
		LogOutput:       "stderr",
		Provisioner:     "xds-v3-grpc",
		XDSConfigSource: "grpc://127.0.0.1:11111",
		RunningContext: &config.RunningContext{
			PodNamespace: "default",
			IPAddress:    "1.1.1.1",
		},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, err)
	gp := p.(*grpcProvisioner)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		gp.translateLoop(ctx)
	}()

	cluster := &clusterv3.Cluster{
		Name: "httpbin.default.svc.cluster.local",
	}
	val, err := proto.Marshal(cluster)
	assert.Nil(t, err)
	resp := &discoveryv3.DiscoveryResponse{
		VersionInfo: "111",
		TypeUrl:     types.ClusterUrl,
		Resources: []*any.Any{
			{
				TypeUrl: types.ClusterUrl,
				Value:   val,
			},
		},
	}
	gp.recvCh <- resp
	ack := <-gp.sendCh
	assert.Nil(t, ack.ErrorDetail)
	assert.Equal(t, ack.VersionInfo, "111")
	assert.Equal(t, ack.TypeUrl, types.ClusterUrl)
	assert.NotNil(t, ack.Node)
}

func TestTranslate(t *testing.T) {
	cfg := &config.Config{
		RunId:           "12345",
		LogLevel:        "info",
		LogOutput:       "stderr",
		Provisioner:     "xds-v3-grpc",
		XDSConfigSource: "grpc://127.0.0.1:11111",
		RunningContext: &config.RunningContext{
			PodNamespace: "default",
			IPAddress:    "1.1.1.1",
		},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, err)
	gp := p.(*grpcProvisioner)
	// As EDS request might be sent when handling CDS,
	// here we create a buffered chan, to not block the
	// send goroutine.
	gp.sendCh = make(chan *discoveryv3.DiscoveryRequest, 1)

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
	val1, err := proto.Marshal(rc)
	assert.Nil(t, err)
	val2, err := proto.Marshal(c)
	assert.Nil(t, err)
	val3, err := proto.Marshal(ep)
	assert.Nil(t, err)

	dr1 := &discoveryv3.DiscoveryResponse{
		VersionInfo: "111",
		TypeUrl:     types.RouteConfigurationUrl,
		Resources: []*any.Any{
			{
				TypeUrl: types.RouteConfigurationUrl,
				Value:   val1,
			},
		},
	}
	dr2 := &discoveryv3.DiscoveryResponse{
		VersionInfo: "111",
		TypeUrl:     types.ClusterUrl,
		Resources: []*any.Any{
			{
				TypeUrl: types.ClusterUrl,
				Value:   val2,
			},
		},
	}
	dr3 := &discoveryv3.DiscoveryResponse{
		VersionInfo: "111",
		TypeUrl:     types.ClusterLoadAssignmentUrl,
		Resources: []*any.Any{
			{
				TypeUrl: types.ClusterLoadAssignmentUrl,
				Value:   val3,
			},
		},
	}

	err = gp.translate(dr1)
	assert.Nil(t, err)
	evs := <-gp.evChan
	assert.Len(t, evs, 1)
	assert.Equal(t, evs[0].Type, types.EventAdd)
	assert.Equal(t, evs[0].Object.(*apisix.Route).Name, "route1.vhost1.rc1")
	assert.Len(t, gp.routes, 1)

	err = gp.translate(dr2)
	assert.Nil(t, err)
	evs = <-gp.evChan
	assert.Len(t, evs, 1)
	assert.Equal(t, evs[0].Type, types.EventAdd)
	assert.Equal(t, evs[0].Object.(*apisix.Upstream).Name, "httpbin.default.svc.cluster.local")
	assert.Len(t, evs[0].Object.(*apisix.Upstream).Nodes, 0)
	assert.Len(t, gp.upstreams, 1)

	err = gp.translate(dr3)
	assert.Nil(t, err)
	evs = <-gp.evChan
	assert.Len(t, evs, 1)
	assert.Equal(t, evs[0].Type, types.EventUpdate)
	assert.Len(t, evs[0].Object.(*apisix.Upstream).Nodes, 1)
	assert.Equal(t, evs[0].Object.(*apisix.Upstream).Nodes[0].Host, "10.0.3.11")
	assert.Equal(t, evs[0].Object.(*apisix.Upstream).Nodes[0].Port, int32(8000))
}

type fakeXdsServer struct {
	t      *testing.T
	ctx    context.Context
	recvCh chan *discoveryv3.DiscoveryRequest
	sendCh chan *discoveryv3.DiscoveryResponse
}

func (srv *fakeXdsServer) StreamAggregatedResources(stream discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				return
			}
			srv.recvCh <- req
		}
	}()

	go func() {
		for {
			resp := <-srv.sendCh
			err := stream.Send(resp)
			if err != nil {
				return
			}
		}
	}()

	<-srv.ctx.Done()
	return nil
}

func (srv *fakeXdsServer) DeltaAggregatedResources(_ discoveryv3.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return errors.New("not yet implemented")
}

func TestGRPCProvisioner(t *testing.T) {
	ln, err := nettest.NewLocalListener("tcp")
	assert.Nil(t, err)
	grpcSrv := grpc.NewServer()
	go func() {
		err := grpcSrv.Serve(ln)
		assert.Nil(t, err)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &fakeXdsServer{
		t:      t,
		sendCh: make(chan *discoveryv3.DiscoveryResponse),
		recvCh: make(chan *discoveryv3.DiscoveryRequest),
		ctx:    ctx,
	}
	discoveryv3.RegisterAggregatedDiscoveryServiceServer(grpcSrv, srv)

	cfg := &config.Config{
		RunId:           "12345",
		LogLevel:        "info",
		LogOutput:       "stderr",
		Provisioner:     "xds-v3-grpc",
		XDSConfigSource: "grpc://" + ln.Addr().String(),
		RunningContext: &config.RunningContext{
			PodNamespace: "default",
			IPAddress:    "1.1.1.1",
		},
	}
	p, err := NewXDSProvisioner(cfg)
	assert.Nil(t, err)

	stopCh := make(chan struct{})
	go func() {
		err := p.Run(stopCh)
		assert.Nil(t, err)
	}()

	var urls []string
	dr := <-srv.recvCh
	urls = append(urls, dr.TypeUrl)
	dr = <-srv.recvCh
	urls = append(urls, dr.TypeUrl)
	urls = append(urls, dr.TypeUrl)

	sort.Strings(urls)
	assert.Equal(t, urls[0], types.ClusterUrl)
	assert.Equal(t, urls[2], types.RouteConfigurationUrl)

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
	val, err := proto.Marshal(rc)
	assert.Nil(t, err)
	resp := &discoveryv3.DiscoveryResponse{
		VersionInfo: "1",
		Resources: []*any.Any{
			{
				TypeUrl: types.RouteConfigurationUrl,
				Value:   val,
			},
		},
		TypeUrl: types.RouteConfigurationUrl,
	}
	srv.sendCh <- resp
	gp := p.(*grpcProvisioner)
	ev := <-gp.evChan
	assert.Len(t, ev, 1)
	assert.Equal(t, ev[0].Object.(*apisix.Route).Name, "route1.vhost1.rc1")
	ack := <-srv.recvCh
	assert.Nil(t, ack.ErrorDetail, nil)
	assert.Equal(t, ack.TypeUrl, types.RouteConfigurationUrl)
}

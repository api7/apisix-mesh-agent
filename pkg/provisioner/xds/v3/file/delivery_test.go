package file

import (
	"testing"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	proto2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/id"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestProcessRouteConfigurationV3(t *testing.T) {
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

func TestProcessClusterV3(t *testing.T) {
	c := &clusterv3.Cluster{
		Name:     "httpbin.default.svc.cluster.local",
		LbPolicy: clusterv3.Cluster_ROUND_ROBIN,
		LoadAssignment: &endpointv3.ClusterLoadAssignment{
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
						{
							HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
								Endpoint: &endpointv3.Endpoint{
									Address: &corev3.Address{
										Address: &corev3.Address_SocketAddress{
											SocketAddress: &corev3.SocketAddress{
												Protocol: corev3.SocketAddress_TCP,
												Address:  "10.0.3.12",
												PortSpecifier: &corev3.SocketAddress_PortValue{
													PortValue: 8000,
												},
											},
										},
									},
								},
							},
							LoadBalancingWeight: &wrappers.UInt32Value{
								Value: 80,
							},
						},
					},
				},
			},
		},
	}
	var opaque any.Any
	opaque.TypeUrl = "type.googleapis.com/" + string(c.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque, c, proto2.MarshalOptions{}))
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
	upstreams := p.processClusterV3(&opaque)
	assert.Len(t, upstreams, 1)
	assert.Equal(t, upstreams[0].Name, "httpbin.default.svc.cluster.local")
	assert.Equal(t, upstreams[0].Id.GetStrVal(), id.GenID(upstreams[0].Name))
	assert.Len(t, upstreams[0].Nodes, 2)
	assert.Equal(t, upstreams[0].Nodes[0].Host, "10.0.3.11")
	assert.Equal(t, upstreams[0].Nodes[0].Port, int32(8000))
	assert.Equal(t, upstreams[0].Nodes[0].Weight, int32(100))
	assert.Equal(t, upstreams[0].Nodes[1].Host, "10.0.3.12")
	assert.Equal(t, upstreams[0].Nodes[1].Port, int32(8000))
	assert.Equal(t, upstreams[0].Nodes[1].Weight, int32(80))
}

func TestProcessClusterLoadAssignment(t *testing.T) {
	cla := &endpointv3.ClusterLoadAssignment{
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
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Protocol: corev3.SocketAddress_TCP,
											Address:  "10.0.3.12",
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: 8000,
											},
										},
									},
								},
							},
						},
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: 80,
						},
					},
				},
			},
		},
	}
	var opaque any.Any
	opaque.TypeUrl = "type.googleapis.com/" + string(cla.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque, cla, proto2.MarshalOptions{}))
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
	// Reject since the cluster is unknown.
	assert.Nil(t, p.processClusterLoadAssignmentV3(&opaque))

	ups := &apisix.Upstream{
		Name: "httpbin.default.svc.cluster.local",
		Nodes: []*apisix.Node{
			{
				Host:   "127.0.0.1",
				Port:   9333,
				Weight: 100,
			},
		},
	}
	p.upstreamCache[ups.Name] = ups
	// Reject since the cluster already has endpoints.
	assert.Nil(t, p.processClusterLoadAssignmentV3(&opaque))

	ups.Nodes = nil
	p.upstreamCache[ups.Name] = ups

	uset := p.processClusterLoadAssignmentV3(&opaque)
	assert.Len(t, uset, 1)
	assert.Len(t, uset[0].Nodes, 2)
	assert.Equal(t, uset[0].Nodes[0].Host, "10.0.3.11")
	assert.Equal(t, uset[0].Nodes[0].Port, int32(8000))
	assert.Equal(t, uset[0].Nodes[0].Weight, int32(100))

	assert.Equal(t, uset[0].Nodes[1].Host, "10.0.3.12")
	assert.Equal(t, uset[0].Nodes[1].Port, int32(8000))
	assert.Equal(t, uset[0].Nodes[1].Weight, int32(80))
}

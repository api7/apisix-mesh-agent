package grpc

import (
	"testing"

	"github.com/api7/apisix-mesh-agent/pkg/id"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	proto2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
	p := &grpcProvisioner{
		logger:    log.DefaultLogger,
		v3Adaptor: adaptor,
	}
	var opaque any.Any
	opaque.TypeUrl = "type.googleapis.com/" + string(rc.ProtoReflect().Descriptor().FullName())
	assert.Nil(t, anypb.MarshalFrom(&opaque, rc, proto2.MarshalOptions{}))
	routes, err := p.processRouteConfigurationV3(&opaque)
	assert.Nil(t, err)
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
	p := &grpcProvisioner{
		logger:    log.DefaultLogger,
		v3Adaptor: adaptor,
		upstreams: make(map[string]*apisix.Upstream),
	}
	ups, err := p.processClusterV3(&opaque)
	assert.Nil(t, err)
	assert.Equal(t, ups.Name, "httpbin.default.svc.cluster.local")
	assert.Equal(t, ups.Id, id.GenID(ups.Name))
	assert.Len(t, ups.Nodes, 2)
	assert.Equal(t, ups.Nodes[0].Host, "10.0.3.11")
	assert.Equal(t, ups.Nodes[0].Port, int32(8000))
	assert.Equal(t, ups.Nodes[0].Weight, int32(100))
	assert.Equal(t, ups.Nodes[1].Host, "10.0.3.12")
	assert.Equal(t, ups.Nodes[1].Port, int32(8000))
	assert.Equal(t, ups.Nodes[1].Weight, int32(80))
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
	p := &grpcProvisioner{
		logger:    log.DefaultLogger,
		v3Adaptor: adaptor,
		upstreams: make(map[string]*apisix.Upstream),
	}
	// Reject since the cluster is unknown.
	ups, err := p.processClusterLoadAssignmentV3(&opaque)
	assert.Nil(t, ups)
	assert.Equal(t, err, _errUnknownClusterName)

	ups = &apisix.Upstream{
		Name: "httpbin.default.svc.cluster.local",
		Nodes: []*apisix.Node{
			{
				Host:   "127.0.0.1",
				Port:   9333,
				Weight: 100,
			},
		},
	}
	p.upstreams[ups.Name] = ups

	ups = &apisix.Upstream{
		Name: "httpbin.default.svc.cluster.local",
	}
	p.upstreams[ups.Name] = ups

	ups, err = p.processClusterLoadAssignmentV3(&opaque)
	assert.Nil(t, err)
	assert.NotNil(t, ups)
	assert.Len(t, ups.Nodes, 2)
	assert.Equal(t, ups.Nodes[0].Host, "10.0.3.11")
	assert.Equal(t, ups.Nodes[0].Port, int32(8000))
	assert.Equal(t, ups.Nodes[0].Weight, int32(100))

	assert.Equal(t, ups.Nodes[1].Host, "10.0.3.12")
	assert.Equal(t, ups.Nodes[1].Port, int32(8000))
	assert.Equal(t, ups.Nodes[1].Weight, int32(80))
}

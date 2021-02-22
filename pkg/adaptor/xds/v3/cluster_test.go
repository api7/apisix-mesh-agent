package v3

import (
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	"github.com/golang/protobuf/ptypes/wrappers"

	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestTranslateClusterLbPolicy(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}
	c := &clusterv3.Cluster{
		Name:     "test",
		LbPolicy: clusterv3.Cluster_ROUND_ROBIN,
	}
	var ups apisix.Upstream
	assert.Nil(t, a.translateClusterLbPolicy(c, &ups))
	assert.Equal(t, ups.Type, "roundrobin")
	c.LbPolicy = clusterv3.Cluster_LEAST_REQUEST
	assert.Nil(t, a.translateClusterLbPolicy(c, &ups))
	assert.Equal(t, ups.Type, "least_conn")

	c.LbPolicy = clusterv3.Cluster_RING_HASH
	assert.Equal(t, a.translateClusterLbPolicy(c, &ups), ErrFeatureNotSupportedYet)
}

func TestTranslateClusterTimeoutSettings(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}
	c := &clusterv3.Cluster{
		Name: "test",
		ConnectTimeout: &duration.Duration{
			Seconds: 10,
		},
		LbPolicy: clusterv3.Cluster_ROUND_ROBIN,
	}
	var ups apisix.Upstream
	assert.Nil(t, a.translateClusterTimeoutSettings(c, &ups))
	assert.Equal(t, ups.Timeout.Connect, float64(10))
}

func TestTranslateClusterLoadAssignment(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}
	la := &endpointv3.ClusterLoadAssignment{
		ClusterName: "test",
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
						// Will be ignored.
						HostIdentifier: &endpointv3.LbEndpoint_EndpointName{},
					},
					{
						// Will be ignored.
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_Pipe{},
								},
							},
						},
					},
					{
						// Will be ignored.
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Protocol: corev3.SocketAddress_UDP,
											Address:  "10.0.3.11",
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: 8000,
											},
										},
									},
								},
							},
						},
					},
					{
						// Will be ignored.
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Protocol: corev3.SocketAddress_TCP,
											Address:  "10.0.3.12",
											PortSpecifier: &corev3.SocketAddress_NamedPort{
												NamedPort: "http",
											},
										},
									},
								},
							},
						},
					},
				},
				LoadBalancingWeight: &wrappers.UInt32Value{
					Value: 50,
				},
			},
		},
	}
	nodes, err := a.TranslateClusterLoadAssignment(la)
	assert.Nil(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, nodes[0].Port, int32(8000))
	assert.Equal(t, nodes[0].Weight, int32(100))
	assert.Equal(t, nodes[0].Host, "10.0.3.11")
}

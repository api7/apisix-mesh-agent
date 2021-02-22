package v3

import (
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/id"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func (adaptor *adaptor) TranslateCluster(c *clusterv3.Cluster) (*apisix.Upstream, error) {
	ups := &apisix.Upstream{
		Name: c.Name,
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: id.GenID(c.Name),
			},
		},
	}
	if err := adaptor.translateClusterLbPolicy(c, ups); err != nil {
		return nil, err
	}
	if err := adaptor.translateClusterTimeoutSettings(c, ups); err != nil {
		return nil, err
	}
	if err := adaptor.translateClusterLoadAssignments(c, ups); err != nil {
		return nil, err
	}

	adaptor.logger.Debugw("got upstream after parsing cluster",
		zap.Any("cluster", c),
	)

	return ups, nil
}

func (adaptor *adaptor) translateClusterLbPolicy(c *clusterv3.Cluster, ups *apisix.Upstream) error {
	switch c.GetLbPolicy() {
	case clusterv3.Cluster_ROUND_ROBIN:
		ups.Type = "roundrobin"
	case clusterv3.Cluster_LEAST_REQUEST:
		// Apache APISIX's lease_conn policy is same to lease request.
		// But is doesn't expose configuration items. So LbConfig field
		// is ignored.
		ups.Type = "least_conn"
	default:
		// Apache APISIX doesn't support Random, Manglev. In addition,
		// also RinghHash (Consistent Hash) is available but the configurations
		// like key is in RouteConfiguration, so we cannot use it either.
		adaptor.logger.Warnw("ignore cluster with unsupported load balancer",
			zap.String("cluster_name", c.Name),
			zap.String("lb_policy", c.GetLbPolicy().String()),
		)
		return ErrFeatureNotSupportedYet
	}
	return nil
}

func (adaptor *adaptor) translateClusterTimeoutSettings(c *clusterv3.Cluster, ups *apisix.Upstream) error {
	if c.GetConnectTimeout() != nil {
		ups.Timeout = &apisix.Upstream_Timeout{
			Connect: float64((*c.GetConnectTimeout()).Seconds),
		}
	}
	return nil
}

func (adaptor *adaptor) translateClusterLoadAssignments(c *clusterv3.Cluster, ups *apisix.Upstream) error {
	if c.GetClusterType() != nil {
		return ErrFeatureNotSupportedYet
	}
	switch c.GetType() {
	case clusterv3.Cluster_EDS:
		return ErrRequireFurtherEDS
	default:
		nodes, err := adaptor.TranslateClusterLoadAssignment(c.GetLoadAssignment())
		if err != nil {
			return err
		}
		ups.Nodes = nodes
		return nil
	}
}

func (adaptor *adaptor) TranslateClusterLoadAssignment(la *endpointv3.ClusterLoadAssignment) ([]*apisix.Node, error) {
	var nodes []*apisix.Node
	for _, eps := range la.GetEndpoints() {
		var weight int32
		if eps.GetLoadBalancingWeight() != nil {
			weight = int32(eps.GetLoadBalancingWeight().GetValue())
		} else {
			weight = 100
		}
		for _, ep := range eps.LbEndpoints {
			node := &apisix.Node{
				Weight: weight,
			}
			if ep.GetLoadBalancingWeight() != nil {
				node.Weight = int32(ep.GetLoadBalancingWeight().GetValue())
			}
			switch identifier := ep.GetHostIdentifier().(type) {
			case *endpointv3.LbEndpoint_Endpoint:
				switch addr := identifier.Endpoint.Address.Address.(type) {
				case *corev3.Address_SocketAddress:
					if addr.SocketAddress.GetProtocol() != corev3.SocketAddress_TCP {
						adaptor.logger.Warnw("ignore endpoint with non-tcp protocol",
							zap.Any("endpoint", ep),
						)
						continue
					}
					node.Host = addr.SocketAddress.GetAddress()
					switch port := addr.SocketAddress.GetPortSpecifier().(type) {
					case *corev3.SocketAddress_PortValue:
						node.Port = int32(port.PortValue)
					case *corev3.SocketAddress_NamedPort:
						adaptor.logger.Warnw("ignore endpoint with unsupported named port",
							zap.Any("endpoint", ep),
						)
						continue
					}
				default:
					adaptor.logger.Warnw("ignore endpoint with unsupported address type",
						zap.Any("endpoint", ep),
					)
					continue
				}
			default:
				adaptor.logger.Warnw("ignore endpoint with unknown endpoint type ",
					zap.Any("endpoint", ep),
				)
				continue
			}
			adaptor.logger.Debugw("got node after parsing endpoint",
				zap.Any("node", node),
				zap.Any("endpoint", ep),
			)
			// Currently Apache APISIX doesn't use the metadata field.
			// So we don't pass ep.Metadata.
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

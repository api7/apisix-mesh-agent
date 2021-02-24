package file

import (
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes/any"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func (p *xdsFileProvisioner) processRouteConfigurationV3(res *any.Any) []*apisix.Route {
	var route routev3.RouteConfiguration
	err := anypb.UnmarshalTo(res, &route, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})
	if err != nil {
		p.logger.Errorw("found invalid RouteConfiguration resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil
	}

	routes, err := p.v3Adaptor.TranslateRouteConfiguration(&route)
	if err != nil {
		p.logger.Errorw("failed to translate RouteConfiguration to APISIX routes",
			zap.Error(err),
			zap.Any("route", &route),
		)
	}
	return routes
}

func (p *xdsFileProvisioner) processClusterV3(res *any.Any) []*apisix.Upstream {
	var cluster clusterv3.Cluster
	err := anypb.UnmarshalTo(res, &cluster, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})
	if err != nil {
		p.logger.Errorw("found invalid Cluster resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil
	}
	ups, err := p.v3Adaptor.TranslateCluster(&cluster)
	if err != nil && err != xdsv3.ErrRequireFurtherEDS {
		p.logger.Errorw("failed to translate Cluster to APISIX routes",
			zap.Error(err),
			zap.Any("cluster", &cluster),
		)
		return nil
	}
	if err == xdsv3.ErrRequireFurtherEDS {
		p.logger.Warnw("cluster depends on another EDS config, an upstream without nodes setting was generated",
			zap.Any("upstream", ups),
		)
	}
	p.upstreamCache[ups.Name] = ups
	return []*apisix.Upstream{ups}
}

func (p *xdsFileProvisioner) processClusterLoadAssignmentV3(res *any.Any) []*apisix.Upstream {
	var cla endpointv3.ClusterLoadAssignment
	err := anypb.UnmarshalTo(res, &cla, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})
	if err != nil {
		p.logger.Errorw("found invalid ClusterLoadAssignment resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil
	}

	ups, ok := p.upstreamCache[cla.ClusterName]
	if !ok {
		p.logger.Warnw("found invalid ClusterLoadAssignment resource",
			zap.String("reason", "cluster unknown"),
			zap.Any("resource", res),
		)
		return nil
	}
	if len(ups.Nodes) > 0 {
		p.logger.Warnw("found redundant ClusterLoadAssignment resource",
			zap.String("reason", "Cluster already has load assignment"),
			zap.Any("resource", res),
		)
		return nil
	}

	nodes, err := p.v3Adaptor.TranslateClusterLoadAssignment(&cla)
	if err != nil {
		p.logger.Errorw("failed to translate ClusterLoadAssignment",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil
	}

	// Do not set on the original ups to avoid race conditions.
	newUps := proto.Clone(ups).(*apisix.Upstream)
	newUps.Nodes = nodes
	p.upstreamCache[cla.ClusterName] = newUps
	return []*apisix.Upstream{newUps}
}

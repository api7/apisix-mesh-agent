package grpc

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

func (p *grpcProvisioner) processRouteConfigurationV3(res *any.Any) ([]*apisix.Route, error) {
	var route routev3.RouteConfiguration
	err := anypb.UnmarshalTo(res, &route, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})
	if err != nil {
		p.logger.Errorw("found invalid RouteConfiguration resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil, err
	}

	opts := &xdsv3.TranslateOptions{
		RouteOriginalDestination: p.routeOwnership,
	}
	routes, err := p.v3Adaptor.TranslateRouteConfiguration(&route, opts)
	if err != nil {
		p.logger.Errorw("failed to translate RouteConfiguration to APISIX routes",
			zap.Error(err),
			zap.Any("route", &route),
		)
		return nil, err
	}
	return routes, nil
}

func (p *grpcProvisioner) processStaticRouteConfigurations(rcs []*routev3.RouteConfiguration) ([]*apisix.Route, error) {
	var (
		routes []*apisix.Route
	)
	opts := &xdsv3.TranslateOptions{
		RouteOriginalDestination: p.routeOwnership,
	}
	for _, rc := range rcs {
		route, err := p.v3Adaptor.TranslateRouteConfiguration(rc, opts)
		if err != nil {
			p.logger.Errorw("failed to translate RouteConfiguration to APISIX routes",
				zap.Error(err),
				zap.Any("route", &route),
			)
			return nil, err
		}
	}
	return routes, nil
}

func (p *grpcProvisioner) processClusterV3(res *any.Any) (*apisix.Upstream, error) {
	var cluster clusterv3.Cluster
	err := anypb.UnmarshalTo(res, &cluster, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})
	if err != nil {
		p.logger.Errorw("found invalid Cluster resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil, err
	}
	ups, err := p.v3Adaptor.TranslateCluster(&cluster)
	if err != nil && err != xdsv3.ErrRequireFurtherEDS {
		return nil, err
	}
	if err == xdsv3.ErrRequireFurtherEDS {
		p.logger.Warnw("cluster depends on another EDS config, an upstream without nodes setting was generated",
			zap.Any("upstream", ups),
		)
		p.edsRequiredClusters.Add(ups.Name)
	}
	return ups, nil
}

func (p *grpcProvisioner) processClusterLoadAssignmentV3(res *any.Any) (*apisix.Upstream, error) {
	var cla endpointv3.ClusterLoadAssignment
	err := anypb.UnmarshalTo(res, &cla, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})
	if err != nil {
		p.logger.Errorw("found invalid ClusterLoadAssignment resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil, err
	}

	ups, ok := p.upstreams[cla.ClusterName]
	if !ok {
		p.logger.Warnw("found invalid ClusterLoadAssignment resource",
			zap.String("reason", "cluster unknown"),
			zap.Any("resource", res),
		)
		return nil, _errUnknownClusterName
	}

	nodes, err := p.v3Adaptor.TranslateClusterLoadAssignment(&cla)
	if err != nil {
		p.logger.Errorw("failed to translate ClusterLoadAssignment",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil, err
	}

	// Do not set on the original ups to avoid race conditions.
	newUps := proto.Clone(ups).(*apisix.Upstream)
	newUps.Nodes = nodes
	p.upstreams[cla.ClusterName] = newUps
	return newUps, nil
}

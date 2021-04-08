package v3

import (
	"github.com/api7/apisix-mesh-agent/pkg/log"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xdswellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	_hcmv3 = "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
)

func (adaptor *adaptor) CollectRouteNamesAndConfigs(l *listenerv3.Listener) ([]string, []*routev3.RouteConfiguration, error) {
	var (
		rdsNames      []string
		staticConfigs []*routev3.RouteConfiguration
	)

	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			if f.Name == xdswellknown.HTTPConnectionManager && f.GetTypedConfig().GetTypeUrl() == _hcmv3 {
				var hcm hcmv3.HttpConnectionManager
				if err := anypb.UnmarshalTo(f.GetTypedConfig(), &hcm, proto.UnmarshalOptions{}); err != nil {
					log.Errorw("failed to unmarshal HttpConnectionManager config",
						zap.Error(err),
						zap.Any("listener", l),
					)
					return nil, nil, err
				}
				if hcm.GetRds() != nil {
					rdsNames = append(rdsNames, hcm.GetRds().GetRouteConfigName())
				} else if hcm.GetRouteConfig() != nil {
					// TODO deep copy?
					staticConfigs = append(staticConfigs, hcm.GetRouteConfig())
				}
			}
		}
	}
	log.Debugw("got route names and config from listener",
		zap.Strings("route_names", rdsNames),
		zap.Any("route_configs", staticConfigs),
		zap.Any("listener", l),
	)
	return rdsNames, staticConfigs, nil
}

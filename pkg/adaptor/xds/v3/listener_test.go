package v3

import (
	"testing"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xdswellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/api7/apisix-mesh-agent/pkg/log"
)

func TestCollectRouteNamesAndConfigs(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}

	var (
		any1 anypb.Any
		any2 anypb.Any
		any3 anypb.Any
	)

	f1 := &hcmv3.HttpConnectionManager{
		RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
			Rds: &hcmv3.Rds{
				RouteConfigName: "route1",
			},
		},
	}
	f2 := &hcmv3.HttpConnectionManager{
		RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
			Rds: &hcmv3.Rds{
				RouteConfigName: "route2",
			},
		},
	}
	f3 := &hcmv3.HttpConnectionManager{
		RouteSpecifier: &hcmv3.HttpConnectionManager_RouteConfig{
			RouteConfig: &routev3.RouteConfiguration{
				Name: "route3",
				VirtualHosts: []*routev3.VirtualHost{
					{
						Name: "v1",
						Routes: []*routev3.Route{
							{
								Name: "route1",
							},
						},
					},
				},
			},
		},
	}

	assert.Nil(t, anypb.MarshalFrom(&any1, f1, proto.MarshalOptions{}))
	assert.Nil(t, anypb.MarshalFrom(&any2, f2, proto.MarshalOptions{}))
	assert.Nil(t, anypb.MarshalFrom(&any3, f3, proto.MarshalOptions{}))

	listener := &listenerv3.Listener{
		Name: "listener1",
		FilterChains: []*listenerv3.FilterChain{
			{
				Filters: []*listenerv3.Filter{
					{
						Name: xdswellknown.HTTPConnectionManager,
						ConfigType: &listenerv3.Filter_TypedConfig{
							TypedConfig: &any1,
						},
					},
					{
						Name: xdswellknown.HTTPConnectionManager,
						ConfigType: &listenerv3.Filter_TypedConfig{
							TypedConfig: &any2,
						},
					},
					{
						Name: xdswellknown.HTTPConnectionManager,
						ConfigType: &listenerv3.Filter_TypedConfig{
							TypedConfig: &any3,
						},
					},
				},
			},
		},
	}
	rdsNames, staticConfigs, err := a.CollectRouteNamesAndConfigs(listener)
	assert.Nil(t, err)
	assert.Equal(t, rdsNames, []string{"route1", "route2"})
	assert.Len(t, staticConfigs, 1)
	assert.Equal(t, staticConfigs[0].Name, "route3")
	assert.Len(t, staticConfigs[0].VirtualHosts, 1)
	assert.Equal(t, staticConfigs[0].VirtualHosts[0].Name, "v1")
}

package v3

import (
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

// XDSAdaptor translates xDS resources like Route, Cluster
// to the equivalent configs in Apache APISIX.
type XDSAdaptor interface {
	// TranslateRouteConfiguration translate a RouteConfiguration to a series APISIX
	// Routes.
	// WARNING: not all fields are translated, only the necessary parts are used, others
	// can be added in the future.
	TranslateRouteConfiguration(*routev3.RouteConfiguration) ([]*apisix.Route, error)
}

type xdsAdaptor struct {
	logger *log.Logger
}

// NewXDSAdaptor creates a XDS based adaptor.
func NewXDSAdaptor(cfg *config.Config) (XDSAdaptor, error) {
	logger, err := log.NewLogger(
		log.WithOutputFile(cfg.LogOutput),
		log.WithLogLevel(cfg.LogLevel),
		log.WithContext("xds_adaptor"),
	)
	if err != nil {
		return nil, err
	}
	return &xdsAdaptor{
		logger: logger,
	}, nil
}

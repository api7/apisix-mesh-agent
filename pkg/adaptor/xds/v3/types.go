package v3

import (
	"errors"

	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

var (
	// ErrRequireFurtherEDS means the translation of Cluster is not complete
	// since it depends on EDS to fetch the load assignment (endpoints).
	// Once this error was given, the Cluster should keep invisible until
	// the EDS config arrived.
	ErrRequireFurtherEDS = errors.New("required further EDS config")
	// ErrFeatureNotSupportedYet means a non-supported feature exists in the
	// xDS resource so the Adaptor goes ahead.
	ErrFeatureNotSupportedYet = errors.New("feature not supported yet")
)

// Adaptor translates xDS resources like Route, Cluster
// to the equivalent configs in Apache APISIX.
// WARNING: not all fields are translated, only the necessary parts are used, others
// can be added in the future.
type Adaptor interface {
	// TranslateRouteConfiguration translates a RouteConfiguration to a series APISIX
	// Routes.
	TranslateRouteConfiguration(*routev3.RouteConfiguration) ([]*apisix.Route, error)
	// TranslateCluster translates a Cluster to an APISIX Upstreams.
	TranslateCluster(*clusterv3.Cluster) (*apisix.Upstream, error)
	// TranslateClusterLoadAssignment translate the ClusterLoadAssignement resources to APISIX
	// Upstream Nodes.
	TranslateClusterLoadAssignment(*endpointv3.ClusterLoadAssignment) ([]*apisix.Node, error)
}

type adaptor struct {
	logger *log.Logger
}

// NewAdaptor creates a XDS based adaptor.
func NewAdaptor(cfg *config.Config) (Adaptor, error) {
	logger, err := log.NewLogger(
		log.WithOutputFile(cfg.LogOutput),
		log.WithLogLevel(cfg.LogLevel),
		log.WithContext("xds_v3_adaptor"),
	)
	if err != nil {
		return nil, err
	}
	return &adaptor{
		logger: logger,
	}, nil
}

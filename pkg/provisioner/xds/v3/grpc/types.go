package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcp "google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner/util"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
	"github.com/api7/apisix-mesh-agent/pkg/version"
)

var (
	_errUnknownResourceTypeUrl = errors.New("unknown resource type url")
	_errUnknownClusterName     = errors.New("unknown cluster name")
	_errRedundantEDS           = errors.New("redundant eds config")
)

// Note this provisioner is based on the xDS State of The World
// protocol, not the Delta one.
type grpcProvisioner struct {
	configSource string
	node         *corev3.Node
	logger       *log.Logger
	evChan       chan []types.Event
	v3Adaptor    xdsv3.Adaptor

	routes []*apisix.Route
	// map is necessary since EDS requires the original cluster
	// by the name.
	upstreams map[string]*apisix.Upstream

	sendCh chan *discoveryv3.DiscoveryRequest
	recvCh chan *discoveryv3.DiscoveryResponse
}

// NewXDSProvisioner creates a provisioner which fetches config over gRPC.
func NewXDSProvisioner(cfg *config.Config) (provisioner.Provisioner, error) {
	if !strings.HasPrefix(cfg.XDSConfigSource, "grpc://") {
		return nil, errors.New("bad xds config source")
	}
	cs := strings.TrimPrefix(cfg.XDSConfigSource, "grpc://")
	logger, err := log.NewLogger(
		log.WithOutputFile(cfg.LogOutput),
		log.WithLogLevel(cfg.LogLevel),
		log.WithContext("xds-grpc-provisioner"),
	)
	if err != nil {
		return nil, err
	}
	adapter, err := xdsv3.NewAdaptor(cfg)
	if err != nil {
		return nil, err
	}

	// TODO Configurable domain suffix.
	dnsDomain := cfg.RunningContext.PodNamespace + ".svc.cluster.local"
	node := &corev3.Node{
		Id:            util.GenNodeId(cfg.RunId, cfg.RunningContext.IPAddress, dnsDomain),
		UserAgentName: fmt.Sprintf("apisix-mesh-agent/%s", version.Short()),
	}
	return &grpcProvisioner{
		node:         node,
		configSource: cs,
		logger:       logger,
		evChan:       make(chan []types.Event),
		v3Adaptor:    adapter,
		sendCh:       make(chan *discoveryv3.DiscoveryRequest),
		recvCh:       make(chan *discoveryv3.DiscoveryResponse),
		upstreams:    make(map[string]*apisix.Upstream),
	}, nil
}

func (p *grpcProvisioner) Channel() <-chan []types.Event {
	return p.evChan
}

func (p *grpcProvisioner) Run(stop chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// TODO Support Credentials.
	conn, err := grpcp.DialContext(ctx, p.configSource,
		grpcp.WithInsecure(),
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			p.logger.Errorw("failed to close gRPC connection to XDS config source",
				zap.Error(err),
				zap.String("config_source", p.configSource),
			)
		}
	}()

	client, err := discoveryv3.NewAggregatedDiscoveryServiceClient(conn).StreamAggregatedResources(ctx)
	if err != nil {
		return err
	}

	go p.sendLoop(ctx, client)
	go p.recvLoop(ctx, client)
	go p.translateLoop(ctx)

	p.firstSend()
	<-stop
	close(p.evChan)
	return nil
}

func (p *grpcProvisioner) firstSend() {
	dr1 := &discoveryv3.DiscoveryRequest{
		Node:    p.node,
		TypeUrl: types.RouteConfigurationUrl,
	}
	dr2 := &discoveryv3.DiscoveryRequest{
		Node:    p.node,
		TypeUrl: types.ClusterUrl,
	}
	dr3 := &discoveryv3.DiscoveryRequest{
		Node:    p.node,
		TypeUrl: types.ClusterLoadAssignmentUrl,
	}

	p.sendCh <- dr1
	p.sendCh <- dr2
	p.sendCh <- dr3
	p.logger.Debugw("sent initial discovery requests for route and cluster")
}

// sendLoop receives pending DiscoveryRequest objects and sends them to client.
// Send operation will be retried continuously until successful or the context is
// cancelled.
func (p *grpcProvisioner) sendLoop(ctx context.Context, client discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesClient) {
	for {
		select {
		case <-ctx.Done():
			return
		case dr := <-p.sendCh:
			p.logger.Debugw("sending discovery request",
				zap.Any("body", dr),
			)
			condFunc := func() (bool, error) {
				if err := client.Send(dr); err != nil {
					p.logger.Errorw("failed to send discovery request",
						zap.Error(err),
						zap.String("config_source", p.configSource),
					)
					return false, nil
				}
				return true, nil
			}
			go func() {
				_ = wait.PollImmediateUntil(time.Second, condFunc, ctx.Done())
			}()
		}
	}
}

// recvLoop receives DiscoveryResponse objects from the wire stream and sends them
// to the recvCh channel.
func (p *grpcProvisioner) recvLoop(ctx context.Context, client discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesClient) {
	var resp *discoveryv3.DiscoveryResponse

	condFunc := func() (bool, error) {
		dr, err := client.Recv()
		if err != nil {
			p.logger.Errorw("failed to receive discovery response",
				zap.Error(err),
			)
			return false, nil
		}
		resp = dr
		return true, nil
	}

	for {
		if wait.PollImmediateUntil(time.Second, condFunc, ctx.Done()) != nil {
			return
		}
		p.logger.Debugw("got discovery response",
			zap.Any("body", resp),
		)
		go func() {
			select {
			case <-ctx.Done():
			case p.recvCh <- resp:
			}
		}()
	}
}

// translateLoop mediates the input DiscoveryResponse objects, translating
// them APISIX resources, and generating an ACK request ultimately.
func (p *grpcProvisioner) translateLoop(ctx context.Context) {
	var verInfo string
	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-p.recvCh:
			ackReq := &discoveryv3.DiscoveryRequest{
				Node:          p.node,
				TypeUrl:       resp.TypeUrl,
				ResponseNonce: resp.Nonce,
			}
			if err := p.translate(resp); err != nil {
				ackReq.ErrorDetail = &status.Status{
					Code:    int32(code.Code_INVALID_ARGUMENT),
					Message: err.Error(),
				}
			} else {
				verInfo = resp.VersionInfo
			}
			ackReq.VersionInfo = verInfo
			p.sendCh <- ackReq
		}
	}
}

func (p *grpcProvisioner) translate(resp *discoveryv3.DiscoveryResponse) error {
	var (
		// Since the type url is fixed, only one field is filled in m and o.
		m      util.Manifest
		o      util.Manifest
		events []types.Event
	)
	// As we use ADS, the TypeUrl field indicates the resource type already.
	switch resp.GetTypeUrl() {
	case types.RouteConfigurationUrl:
		for _, res := range resp.GetResources() {
			partial, err := p.processRouteConfigurationV3(res)
			if err != nil {
				return err
			}
			m.Routes = append(m.Routes, partial...)
		}
		o.Routes = p.routes
		p.routes = m.Routes

	case types.ClusterUrl:
		newUps := make(map[string]*apisix.Upstream)
		for _, res := range resp.GetResources() {
			ups, err := p.processClusterV3(res)
			if err != nil {
				if err == xdsv3.ErrFeatureNotSupportedYet {
					p.logger.Warnw("failed to translate Cluster to APISIX upstreams",
						zap.Error(err),
						zap.Any("cluster", res),
					)
					continue
				} else {
					p.logger.Errorw("failed to translate Cluster to APISIX upstreams",
						zap.Error(err),
						zap.Any("cluster", res),
					)
					return err
				}
			}
			m.Upstreams = append(m.Upstreams, ups)
			newUps[ups.Name] = ups
		}
		// TODO Refactor util.Manifest to just use map.
		for _, ups := range p.upstreams {
			o.Upstreams = append(o.Upstreams, ups)
		}
		p.upstreams = newUps
	case types.ClusterLoadAssignmentUrl:
		for _, res := range resp.GetResources() {
			ups, err := p.processClusterLoadAssignmentV3(res)
			if err != nil {
				return err
			}
			p.upstreams[ups.Name] = ups
			m.Upstreams = append(m.Upstreams, ups)
		}

	default:
		return _errUnknownResourceTypeUrl
	}
	// Always generate update event for EDS.
	if resp.GetTypeUrl() == types.ClusterLoadAssignmentUrl {
		for _, ups := range m.Upstreams {
			events = append(events, types.Event{
				Type:   types.EventUpdate,
				Object: ups,
			})
		}
	} else {
		events = p.generateEvents(&m, &o)
	}
	go func() {
		p.evChan <- events
	}()
	return nil
}

func (p *grpcProvisioner) generateEvents(m, o *util.Manifest) []types.Event {
	p.logger.Debugw("comparing old and new manifests",
		zap.Any("old", o),
		zap.Any("new", m),
	)
	var (
		added   *util.Manifest
		deleted *util.Manifest
		updated *util.Manifest
		count   int
	)
	if o == nil {
		added = m
	} else if m == nil {
		deleted = o
	} else {
		added, deleted, updated = o.DiffFrom(m)
	}
	p.logger.Debugw("found changes (after converting to APISIX resources)",
		zap.Any("added", added),
		zap.Any("deleted", deleted),
		zap.Any("updated", updated),
	)

	if added != nil {
		count += added.Size()
	}
	if deleted != nil {
		count += deleted.Size()
	}
	if updated != nil {
		count += updated.Size()
	}
	if count == 0 {
		return nil
	}
	events := make([]types.Event, 0, count)
	if added != nil {
		events = append(events, added.Events(types.EventAdd)...)
	}
	if deleted != nil {
		events = append(events, deleted.Events(types.EventDelete)...)
	}
	if updated != nil {
		events = append(events, updated.Events(types.EventUpdate)...)
	}
	return events
}

package xds

import (
	"io/ioutil"
	"os"
	"path/filepath"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/ptypes/any"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	apisixutil "github.com/api7/apisix-mesh-agent/pkg/apisix"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

type resourceManifest struct {
	routes    []*apisix.Route
	upstreams []*apisix.Upstream
}

// diffFrom checks the difference between rm and rm2 from rm's point of view.
func (rm *resourceManifest) diffFrom(rm2 *resourceManifest) (*resourceManifest, *resourceManifest, *resourceManifest) {
	var (
		added   resourceManifest
		updated resourceManifest
		deleted resourceManifest
	)

	a, d, u := apisixutil.CompareRoutes(rm.routes, rm2.routes)
	added.routes = append(added.routes, a...)
	updated.routes = append(updated.routes, u...)
	deleted.routes = append(deleted.routes, d...)
	return &added, &deleted, &updated
}

type xdsFileProvisioner struct {
	logger    *log.Logger
	watcher   *fsnotify.Watcher
	evChan    chan []types.Event
	v3Adaptor xdsv3.Adaptor
	files     []string
	state     map[string]*resourceManifest
}

// NewXDSProvisionerFromFiles creates a files backed Provisioner, it watches
// on the given files/directories, files will be parsed into xDS objects,
// invalid items will be ignored but leave with a log.
// Note files watched by this Provisioner should be in the format DiscoveryResponse
// (see https://github.com/envoyproxy/data-plane-api/blob/main/envoy/service/discovery/v3/discovery.proto#L68
// for more details).
// Currently only JSON are suppported as the file type and only xDS V3 are supported.
func NewXDSProvisionerFromFiles(cfg *config.Config) (provisioner.Provisioner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	logger, err := log.NewLogger(
		log.WithContext("xds-file-provisioner"),
		log.WithLogLevel(cfg.LogLevel),
		log.WithOutputFile(cfg.LogOutput),
	)
	if err != nil {
		return nil, err
	}
	adaptor, err := xdsv3.NewAdaptor(cfg)
	if err != nil {
		return nil, err
	}
	p := &xdsFileProvisioner{
		watcher:   watcher,
		logger:    logger,
		v3Adaptor: adaptor,
		evChan:    make(chan []types.Event),
		files:     cfg.XDSWatchFiles,
		state:     make(map[string]*resourceManifest),
	}
	return p, nil
}

func (p *xdsFileProvisioner) Run(stop chan struct{}) error {
	p.logger.Infow("xds file provisioner started")
	defer func() {
		_ = p.logger.Close()
	}()
	defer p.logger.Infow("xds file provisioner exited")
	defer close(p.evChan)

	if err := p.handleInitialFileEvents(); err != nil {
		return err
	}

	for _, file := range p.files {
		if err := p.watcher.Add(file); err != nil {
			return err
		}
	}

	for {
		select {
		case <-stop:
			if err := p.watcher.Close(); err != nil {
				p.logger.Errorw("failed to close watcher",
					zap.Error(err),
				)
			}
			return nil
		case err := <-p.watcher.Errors:
			p.logger.Errorw("detected watch errors",
				zap.Error(err),
			)
		case ev := <-p.watcher.Events:
			if ev.Op != fsnotify.Write && ev.Op != fsnotify.Remove {
				p.logger.Debugw("ignore unnecessary file change event",
					zap.String("filename", ev.Name),
					zap.String("type", ev.Op.String()),
				)
				continue
			} else {
				p.logger.Infow("file change event arrived",
					zap.String("filename", ev.Name),
					zap.String("type", ev.Op.String()),
				)
			}
			p.handleFileEvent(ev)
		}
	}
}

func (p *xdsFileProvisioner) handleInitialFileEvents() error {
	var files []string

	for _, file := range p.files {
		info, err := os.Stat(file)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, file)
		} else {
			err = filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				files = append(files, path)
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	for _, file := range files {
		p.handleFileEvent(fsnotify.Event{
			Name: file,
			Op:   fsnotify.Write,
		})
	}
	return nil
}

func (p *xdsFileProvisioner) Channel() <-chan []types.Event {
	return p.evChan
}

func (p *xdsFileProvisioner) handleFileEvent(ev fsnotify.Event) {
	var (
		events []types.Event
	)
	if ev.Op != fsnotify.Remove {
		data, err := ioutil.ReadFile(ev.Name)
		if err != nil {
			p.logger.Errorw("failed to read file",
				zap.Error(err),
				zap.String("filename", ev.Name),
				zap.String("type", ev.Op.String()),
			)
			return
		}

		var dr discoveryv3.DiscoveryResponse
		if err := protojson.Unmarshal(data, &dr); err != nil {
			p.logger.Errorw("failed to unmarshal file",
				zap.Error(err),
				zap.String("filename", ev.Name),
				zap.String("type", ev.Op.String()),
			)
			return
		}
		events = p.generateEventsFromDiscoveryResponseV3(ev.Name, &dr)
	} else {
		rmo, ok := p.state[ev.Name]
		if ok {
			events = p.generateEvents(ev.Name, rmo, nil)
		}
	}

	// Send events in another goroutine to avoid blocking the watch.
	if len(events) > 0 {
		go func() {
			p.evChan <- events
		}()
	}
}

func (p *xdsFileProvisioner) generateEventsFromDiscoveryResponseV3(filename string, dr *discoveryv3.DiscoveryResponse) []types.Event {
	p.logger.Debugw("parsing discovery response v3",
		zap.Any("content", dr),
	)
	var (
		rm resourceManifest
	)
	for _, res := range dr.GetResources() {
		switch res.GetTypeUrl() {
		case "type.googleapis.com/envoy.config.route.v3.RouteConfiguration":
			rm.routes = append(rm.routes, p.processRouteConfigurationV3(res)...)
		case "type.googleapis.com/envoy.config.cluster.v3.Cluster":
			rm.upstreams = append(rm.upstreams, p.processClusterV3(res)...)
		default:
			p.logger.Warnw("ignore unnecessary resource",
				zap.String("type", res.GetTypeUrl()),
				zap.Any("resource", res),
			)
		}
	}
	rmo := p.state[filename]
	return p.generateEvents(filename, rmo, &rm)
}

func (p *xdsFileProvisioner) generateEvents(filename string, rmo, rm *resourceManifest) []types.Event {
	var (
		added   *resourceManifest
		deleted *resourceManifest
		updated *resourceManifest
	)
	if rmo == nil {
		added = rm
	} else if rm == nil {
		deleted = rmo
	} else {
		added, deleted, updated = rmo.diffFrom(rm)
	}
	p.logger.Debugw("found changes (after converting to APISIX resources) in xds file",
		zap.String("filename", filename),
		zap.Any("added", added),
		zap.Any("deleted", deleted),
		zap.Any("updated", updated),
	)
	p.state[filename] = rm

	var count int
	if added != nil {
		count += len(added.routes)
	}
	if deleted != nil {
		count += len(deleted.routes)
	}
	if updated != nil {
		count += len(updated.routes)
	}
	if count == 0 {
		return nil
	}
	events := make([]types.Event, 0, count)
	if added != nil {
		for _, r := range added.routes {
			events = append(events, types.Event{
				Type:   types.EventAdd,
				Object: r,
			})
		}
	}
	if deleted != nil {
		for _, r := range deleted.routes {
			events = append(events, types.Event{
				Type:      types.EventDelete,
				Tombstone: r,
			})
		}
	}
	if updated != nil {
		for _, r := range updated.routes {
			events = append(events, types.Event{
				Type:   types.EventUpdate,
				Object: r,
			})
		}
	}
	return events
}

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
		p.logger.Warnw("cluster depends on another EDS config, an upstream withou nodes setting was generated",
			zap.Any("upstream", ups),
		)
	}
	return []*apisix.Upstream{ups}
}

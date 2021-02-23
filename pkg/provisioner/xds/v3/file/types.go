package file

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"

	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	xdsv3 "github.com/api7/apisix-mesh-agent/pkg/adaptor/xds/v3"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

type xdsFileProvisioner struct {
	logger                  *log.Logger
	watcher                 *fsnotify.Watcher
	evChan                  chan []types.Event
	v3Adaptor               xdsv3.Adaptor
	files                   []string
	state                   map[string]*manifest
	upstreamCache           map[string]*apisix.Upstream
	updatedUpstreamsFromEDS map[string][]*apisix.Upstream
}

// NewXDSProvisioner creates a files backed Provisioner, it watches
// on the given files/directories, files will be parsed into xDS objects,
// invalid items will be ignored but leave with a log.
// Note files watched by this Provisioner should be in the format DiscoveryResponse
// (see https://github.com/envoyproxy/data-plane-api/blob/main/envoy/service/discovery/v3/discovery.proto#L68
// for more details).
// Currently only JSON are suppported as the file type and only xDS V3 are supported.
func NewXDSProvisioner(cfg *config.Config) (provisioner.Provisioner, error) {
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
		watcher:                 watcher,
		logger:                  logger,
		v3Adaptor:               adaptor,
		evChan:                  make(chan []types.Event),
		files:                   cfg.XDSWatchFiles,
		state:                   make(map[string]*manifest),
		upstreamCache:           make(map[string]*apisix.Upstream),
		updatedUpstreamsFromEDS: make(map[string][]*apisix.Upstream),
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
			switch ev.Op {
			case fsnotify.Create, fsnotify.Write, fsnotify.Remove:
				p.logger.Infow("file change event arrived",
					zap.String("filename", ev.Name),
					zap.String("type", ev.Op.String()),
				)
			default:
				p.logger.Debugw("ignore unnecessary file change event",
					zap.String("filename", ev.Name),
					zap.String("type", ev.Op.String()),
				)
				continue
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
			// Upstreams which nodes are supported by EDS should reset
			// its nodes to nil, the event should be update, not delete.
			for _, ups := range p.updatedUpstreamsFromEDS[ev.Name] {
				// Do not modify the original ups to avoid race conditions.
				newUps := proto.Clone(ups).(*apisix.Upstream)
				newUps.Nodes = nil
				events = append(events, types.Event{
					Type:   types.EventUpdate,
					Object: newUps,
				})
			}
			delete(p.updatedUpstreamsFromEDS, ev.Name)
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
		rm               manifest
		updatedUpstreams []*apisix.Upstream
	)
	for _, res := range dr.GetResources() {
		switch res.GetTypeUrl() {
		case "type.googleapis.com/envoy.config.route.v3.RouteConfiguration":
			rm.Routes = append(rm.Routes, p.processRouteConfigurationV3(res)...)
		case "type.googleapis.com/envoy.config.cluster.v3.Cluster":
			rm.Upstreams = append(rm.Upstreams, p.processClusterV3(res)...)
		case "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":
			var slot int
			ups := p.processClusterLoadAssignmentV3(res)
			for i := 0; i < len(ups); i++ {
				var found bool
				for j := 0; j < len(rm.Upstreams); j++ {
					// EDS should be merged to the CDS if the CDS are in the
					// same DiscoveryResponse.
					if rm.Upstreams[i].Name == ups[i].Name {
						found = true
						rm.Upstreams[i] = ups[i]
						break
					}
					// else the upstreams generated by EDS should be appended.
				}
				if !found {
					ups[slot] = ups[i]
					slot++
				}
			}
			for i := slot; i < len(ups); i++ {
				ups[i] = nil
			}
			ups = ups[:slot]
			updatedUpstreams = append(updatedUpstreams, ups...)
		default:
			p.logger.Warnw("ignore unnecessary resource",
				zap.String("type", res.GetTypeUrl()),
				zap.Any("resource", res),
			)
		}
	}
	evs := p.generateEvents(filename, p.state[filename], &rm)

	if len(updatedUpstreams) > 0 {
		updatedUpstreamsFromEDS := p.updatedUpstreamsFromEDS[filename]
		// These upstreams updated since EDS config change.
		// While EDS config might in different files, we cannot just append them to
		// `rm` or update event will be set to add (since the last state of EDS
		// config file might not in p.state). So here we process them specially.
		for _, ups := range updatedUpstreams {
			evs = append(evs, types.Event{
				Type:   types.EventUpdate,
				Object: ups,
			})
			updatedUpstreamsFromEDS = append(updatedUpstreamsFromEDS, ups)
		}

		p.updatedUpstreamsFromEDS[filename] = updatedUpstreamsFromEDS
		p.logger.Debugw("found upstream changes due to EDS config",
			zap.String("filename", filename),
			zap.Any("upstreams", updatedUpstreams),
		)
	}

	return evs
}

func (p *xdsFileProvisioner) generateEvents(filename string, rmo, rm *manifest) []types.Event {
	var (
		added   *manifest
		deleted *manifest
		updated *manifest
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
		count += added.size()
	}
	if deleted != nil {
		count += deleted.size()
	}
	if updated != nil {
		count += updated.size()
	}
	if count == 0 {
		return nil
	}
	events := make([]types.Event, 0, count)
	if added != nil {
		events = append(events, added.events(types.EventAdd)...)
	}
	if deleted != nil {
		events = append(events, deleted.events(types.EventDelete)...)
	}
	if updated != nil {
		events = append(events, updated.events(types.EventUpdate)...)
	}
	return events
}

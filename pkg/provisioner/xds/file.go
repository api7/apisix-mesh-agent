package xds

import (
	"encoding/json"
	"io/ioutil"

	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/ptypes/any"
	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	"github.com/api7/apisix-mesh-agent/pkg/types"
)

type xdsFileProvisioner struct {
	logger  *log.Logger
	watcher *fsnotify.Watcher
	evChan  chan []types.Event
	files   []string
	state   map[string]*any.Any
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
	p := &xdsFileProvisioner{
		watcher: watcher,
		logger:  logger,
		evChan:  make(chan *types.Event),
		files:   cfg.XDSWatchFiles,
		state:   make(map[string]*any.Any),
	}
	return p, nil
}

func (p *xdsFileProvisioner) Run(stop chan struct{}) error {
	p.logger.Infow("xds file provisioner started")
	defer p.logger.Close()
	defer p.logger.Infow("xds file provisioner exited")
	defer close(p.evChan)

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
			p.logger.Infow("file change event arrived",
				zap.String("filename", ev.Name),
				zap.String("type", ev.Op.String()),
			)
			p.handleFileEvent(ev)
		}
	}
}

func (p *xdsFileProvisioner) Channel() <-chan []types.Event {
	return p.evChan
}

func (p *xdsFileProvisioner) handleFileEvent(ev fsnotify.Event) {
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
		if err := json.Unmarshal(data, &dr); err != nil {
			p.logger.Errorw("failed to unmarshal file",
				zap.Error(err),
				zap.String("filename", ev.Name),
				zap.String("type", ev.Op.String()),
			)
			return
		}
		events, err := p.generateEventsFromDiscoveryResponseV3(ev.Name, &dr)
		if err != nil {
			return
		}

		// Send events in another goroutine to avoid blocking the watch.
		go func() {
			p.evChan <- events
		}()
	}
}

func (p *xdsFileProvisioner) generateEventsFromDiscoveryResponseV3(filename string, dr *discoveryv3.DiscoveryResponse) ([]types.Event, error) {
	p.logger.Debugw("parsing discovery response v3",
		zap.Any("content", dr),
	)
	for _, res := range dr.GetResources() {
		if !ResourceInUse(res.GetTypeUrl()) {
			continue
		}
	}
}

func (p *xdsFileProvisioner) generateDeleteEvents(filename string) []types.Event {

}

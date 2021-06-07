package nacos

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner/util"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

var (
	_ provisioner.Provisioner = (*nacosProvisioner)(nil)

	// ErrUnknownNacosScheme means provided nacos address is not HTTP or HTTPS
	ErrUnknownNacosScheme = errors.New("cannot detect Nacos service port")
)

// TODO: Add e2e test
type nacosProvisioner struct {
	clientConfig  *constant.ClientConfig
	serverConfigs []constant.ServerConfig

	logger *log.Logger

	// nacos config client
	configClient config_client.IConfigClient

	// eventsCh receive changes from client
	eventsCh chan []types.Event
	// mu should be acquire before update routes or upstreams
	mu sync.Mutex
	// last state of routes
	routes []*apisix.Route
	// last state of upstreams
	upstreams []*apisix.Upstream
}

func NewProvisioner(cfg *config.Config) (provisioner.Provisioner, error) {
	clientLogLevel := cfg.LogLevel
	switch clientLogLevel {
	case "dpanic", "panic", "fatal": // valid in zap but invalid in nacos-go
		clientLogLevel = "error"
	default:
		clientLogLevel = "info"
	}

	clientConfig := constant.NewClientConfig(
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithRotateTime("1h"), // Log rotate time
		constant.WithLogLevel(clientLogLevel),
	)

	if !strings.Contains(cfg.NacosSource, "://") {
		cfg.NacosSource = "http://" + cfg.NacosSource
	}
	u, err := url.Parse(cfg.NacosSource)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrUnknownNacosScheme
	}

	var port int
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return nil, err
		}
	} else if u.Scheme == "http" {
		port = 80
	} else if u.Scheme == "https" {
		port = 443
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      u.Hostname(),
			ContextPath: u.Path,
			Port:        uint64(port),
			Scheme:      u.Scheme,
		},
	}

	logger, err := log.NewLogger(
		log.WithOutputFile(cfg.LogOutput),
		log.WithLogLevel(cfg.LogLevel),
		log.WithContext("nacos-provisioner"),
	)
	if err != nil {
		return nil, err
	}

	return &nacosProvisioner{
		clientConfig:  clientConfig,
		serverConfigs: serverConfigs,
		eventsCh:      make(chan []types.Event),
		logger:        logger,
	}, nil
}

func (p *nacosProvisioner) Channel() <-chan []types.Event {
	return p.eventsCh
}

func (p *nacosProvisioner) Run(stop chan struct{}) error {
	defer close(p.eventsCh)

	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  p.clientConfig,
			ServerConfigs: p.serverConfigs,
		},
	)
	if err != nil {
		return err
	}
	p.configClient = configClient

	err = p.fullSync()
	if err != nil {
		return err
	}

	err = p.watch()
	if err != nil {
		return err
	}

	for {
		select {
		case <-stop:
			p.logger.Infof("receive stop signal")
			return nil
		}
	}
}

func (p *nacosProvisioner) fullSync() error {
	// TODO: Support multiple DataId
	routeConf, err := p.configClient.GetConfig(vo.ConfigParam{
		DataId: "cfg.routes",
		Group:  "org.apache.apisix",
	})
	if err != nil {
		return err
	}

	routes, err := routesFromConf(routeConf)
	if err != nil {
		return err
	}
	p.syncRoutes(routes)

	// TODO: Support multiple DataId
	upstreamConf, err := p.configClient.GetConfig(vo.ConfigParam{
		DataId: "cfg.upstreams",
		Group:  "org.apache.apisix",
	})
	if err != nil {
		return err
	}

	upstreams, err := upstreamsFromConf(upstreamConf)
	if err != nil {
		return err
	}
	p.syncUpstreams(upstreams)
	return nil
}

func (p *nacosProvisioner) watch() error {
	// TODO: Support multiple DataId
	err := p.configClient.ListenConfig(vo.ConfigParam{
		DataId:   "cfg.routes",
		Group:    "org.apache.apisix",
		OnChange: p.routeOnChange,
	})
	if err != nil {
		return err
	}

	// TODO: Support multiple DataId
	err = p.configClient.ListenConfig(vo.ConfigParam{
		DataId:   "cfg.upstreams",
		Group:    "org.apache.apisix",
		OnChange: p.upstreamOnChange,
	})
	return err
}

func (p *nacosProvisioner) syncRoutes(routes []*apisix.Route) {
	var (
		oldM util.Manifest
		newM util.Manifest
	)
	p.mu.Lock()
	oldM.Routes = p.routes
	newM.Routes = routes
	p.routes = routes
	p.syncManifest(&oldM, &newM)
	// unlock after sync finished to keep events order
	p.mu.Unlock()
}

func (p *nacosProvisioner) syncUpstreams(upstreams []*apisix.Upstream) {
	var (
		oldM util.Manifest
		newM util.Manifest
	)
	p.mu.Lock()
	oldM.Upstreams = p.upstreams
	newM.Upstreams = upstreams
	p.upstreams = upstreams
	p.syncManifest(&oldM, &newM)
	// unlock after sync finished to keep events order
	p.mu.Unlock()
}

// syncManifest assumes that it's called in the order of events, which means mu should be acquired outside
func (p *nacosProvisioner) syncManifest(old, new *util.Manifest) {
	added, deleted, updated := old.DiffFrom(new)
	p.logger.Infow("sync",
		zap.Any("added", added),
		zap.Any("deleted", deleted),
		zap.Any("updated", updated),
	)

	var count int
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
		return
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
	p.eventsCh <- events
}

func (p *nacosProvisioner) routeOnChange(namespace, group, dataId, data string) {
	p.logger.Debugf("route change: %v/%v %v: %v\n", namespace, group, dataId, data)
	routes, err := routesFromConf(data)
	if err != nil {
		p.logger.Errorw("unmarshal routes failed",
			zap.Error(err),
		)
	}
	p.logger.Debugf("got routes: %v\n", routes)
	p.syncRoutes(routes)
}

func (p *nacosProvisioner) upstreamOnChange(namespace, group, dataId, data string) {
	p.logger.Debugf("upstream change: %v/%v %v: %v\n", namespace, group, dataId, data)
	upstreams, err := upstreamsFromConf(data)
	if err != nil {
		p.logger.Errorw("unmarshal upstreams failed",
			zap.Error(err),
		)
	}
	p.logger.Debugf("got upstreams: %v\n", upstreams)
	p.syncUpstreams(upstreams)
}

func routesFromConf(data string) ([]*apisix.Route, error) {
	var routes []*apisix.Route
	if len(data) == 0 {
		return routes, nil
	}
	// TODO: Should support `host`, `uri`, `upstream`
	// See https://github.com/apache/apisix/blob/master/apisix/schema_def.lua
	err := json.Unmarshal([]byte(data), &routes)
	if err != nil {
		return nil, err
	}
	return routes, nil
}

func upstreamsFromConf(data string) ([]*apisix.Upstream, error) {
	var upstreams []*apisix.Upstream
	if len(data) == 0 {
		return upstreams, nil
	}
	// TODO: Should support hashmap nodes
	// See https://github.com/apache/apisix/blob/master/apisix/schema_def.lua
	err := json.Unmarshal([]byte(data), &upstreams)
	if err != nil {
		return nil, err
	}
	return upstreams, nil
}

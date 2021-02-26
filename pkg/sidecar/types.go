package sidecar

import (
	"net"
	"sync/atomic"

	"github.com/api7/apisix-mesh-agent/pkg/etcdv3"

	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	xdsv3file "github.com/api7/apisix-mesh-agent/pkg/provisioner/xds/v3/file"
	"github.com/api7/apisix-mesh-agent/pkg/types"
)

// Sidecar is the entity to joint provisioner, cache, etcd and launch
// the program.
type Sidecar struct {
	logger       *log.Logger
	provisioner  provisioner.Provisioner
	cache        cache.Cache
	grpcListener net.Listener
	etcdSrv      etcdv3.EtcdV3
	revision     int64
}

// NewSidecar creates a Sidecar object.
func NewSidecar(cfg *config.Config) (*Sidecar, error) {
	p, err := newProvisioner(cfg)
	if err != nil {
		return nil, err
	}
	logger, err := log.NewLogger(
		log.WithContext("sidecar"),
		log.WithLogLevel(cfg.LogLevel),
		log.WithOutputFile(cfg.LogOutput),
	)
	if err != nil {
		return nil, err
	}

	li, err := net.Listen("tcp", cfg.GRPCListen)
	if err != nil {
		return nil, err
	}
	s := &Sidecar{
		grpcListener: li,
		logger:       logger,
		provisioner:  p,
		cache:        cache.NewInMemoryCache(),
	}
	etcd, err := etcdv3.NewEtcdV3Server(cfg, s.cache, s)
	if err != nil {
		return nil, err
	}
	s.etcdSrv = etcd
	return s, nil
}

// Run runs the sidecar program.
func (s *Sidecar) Run(stop chan struct{}) error {
	s.logger.Info("sidecar started")
	defer s.logger.Info("sidecar exited")

	go func() {
		if err := s.provisioner.Run(stop); err != nil {
			s.logger.Fatalw("provisioner run failed",
				zap.Error(err),
			)
		}
	}()

	go func() {
		if err := s.etcdSrv.Serve(s.grpcListener); err != nil {
			s.logger.Fatalw("etcd v3 server run failed",
				zap.Error(err),
			)
		}
	}()

loop:
	for {
		events, ok := <-s.provisioner.Channel()
		if !ok {
			break loop
		}
		s.reflectToLog(events)
		s.reflectToCache(events)
		// sidecar goroutine doesn't need to watch on stop channel,
		// since it can receive the quit signal from the provisioner.
	}

	return nil
}

func (s *Sidecar) reflectToLog(events []types.Event) {
	s.logger.Debugw("events arrived from provisioner",
		zap.Any("events", events),
	)
}

// Revision implements etcdv3.Revisioner.
func (s *Sidecar) Revision() int64 {
	return atomic.LoadInt64(&s.revision)
}

func newProvisioner(cfg *config.Config) (provisioner.Provisioner, error) {
	switch cfg.Provisioner {
	case config.XDSV3FileProvisioner:
		return xdsv3file.NewXDSProvisioner(cfg)
	default:
		return nil, config.ErrUnknownProvisioner
	}
}

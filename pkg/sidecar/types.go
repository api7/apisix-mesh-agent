package sidecar

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/etcdv3"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	xdsv3file "github.com/api7/apisix-mesh-agent/pkg/provisioner/xds/v3/file"
	xdsv3grpc "github.com/api7/apisix-mesh-agent/pkg/provisioner/xds/v3/grpc"
	"github.com/api7/apisix-mesh-agent/pkg/types"
)

// Sidecar is the entity to joint provisioner, cache, etcd and launch
// the program.
type Sidecar struct {
	runId        string
	logger       *log.Logger
	provisioner  provisioner.Provisioner
	cache        cache.Cache
	grpcListener net.Listener
	etcdSrv      etcdv3.EtcdV3
	revision     int64
	apisixRunner *apisixRunner
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

	var ar *apisixRunner
	if cfg.RunMode == config.BundleMode {
		ar = &apisixRunner{
			home:   cfg.APISIXHomePath,
			bin:    cfg.APISIXBinPath,
			done:   make(chan struct{}),
			logger: logger,
		}
	}
	s := &Sidecar{
		runId:        cfg.RunId,
		grpcListener: li,
		logger:       logger,
		provisioner:  p,
		cache:        cache.NewInMemoryCache(),
		apisixRunner: ar,
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
	s.logger.Infow("sidecar started",
		zap.String("id", s.runId),
	)
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
	time.Sleep(time.Second)

	defer func() {
		shutCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		if err := s.etcdSrv.Shutdown(shutCtx); err != nil {
			s.logger.Errorw("failed to shutdown etcd server",
				zap.Error(err),
			)
		}
		cancel()
	}()

	if s.apisixRunner != nil {
		// Launch Apache APISIX after the main logic of apisix-mesh-agent was started,
		// so that once APISIX started, it can fetch configuration from apisix-mesh-agent.
		if err := s.apisixRunner.run(stop); err != nil {
			return err
		}
	}

loop:
	for {
		events, ok := <-s.provisioner.Channel()
		if !ok {
			break loop
		}
		s.reflectToLog(events)
		// TODO may reflect to etcd after cache one by one.
		s.reflectToCache(events)
		s.reflectToEtcd(events)
		// sidecar goroutine doesn't need to watch on stop channel,
		// since it can receive the quit signal from the provisioner.
	}

	if s.apisixRunner != nil {
		s.apisixRunner.shutdown()
	}

	return nil
}

func (s *Sidecar) reflectToLog(events []types.Event) {
	s.logger.Debugw("events arrived from provisioner",
		zap.Any("events", events),
	)
}

func (s *Sidecar) reflectToEtcd(events []types.Event) {
	go func(events []types.Event) {
		s.etcdSrv.PushEvents(events)
	}(events)
}

// Revision implements etcdv3.Revisioner.
func (s *Sidecar) Revision() int64 {
	return atomic.LoadInt64(&s.revision)
}

func newProvisioner(cfg *config.Config) (provisioner.Provisioner, error) {
	switch cfg.Provisioner {
	case config.XDSV3FileProvisioner:
		return xdsv3file.NewXDSProvisioner(cfg)
	case config.XDSV3GRPCProvisioner:
		return xdsv3grpc.NewXDSProvisioner(cfg)
	default:
		return nil, config.ErrUnknownProvisioner
	}
}

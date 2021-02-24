package sidecar

import (
	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
	xdsv3file "github.com/api7/apisix-mesh-agent/pkg/provisioner/xds/v3/file"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"go.uber.org/zap"
)

// Sidecar is the entity to joint provisioner, cache, etcd and launch
// the program.
type Sidecar struct {
	logger      *log.Logger
	provisioner provisioner.Provisioner
	cache       cache.Cache
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

	return &Sidecar{
		logger:      logger,
		provisioner: p,
		cache:       cache.NewInMemoryCache(),
	}, nil
}

// Run runs the sidecar program.
func (s *Sidecar) Run(stop chan struct{}) error {
	s.logger.Info("sidecar started")
	defer s.logger.Info("sidecar exited")

	go func() {
		if err := s.provisioner.Run(stop); err != nil {
			s.logger.Errorw("provisioner run failed",
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

func newProvisioner(cfg *config.Config) (provisioner.Provisioner, error) {
	switch cfg.Provisioner {
	case config.XDSV3FileProvisioner:
		return xdsv3file.NewXDSProvisioner(cfg)
	default:
		return nil, config.ErrUnknownProvisioner
	}
}

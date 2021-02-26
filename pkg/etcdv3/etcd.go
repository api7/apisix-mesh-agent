package etcdv3

import (
	"net"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/etcdserverpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
)

// EtcdV3 abstracts the behaviors of the mimicking ETCD v3 server.
type EtcdV3 interface {
	// Serve accepts a listener and launches the ETCD v3 server.
	Serve(net.Listener) error
	// Shutdown closes the ETCD v3 server.
	Shutdown() error
}

// Revisioner defines how to get the current revision.
type Revisioner interface {
	// Revision returns the current revision.
	Revision() int64
}

type etcdV3 struct {
	metaMu     sync.RWMutex
	metaCache  map[string]meta
	revisioner Revisioner
	keyPrefix  string
	logger     *log.Logger
	cache      cache.Cache
	grpcSrv    *grpc.Server
}

type meta struct {
	createRevision int64
	modRevision    int64
}

// NewEtcdV3Server creates the ETCD v3 server.
func NewEtcdV3Server(cfg *config.Config, cache cache.Cache, revisioner Revisioner) (EtcdV3, error) {
	logger, err := log.NewLogger(
		log.WithLogLevel(cfg.LogLevel),
		log.WithOutputFile(cfg.LogOutput),
		log.WithContext("etcdv3"),
	)
	if err != nil {
		return nil, err
	}
	return &etcdV3{
		revisioner: revisioner,
		cache:      cache,
		logger:     logger,
		keyPrefix:  cfg.EtcdKeyPrefix,
		metaCache:  make(map[string]meta),
	}, nil
}

func (e *etcdV3) Serve(listener net.Listener) error {
	kep := keepalive.EnforcementPolicy{
		MinTime: 15 * time.Second,
	}
	kp := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Timeout:           10 * time.Second,
	}

	grpcSrv := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kep),
		grpc.KeepaliveParams(kp),
	)
	e.grpcSrv = grpcSrv
	etcdserverpb.RegisterKVServer(grpcSrv, e)

	if err := grpcSrv.Serve(listener); err != nil {
		return err
	}
	return nil
}

func (e *etcdV3) Shutdown() error {
	e.grpcSrv.GracefulStop()
	return nil
}

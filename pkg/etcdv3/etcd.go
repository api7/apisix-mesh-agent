package etcdv3

import (
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"google.golang.org/protobuf/proto"

	"go.etcd.io/etcd/api/v3/mvccpb"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"

	"github.com/api7/apisix-mesh-agent/pkg/types"

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
	PushEvent(*types.Event)
}

// Revisioner defines how to get the current revision.
type Revisioner interface {
	// Revision returns the current revision.
	Revision() int64
}

type etcdV3 struct {
	// TODO metadata should be embedded into cache.
	metaMu      sync.RWMutex
	metaCache   map[string]meta
	revisioner  Revisioner
	keyPrefix   string
	logger      *log.Logger
	cache       cache.Cache
	grpcSrv     *grpc.Server
	watcherMu   sync.RWMutex
	nextWatchId int64
	watchers    map[int64]*watchStream
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
		watchers:   make(map[int64]*watchStream),
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
	etcdserverpb.RegisterWatchServer(grpcSrv, e)

	if err := grpcSrv.Serve(listener); err != nil {
		return err
	}
	return nil
}

func (e *etcdV3) Shutdown() error {
	e.grpcSrv.GracefulStop()
	return nil
}

func (e *etcdV3) PushEvent(ev *types.Event) {
	var (
		obj    interface{}
		name   string
		evType mvccpb.Event_EventType
	)
	if ev.Type == types.EventDelete {
		obj = ev.Tombstone
		evType = mvccpb.DELETE
	} else {
		obj = ev.Object
		evType = mvccpb.PUT
	}

	switch o := obj.(type) {
	case *apisix.Route:
		name = e.keyPrefix + "/routes/" + o.Id
	case *apisix.Upstream:
		name = e.keyPrefix + "/upstreams/" + o.Id
	default:
		// ignore other resources for now.
		return
	}
	e.metaMu.RLock()
	m, ok := e.metaCache[name]
	e.metaMu.RUnlock()
	rev := e.revisioner.Revision()
	m.modRevision = rev
	if !ok {
		m.createRevision = rev
	}
	value, err := _pbjsonMarshalOpts.Marshal(obj.(proto.Message))
	if err != nil {
		e.logger.Errorw("protojson marshal error",
			zap.Error(err),
			zap.Any("resource", obj),
		)
		return
	}
	event := &mvccpb.Event{
		Type: evType,
		Kv: &mvccpb.KeyValue{
			Key:            []byte(name),
			CreateRevision: m.createRevision,
			ModRevision:    m.modRevision,
			Value:          value,
		},
	}
	e.metaMu.Lock()
	if ev.Type == types.EventDelete {
		delete(e.metaCache, name)
	} else {
		e.metaCache[name] = m
	}
	e.metaMu.Unlock()

	e.watcherMu.RLock()
	for _, ws := range e.watchers {
		ws.mu.RLock()
		var resps []*etcdserverpb.WatchResponse
		switch obj.(type) {
		case *apisix.Route:
			for id := range ws.route {
				resps = append(resps, &etcdserverpb.WatchResponse{
					Header: &etcdserverpb.ResponseHeader{
						Revision: e.revisioner.Revision(),
					},
					WatchId: id,
					Created: true,
					Events: []*mvccpb.Event{
						event,
					},
				})
			}
		case *apisix.Upstream:
			for id := range ws.upstream {
				resps = append(resps, &etcdserverpb.WatchResponse{
					Header: &etcdserverpb.ResponseHeader{
						Revision: e.revisioner.Revision(),
					},
					WatchId: id,
					Created: true,
					Events: []*mvccpb.Event{
						event,
					},
				})
			}
		}
		ws.mu.RUnlock()
		go func(ws *watchStream) {
			for _, resp := range resps {
				select {
				case ws.eventCh <- resp:
				case <-ws.ctx.Done():
					// Must watch on the ctx.Done() or this goroutine might be leaky.
					return
				}
			}
		}(ws)
	}
	e.watcherMu.RUnlock()
}

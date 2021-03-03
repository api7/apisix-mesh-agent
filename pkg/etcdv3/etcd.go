package etcdv3

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/soheilhy/cmux"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/proto"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

// EtcdV3 abstracts the behaviors of the mimicking ETCD v3 server.
type EtcdV3 interface {
	// Serve accepts a listener and launches the ETCD v3 server.
	Serve(net.Listener) error
	// Shutdown closes the ETCD v3 server.
	Shutdown(context.Context) error
	// PushEvents accepts a bunch of events and converts them to ETCD events,
	// then sending to watch clients.
	PushEvents([]types.Event)
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
	httpSrv     *http.Server
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
	m := cmux.New(listener)
	grpcl := m.Match(cmux.HTTP2())
	httpl := m.Match(cmux.HTTP1Fast())

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

	mux := http.NewServeMux()
	mux.HandleFunc("/version", e.version)
	e.httpSrv = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := e.httpSrv.Serve(httpl); err != nil && !strings.Contains(err.Error(), "mux: listener closed") {
			e.logger.Errorw("http server serve failure",
				zap.Error(err),
			)
		}
	}()

	go func() {
		if err := grpcSrv.Serve(grpcl); err != nil {
			e.logger.Errorw("grpc server serve failure",
				zap.Error(err),
			)
		}
	}()

	if err := m.Serve(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		return err
	}

	return nil
}

func (e *etcdV3) version(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"etcdserver":"3.5.0-pre","etcdcluster":"3.5.0"}`))
	if err != nil {
		e.logger.Warnw("failed to send version info",
			zap.Error(err),
		)
	}
}

func (e *etcdV3) Shutdown(ctx context.Context) error {
	e.grpcSrv.GracefulStop()
	if err := e.httpSrv.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func (e *etcdV3) PushEvents(events []types.Event) {
	for _, ev := range events {
		e.pushEvent(&ev)
	}
}

func (e *etcdV3) pushEvent(ev *types.Event) {
	e.logger.Debugw("receive event",
		zap.Any("event", ev),
	)
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

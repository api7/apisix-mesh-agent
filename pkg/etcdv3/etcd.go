package etcdv3

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/soheilhy/cmux"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	etcdservergw "go.etcd.io/etcd/api/v3/etcdserverpb/gw"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

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
	ctx context.Context
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
	// This context is used to notify the gateway grpc conn should be closed.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.ctx = ctx

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
	if gwmux, err := e.registerGateway(listener.Addr().String()); err != nil {
		e.logger.Errorw("failed to register gateway",
			zap.Error(err),
		)
		return err
	} else {
		mux := http.NewServeMux()
		mux.Handle(
			"/v3/",
			wsproxy.WebsocketProxy(
				gwmux,
				wsproxy.WithRequestMutator(
					func(incoming *http.Request, outgoing *http.Request) *http.Request {
						outgoing.Method = "POST"
						return outgoing
					},
				),
			),
		)
		mux.HandleFunc("/version", e.version)
		e.httpSrv = &http.Server{
			Handler: mux,
		}
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

	if err := m.Serve(); err != nil && !reasonableFailure(err) {
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
	m.modRevision = ev.Revision
	if !ok {
		m.createRevision = ev.Revision
	}
	value, err := json.Marshal(obj)
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
				resp := &etcdserverpb.WatchResponse{
					Header: &etcdserverpb.ResponseHeader{
						Revision: ev.Revision,
					},
					WatchId: id,
					Events: []*mvccpb.Event{
						event,
					},
				}
				resps = append(resps, resp)
				ws.etcd.logger.Debugw("push to client",
					zap.Any("watch_id", resp.WatchId),
					zap.Any("revision", resp.Header.Revision),
					zap.Any("resource", "route"),
					zap.Any("events", event),
				)
			}
		case *apisix.Upstream:
			for id := range ws.upstream {
				resp := &etcdserverpb.WatchResponse{
					Header: &etcdserverpb.ResponseHeader{
						Revision: ev.Revision,
					},
					WatchId: id,
					Events: []*mvccpb.Event{
						event,
					},
				}
				resps = append(resps, resp)
				ws.etcd.logger.Debugw("push to client",
					zap.Any("watch_id", resp.WatchId),
					zap.Any("revision", resp.Header.Revision),
					zap.Any("resource", "upstream"),
					zap.Any("events", event),
				)
			}
		}
		ws.mu.RUnlock()
		// Must be non-blocking to release e.watcherMu, because once ws.ctx is done,
		// e.watcherMu will be acquired to execute watchers cleanup
		go func(ws *watchStream) {
			for _, resp := range resps {
				select {
				case ws.eventCh <- resp:
				case <-ws.ctx.Done():
					ws.etcd.logger.Debugw("context done, etcd push aborted",
						zap.Any("revision", resp.Header.Revision),
						zap.Any("resp", resp),
						zap.Any("watch_id", resp.WatchId),
					)
					// Must watch on the ctx.Done() or this goroutine might be leaky.
					return
				}
			}
		}(ws)
	}
	e.watcherMu.RUnlock()
}

func (e *etcdV3) registerGateway(addr string) (*gatewayruntime.ServeMux, error) {
	e.logger.Infow("registering grpc gateway")
	grpcConn, err := grpc.DialContext(e.ctx, addr,
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	gwmux := gatewayruntime.NewServeMux()
	if err := etcdservergw.RegisterKVHandler(e.ctx, gwmux, grpcConn); err != nil {
		return nil, err
	}
	if err := etcdservergw.RegisterWatchHandler(e.ctx, gwmux, grpcConn); err != nil {
		return nil, err
	}
	go func() {
		<-e.ctx.Done()
		if err := grpcConn.Close(); err != nil {
			e.logger.Warnw("failed to close local gateway grpc conn: ",
				zap.Error(err),
			)
		}
	}()
	return gwmux, nil
}

func reasonableFailure(err error) bool {
	if err == http.ErrServerClosed {
		return true
	}
	if strings.Contains(err.Error(), "mux: listener closed") {
		return true
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

package etcdv3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.uber.org/zap"
)

var (
	_sourceMu sync.Mutex
	_source   rand.Source
)

func init() {
	_source = rand.NewSource(int64(time.Now().Nanosecond()))
}

func randInt64() int64 {
	_sourceMu.Lock()
	defer _sourceMu.Unlock()
	return _source.Int63()
}

type watchStream struct {
	id       int64
	ctx      context.Context
	etcd     *etcdV3
	stream   etcdserverpb.Watch_WatchServer
	mu       sync.RWMutex
	route    map[int64]struct{}
	upstream map[int64]struct{}
	eventCh  chan *etcdserverpb.WatchResponse
}

func (ws *watchStream) cancelWatch(id int64) bool {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if _, ok := ws.route[id]; ok {
		delete(ws.route, id)
		return true
	}
	if _, ok := ws.upstream[id]; ok {
		delete(ws.upstream, id)
		return true
	}
	return false
}

func (ws *watchStream) createWatch(id int64, resource string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if resource == "route" {
		if _, ok := ws.route[id]; ok {
			return errors.New("duplicated watch id on stream")
		}
		ws.route[id] = struct{}{}
	} else if resource == "upstream" {
		if _, ok := ws.upstream[id]; ok {
			return errors.New("duplicated watch id on stream")
		}
		ws.upstream[id] = struct{}{}
	}
	return nil
}

func (ws *watchStream) firstWatch(id int64, resource string, minRev int64) error {
	var (
		kvs []*mvccpb.KeyValue
		err error
	)
	if resource == "route" {
		kvs, err = ws.findAllRoutes(minRev)
	} else if resource == "upstream" {
		kvs, err = ws.findAllUpstreams(minRev)
	}
	if err != nil {
		return err
	}
	if len(kvs) == 0 {
		return nil
	}
	evs := make([]*mvccpb.Event, 0, len(kvs))
	for _, kv := range kvs {
		evs = append(evs, &mvccpb.Event{
			Type: mvccpb.PUT,
			Kv:   kv,
		})
	}
	resp := &etcdserverpb.WatchResponse{
		Header: &etcdserverpb.ResponseHeader{
			Revision: ws.etcd.revisioner.Revision(),
		},
		WatchId: id,
		Created: true,
		Events:  evs,
	}
	if err := ws.stream.Send(resp); err != nil {
		return err
	}
	return nil
}

func (ws *watchStream) findAllRoutes(minRev int64) ([]*mvccpb.KeyValue, error) {
	routes, err := ws.etcd.cache.Route().List()
	if err != nil {
		ws.etcd.logger.Errorw("failed to list routes",
			zap.Error(err),
		)
		return nil, _errInternalError
	}
	var kvs []*mvccpb.KeyValue
	for _, r := range routes {
		key := ws.etcd.keyPrefix + "/routes/" + r.Id
		ws.etcd.metaMu.RLock()
		m, ok := ws.etcd.metaCache[r.Name]
		ws.etcd.metaMu.RUnlock()
		if !ok {
			continue
		}
		if m.modRevision >= minRev {
			value, err := _pbjsonMarshalOpts.Marshal(r)
			if err != nil {
				ws.etcd.logger.Errorw("protojson marshal failure",
					zap.Error(err),
					zap.Any("route", r),
				)
				return nil, err
			}
			kvs = append(kvs, &mvccpb.KeyValue{
				Key:            []byte(key),
				CreateRevision: m.createRevision,
				ModRevision:    m.modRevision,
				Value:          value,
			})
		}
	}
	return kvs, nil
}

func (ws *watchStream) findAllUpstreams(minRev int64) ([]*mvccpb.KeyValue, error) {
	upstreams, err := ws.etcd.cache.Upstream().List()
	if err != nil {
		ws.etcd.logger.Errorw("failed to list upstreams",
			zap.Error(err),
		)
		return nil, _errInternalError
	}
	var kvs []*mvccpb.KeyValue
	for _, u := range upstreams {
		key := ws.etcd.keyPrefix + "/upstreams/" + u.Id
		ws.etcd.metaMu.RLock()
		m, ok := ws.etcd.metaCache[u.Name]
		ws.etcd.metaMu.RUnlock()
		if !ok {
			continue
		}
		if m.modRevision >= minRev {
			value, err := _pbjsonMarshalOpts.Marshal(u)
			if err != nil {
				ws.etcd.logger.Errorw("protojson marshal failure",
					zap.Error(err),
					zap.Any("upstream", u),
				)
				return nil, err
			}
			kvs = append(kvs, &mvccpb.KeyValue{
				Key:            []byte(key),
				CreateRevision: m.createRevision,
				ModRevision:    m.modRevision,
				Value:          value,
			})
		}
	}
	return kvs, nil
}

func (e *etcdV3) Watch(stream etcdserverpb.Watch_WatchServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	ws := &watchStream{
		stream:   stream,
		route:    make(map[int64]struct{}),
		upstream: make(map[int64]struct{}),
		etcd:     e,
		eventCh:  make(chan *etcdserverpb.WatchResponse),
		ctx:      ctx,
	}
	e.watcherMu.Lock()
	id := e.nextWatchId
	e.nextWatchId++
	ws.id = id
	e.watchers[id] = ws
	e.watcherMu.Unlock()

	defer func() {
		e.watcherMu.Lock()
		delete(e.watchers, id)
		e.watcherMu.Unlock()
		cancel()
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := ws.onWire(); err != nil {
			errCh <- err
		}
	}()

	for {
		select {
		case resp := <-ws.eventCh:
			if err := ws.stream.Send(resp); err != nil {
				ws.etcd.logger.Warnw("failed to send WatchResponse",
					zap.Any("watch_response", resp),
				)
				return err
			}
		case werr := <-errCh:
			return werr
		}
	}
}

func (ws *watchStream) onWire() error {
	for {
		req, err := ws.stream.Recv()
		if err != io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		ws.etcd.logger.Debugw("got watch request",
			zap.Any("body", req),
		)
		if err = ws.etcd.checkWatchRequestConformance(req); err != nil {
			if err == rpctypes.ErrKeyNotFound {
				// Silenced other unsupported keys.
				continue
			}
			return err
		}

		switch uv := req.RequestUnion.(type) {
		case *etcdserverpb.WatchRequest_CreateRequest:
			if uv.CreateRequest == nil {
				continue
			}
			var (
				resource string
				id       int64
			)
			if string(uv.CreateRequest.Key) == ws.etcd.keyPrefix+"/routes" {
				resource = "route"
			} else if string(uv.CreateRequest.Key) == ws.etcd.keyPrefix+"/upstreams" {
				resource = "upstream"
			} // others are not concerned
			if uv.CreateRequest.WatchId == 0 {
				id = randInt64()
			} else {
				id = uv.CreateRequest.WatchId
			}
			if err := ws.createWatch(id, resource); err != nil {
				return err
			}
			if uv.CreateRequest.StartRevision != 0 {
				if err := ws.firstWatch(id, resource, uv.CreateRequest.StartRevision); err != nil {
					return err
				}
			}

		case *etcdserverpb.WatchRequest_CancelRequest:
			if uv.CancelRequest != nil {
				if !ws.cancelWatch(uv.CancelRequest.WatchId) {
					return fmt.Errorf("unknown watch id <%d>", uv.CancelRequest.WatchId)
				}
				err = ws.stream.Send(&etcdserverpb.WatchResponse{
					Header: &etcdserverpb.ResponseHeader{
						Revision: ws.etcd.revisioner.Revision(),
					},
					WatchId:  uv.CancelRequest.WatchId,
					Canceled: true,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

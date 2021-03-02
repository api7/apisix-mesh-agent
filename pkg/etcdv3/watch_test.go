package etcdv3

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestCreateAndCancelWatch(t *testing.T) {
	ws := &watchStream{
		route:    make(map[int64]struct{}),
		upstream: make(map[int64]struct{}),
	}
	assert.Nil(t, ws.createWatch(1, "route"))
	assert.Nil(t, ws.createWatch(2, "upstream"))
	assert.Equal(t, ws.createWatch(1, "route"), _errDuplicatedWatchId)
	assert.Equal(t, ws.createWatch(2, "upstream"), _errDuplicatedWatchId)

	assert.Equal(t, ws.cancelWatch(1), true)
	assert.Equal(t, ws.cancelWatch(1), false)
	assert.Equal(t, ws.cancelWatch(2), true)
	assert.Equal(t, ws.cancelWatch(2), false)
}

func TestFindAllRoutes(t *testing.T) {
	f := &fakeRevisioner{rev: 1}
	cfg := &config.Config{
		LogLevel:      "debug",
		LogOutput:     "stderr",
		EtcdKeyPrefix: "/apisix",
	}
	c := cache.NewInMemoryCache()
	err := c.Route().Insert(&apisix.Route{
		Name: "/apisix/routes/01",
		Id:   "01",
	})
	assert.Nil(t, err)
	err = c.Route().Insert(&apisix.Route{
		Name: "/apisix/routes/02",
		Id:   "02",
	})
	assert.Nil(t, err)
	err = c.Route().Insert(&apisix.Route{
		Name: "/apisix/routes/03",
		Id:   "03",
	})
	assert.Nil(t, err)

	etcd, err := NewEtcdV3Server(cfg, c, f)
	assert.Nil(t, err)

	ws := &watchStream{
		etcd:     etcd.(*etcdV3),
		route:    make(map[int64]struct{}),
		upstream: make(map[int64]struct{}),
	}
	ws.etcd.metaCache = map[string]meta{
		"/apisix/routes/01": {
			createRevision: 3,
			modRevision:    5,
		},
		"/apisix/routes/02": {
			createRevision: 1,
			modRevision:    1,
		},
		"/apisix/routes/03": {
			createRevision: 4,
			modRevision:    8,
		},
	}
	kvs, err := ws.findAllRoutes(0)
	assert.Nil(t, err)
	assert.Len(t, kvs, 3)

	kvs, err = ws.findAllRoutes(4)
	assert.Nil(t, err)
	assert.Len(t, kvs, 2)

	kvs, err = ws.findAllRoutes(11)
	assert.Nil(t, err)
	assert.Len(t, kvs, 0)
}

func TestFindAllUpstreams(t *testing.T) {
	f := &fakeRevisioner{rev: 1}
	cfg := &config.Config{
		LogLevel:      "debug",
		LogOutput:     "stderr",
		EtcdKeyPrefix: "/apisix",
	}
	c := cache.NewInMemoryCache()
	err := c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/01",
		Id:   "01",
	})
	assert.Nil(t, err)
	err = c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/02",
		Id:   "02",
	})
	assert.Nil(t, err)
	err = c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/03",
		Id:   "03",
	})
	assert.Nil(t, err)

	etcd, err := NewEtcdV3Server(cfg, c, f)
	assert.Nil(t, err)

	ws := &watchStream{
		etcd:     etcd.(*etcdV3),
		route:    make(map[int64]struct{}),
		upstream: make(map[int64]struct{}),
	}
	ws.etcd.metaCache = map[string]meta{
		"/apisix/upstreams/01": {
			createRevision: 3,
			modRevision:    5,
		},
		"/apisix/upstreams/02": {
			createRevision: 1,
			modRevision:    1,
		},
		"/apisix/upstreams/03": {
			createRevision: 4,
			modRevision:    8,
		},
	}
	kvs, err := ws.findAllUpstreams(0)
	assert.Nil(t, err)
	assert.Len(t, kvs, 3)

	kvs, err = ws.findAllUpstreams(4)
	assert.Nil(t, err)
	assert.Len(t, kvs, 2)

	kvs, err = ws.findAllUpstreams(11)
	assert.Nil(t, err)
	assert.Len(t, kvs, 0)
}

type fakeWatchServer struct {
	grpc.ServerStream

	ctx    context.Context
	reqCh  chan *etcdserverpb.WatchRequest
	respCh chan *etcdserverpb.WatchResponse
}

var _ etcdserverpb.Watch_WatchServer = &fakeWatchServer{}

func (f *fakeWatchServer) SetHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeWatchServer) SendHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeWatchServer) SendTrailer(_ metadata.MD) error {
	return nil
}

func (f *fakeWatchServer) SendMsg(_ interface{}) error {
	return nil
}

func (f *fakeWatchServer) RecvMsg(_ interface{}) error {
	return nil
}

func (f *fakeWatchServer) Context() context.Context {
	return f.ctx
}

func (f *fakeWatchServer) Send(resp *etcdserverpb.WatchResponse) error {
	select {
	case f.respCh <- resp:
		return nil
	case <-time.After(time.Second):
		return errors.New("timed out")
	}
}

func (f *fakeWatchServer) Recv() (*etcdserverpb.WatchRequest, error) {
	select {
	case req := <-f.reqCh:
		return req, nil
	case <-f.ctx.Done():
		return nil, io.EOF
	}
}

func TestFirstWatch(t *testing.T) {
	fr := &fakeRevisioner{rev: 1}
	cfg := &config.Config{
		LogLevel:      "debug",
		LogOutput:     "stderr",
		EtcdKeyPrefix: "/apisix",
	}
	c := cache.NewInMemoryCache()
	err := c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/01",
		Id:   "01",
	})
	assert.Nil(t, err)
	err = c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/02",
		Id:   "02",
	})
	assert.Nil(t, err)
	err = c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/03",
		Id:   "03",
	})
	assert.Nil(t, err)

	etcd, err := NewEtcdV3Server(cfg, c, fr)
	assert.Nil(t, err)

	reqCh := make(chan *etcdserverpb.WatchRequest)
	respCh := make(chan *etcdserverpb.WatchResponse)

	stream := &fakeWatchServer{
		ServerStream: nil,
		reqCh:        reqCh,
		respCh:       respCh,
	}
	ws := &watchStream{
		id:     1,
		etcd:   etcd.(*etcdV3),
		stream: stream,
	}
	ws.etcd.metaCache = map[string]meta{
		"/apisix/upstreams/01": {
			createRevision: 3,
			modRevision:    5,
		},
		"/apisix/upstreams/02": {
			createRevision: 1,
			modRevision:    1,
		},
		"/apisix/upstreams/03": {
			createRevision: 4,
			modRevision:    8,
		},
	}

	go func() {
		err := ws.firstWatch(134, "upstream", 2)
		assert.Nil(t, err)
	}()

	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't get response on time")
	case resp := <-stream.respCh:
		assert.Equal(t, resp.WatchId, int64(134))
		assert.Len(t, resp.Events, 2)
	}
}

func TestWatch(t *testing.T) {
	fr := &fakeRevisioner{rev: 1}
	cfg := &config.Config{
		LogLevel:      "debug",
		LogOutput:     "stderr",
		EtcdKeyPrefix: "/apisix",
	}
	c := cache.NewInMemoryCache()
	err := c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/01",
		Id:   "01",
	})
	assert.Nil(t, err)
	err = c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/02",
		Id:   "02",
	})
	assert.Nil(t, err)
	err = c.Upstream().Insert(&apisix.Upstream{
		Name: "/apisix/upstreams/03",
		Id:   "03",
	})
	assert.Nil(t, err)
	err = c.Route().Insert(&apisix.Route{
		Name: "/apisix/routes/01",
		Id:   "01",
	})
	assert.Nil(t, err)
	err = c.Route().Insert(&apisix.Route{
		Name: "/apisix/routes/02",
		Id:   "02",
	})
	assert.Nil(t, err)
	err = c.Route().Insert(&apisix.Route{
		Name: "/apisix/routes/03",
		Id:   "03",
	})
	assert.Nil(t, err)

	etcd, err := NewEtcdV3Server(cfg, c, fr)
	assert.Nil(t, err)

	reqCh := make(chan *etcdserverpb.WatchRequest)
	respCh := make(chan *etcdserverpb.WatchResponse)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &fakeWatchServer{
		ServerStream: nil,
		ctx:          ctx,
		reqCh:        reqCh,
		respCh:       respCh,
	}
	etcd.(*etcdV3).metaCache = map[string]meta{
		"/apisix/upstreams/01": {
			createRevision: 3,
			modRevision:    5,
		},
		"/apisix/upstreams/02": {
			createRevision: 1,
			modRevision:    1,
		},
		"/apisix/upstreams/03": {
			createRevision: 4,
			modRevision:    8,
		},
		"/apisix/routes/01": {
			createRevision: 1,
			modRevision:    2,
		},
		"/apisix/routes/02": {
			createRevision: 1,
			modRevision:    3,
		},
		"/apisix/routes/03": {
			createRevision: 4,
			modRevision:    6,
		},
	}

	checkStopC := make(chan struct{})
	go func() {
		err := etcd.(*etcdV3).Watch(stream)
		assert.Nil(t, err)
		close(checkStopC)
	}()

	wr := &etcdserverpb.WatchRequest{
		RequestUnion: &etcdserverpb.WatchRequest_CreateRequest{
			CreateRequest: &etcdserverpb.WatchCreateRequest{
				Key:           []byte("/apisix/routes"),
				RangeEnd:      []byte("/apisix/routet"),
				StartRevision: 1,
			},
		},
	}
	stream.reqCh <- wr
	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't recv watch response in time")
	case resp := <-stream.respCh:
		assert.Len(t, resp.Events, 3)
	}

	wr = &etcdserverpb.WatchRequest{
		RequestUnion: &etcdserverpb.WatchRequest_CreateRequest{
			CreateRequest: &etcdserverpb.WatchCreateRequest{
				Key:           []byte("/apisix/upstreams"),
				RangeEnd:      []byte("/apisix/upstreamt"),
				StartRevision: 3,
			},
		},
	}
	stream.reqCh <- wr
	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't recv watch response in time")
	case resp := <-stream.respCh:
		assert.Len(t, resp.Events, 2)
	}

	cancel()
	// Make sure the watch goroutine exited.
	<-checkStopC
}

func TestWatchWithPushEvents(t *testing.T) {
	fr := &fakeRevisioner{rev: 1}
	cfg := &config.Config{
		LogLevel:      "debug",
		LogOutput:     "stderr",
		EtcdKeyPrefix: "/apisix",
	}
	c := cache.NewInMemoryCache()
	etcd, err := NewEtcdV3Server(cfg, c, fr)
	assert.Nil(t, err)

	reqCh := make(chan *etcdserverpb.WatchRequest)
	respCh := make(chan *etcdserverpb.WatchResponse)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &fakeWatchServer{
		ServerStream: nil,
		ctx:          ctx,
		reqCh:        reqCh,
		respCh:       respCh,
	}
	checkStopC := make(chan struct{})
	go func() {
		err := etcd.(*etcdV3).Watch(stream)
		assert.Nil(t, err)
		close(checkStopC)
	}()

	wr := &etcdserverpb.WatchRequest{
		RequestUnion: &etcdserverpb.WatchRequest_CreateRequest{
			CreateRequest: &etcdserverpb.WatchCreateRequest{
				Key:           []byte("/apisix/routes"),
				RangeEnd:      []byte("/apisix/routet"),
				StartRevision: 1,
			},
		},
	}
	stream.reqCh <- wr
	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't receive response in time")
	case resp := <-stream.respCh:
		assert.Len(t, resp.Events, 0)
	}

	events := []types.Event{
		{
			Type: types.EventAdd,
			Object: &apisix.Route{
				Id:   "135d",
				Name: "a.b.c.d.com",
			},
		},
	}
	etcd.PushEvents(events)
	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't receive events on time")
	case resp := <-stream.respCh:
		assert.Len(t, resp.Events, 1)
		assert.Equal(t, resp.Events[0].Type, mvccpb.PUT)
		assert.Equal(t, resp.Events[0].Kv.Key, []byte("/apisix/routes/135d"))
	}

	wr = &etcdserverpb.WatchRequest{
		RequestUnion: &etcdserverpb.WatchRequest_CreateRequest{
			CreateRequest: &etcdserverpb.WatchCreateRequest{
				Key:           []byte("/apisix/upstreams"),
				RangeEnd:      []byte("/apisix/upstreamt"),
				StartRevision: 0,
			},
		},
	}
	stream.reqCh <- wr
	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't receive response in time")
	case resp := <-stream.respCh:
		assert.Len(t, resp.Events, 0)
	}

	events = []types.Event{
		{
			Type: types.EventAdd,
			Object: &apisix.Upstream{
				Id:   "98k",
				Name: "aha",
			},
		},
		{
			Type: types.EventDelete,
			Tombstone: &apisix.Route{
				Id:   "135d",
				Name: "a.b.c.d.com",
			},
		},
	}
	etcd.PushEvents(events)

	var (
		ev1 *mvccpb.Event
		ev2 *mvccpb.Event
	)
	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't receive events on time")
	case resp := <-stream.respCh:
		assert.Len(t, resp.Events, 1)
		ev1 = resp.Events[0]
	}
	select {
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "didn't receive events on time")
	case resp := <-stream.respCh:
		assert.Len(t, resp.Events, 1)
		ev2 = resp.Events[0]
	}
	if string(ev1.Kv.Key) > string(ev2.Kv.Key) {
		ev1, ev2 = ev2, ev1
	}

	assert.Equal(t, ev1.Type, mvccpb.DELETE)
	assert.Equal(t, ev1.Kv.Key, []byte("/apisix/routes/135d"))
	assert.Equal(t, ev2.Type, mvccpb.PUT)
	assert.Equal(t, ev2.Kv.Key, []byte("/apisix/upstreams/98k"))

	cancel()
	// Make sure the watch goroutine exited.
	<-checkStopC
}

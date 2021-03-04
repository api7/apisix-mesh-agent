package etcdv3

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"golang.org/x/net/nettest"
	"google.golang.org/grpc"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestNewEtcdV3Server(t *testing.T) {
	cfg := config.NewDefaultConfig()
	// defined in kv_test.go
	fr := &fakeRevisioner{rev: 0}

	srv, err := NewEtcdV3Server(cfg, cache.NewInMemoryCache(), fr)
	assert.Nil(t, err)
	assert.NotNil(t, srv)
}

func TestEtcdV3ServerRun(t *testing.T) {
	cfg := config.NewDefaultConfig()
	// defined in kv_test.go
	fr := &fakeRevisioner{rev: 3}

	c := cache.NewInMemoryCache()
	srv, err := NewEtcdV3Server(cfg, c, fr)
	assert.Nil(t, err)
	assert.NotNil(t, srv)

	stopCh := make(chan struct{})
	listener, err := nettest.NewLocalListener("tcp")
	assert.Nil(t, err)
	go func() {
		err := srv.Serve(listener)
		assert.Nil(t, err)
		close(stopCh)
	}()

	dialCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, listener.Addr().String(),
		grpc.WithBlock(),
		grpc.WithInsecure(),
	)
	assert.Nil(t, err)

	client := etcdserverpb.NewKVClient(conn)
	rr := &etcdserverpb.RangeRequest{
		Key: []byte("/apisix/routes/1"),
	}
	resp, err := client.Range(context.Background(), rr)
	assert.Nil(t, err)
	assert.Len(t, resp.Kvs, 0)

	u := &apisix.Upstream{
		Id: "1",
	}
	assert.Nil(t, c.Upstream().Insert(u))

	rr.Key = []byte("/apisix/upstreams")
	rr.RangeEnd = []byte("/apisix/upstreamt")
	resp, err = client.Range(context.Background(), rr)
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.Len(t, resp.Kvs, 1)
	assert.Equal(t, resp.Kvs[0].Key, []byte("/apisix/upstreams/1"))

	assert.Nil(t, srv.Shutdown(context.Background()))
	select {
	case <-stopCh:
		break
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "etcd v3 server didn't stop")
	}
}

func TestPushEvents(t *testing.T) {
	events := []types.Event{
		{
			Type:   types.EventAdd,
			Object: &apisix.Route{Id: "123"},
		},
		{
			Type:   types.EventAdd,
			Object: &apisix.Route{Id: "124"},
		},
		{
			Type:   types.EventAdd,
			Object: &apisix.Upstream{Id: "125"},
		},
	}
	f := &fakeRevisioner{rev: 1}
	cfg := &config.Config{
		LogLevel:      "debug",
		LogOutput:     "stderr",
		EtcdKeyPrefix: "/apisix",
	}
	etcd, err := NewEtcdV3Server(cfg, cache.NewInMemoryCache(), f)
	assert.Nil(t, err)
	ws := &watchStream{
		ctx:      context.Background(),
		eventCh:  make(chan *etcdserverpb.WatchResponse),
		etcd:     etcd.(*etcdV3),
		route:    make(map[int64]struct{}),
		upstream: make(map[int64]struct{}),
	}
	etcd.(*etcdV3).watchers[1] = ws
	ws.route[1] = struct{}{}
	ws.upstream[1] = struct{}{}

	etcd.PushEvents(events)

	for i := 0; i < 3; i++ {
		select {
		case <-time.After(2 * time.Second):
			assert.FailNow(t, "didn't receive event in time")
		case <-ws.eventCh:
		}
	}
}

func TestVersion(t *testing.T) {
	rw := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/version", nil)
	e := &etcdV3{
		logger: log.DefaultLogger,
	}
	e.version(rw, req)
	assert.Equal(t, rw.Code, 200)
	assert.Equal(t, rw.Body.String(), `{"etcdserver":"3.5.0-pre","etcdcluster":"3.5.0"}`)
}

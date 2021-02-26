package etcdv3

import (
	"context"
	"testing"
	"time"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"golang.org/x/net/nettest"
	"google.golang.org/grpc"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
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
	assert.Nil(t, resp)
	// err.Error() also has the code information.
	assert.Contains(t, err.Error(), rpctypes.ErrKeyNotFound.Error())

	u := &apisix.Upstream{
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "1",
			},
		},
	}
	assert.Nil(t, c.Upstream().Insert(u))

	rr.Key = []byte("/apisix/upstreams")
	rr.RangeEnd = []byte("/apisix/upstreamt")
	resp, err = client.Range(context.Background(), rr)
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.Len(t, resp.Kvs, 1)
	assert.Equal(t, resp.Kvs[0].Key, []byte("/apisix/upstreams/1"))

	assert.Nil(t, srv.Shutdown())
	select {
	case <-stopCh:
		break
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "etcd v3 server didn't stop")
	}
}

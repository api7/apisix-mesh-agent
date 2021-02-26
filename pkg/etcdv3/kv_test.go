package etcdv3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"google.golang.org/protobuf/proto"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

type fakeRevisioner struct {
	rev int64
}

func (f *fakeRevisioner) Revision() int64 {
	return f.rev
}

func TestComposeKeyValue(t *testing.T) {
	fr := &fakeRevisioner{rev: 3}
	e := &etcdV3{
		metaCache:  make(map[string]meta),
		revisioner: fr,
		keyPrefix:  "",
	}
	key := []byte("/apisix/route/0003")
	value := []byte("null")

	kv := e.composeKeyValue(key, value)
	assert.NotNil(t, kv)
	assert.Equal(t, kv.Key, key)
	assert.Equal(t, kv.Value, value)
	assert.Equal(t, kv.CreateRevision, int64(3))
	assert.Equal(t, kv.ModRevision, int64(3))

	m, ok := e.metaCache[string(key)]
	assert.Equal(t, ok, true)

	m.modRevision = 111
	m.createRevision = 19
	e.metaCache[string(key)] = m

	kv = e.composeKeyValue(key, value)
	assert.NotNil(t, kv)
	assert.Equal(t, kv.Key, key)
	assert.Equal(t, kv.Value, value)
	assert.Equal(t, kv.CreateRevision, int64(19))
	assert.Equal(t, kv.ModRevision, int64(111))
}

func TestFindExactKey(t *testing.T) {
	fr := &fakeRevisioner{rev: 3}
	e := &etcdV3{
		metaCache:  make(map[string]meta),
		revisioner: fr,
		keyPrefix:  "/apisix",
		cache:      cache.NewInMemoryCache(),
		logger:     log.DefaultLogger,
	}

	// Key prefix not match
	resp, err := e.findExactKey([]byte("/k8s/routes"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	// Find exact value with key like "/apisix/route/00001".
	resp, err = e.findExactKey([]byte("/apisix/route"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	resp, err = e.findExactKey([]byte("/apisix/others/0123"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	resp, err = e.findExactKey([]byte("/apisix/routes/0123"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	route := &apisix.Route{
		Uris: []string{"/foo*"},
		Name: "route1",
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "0123",
			},
		},
		Status: 1,
	}
	assert.Nil(t, e.cache.Route().Insert(route))

	resp, err = e.findExactKey([]byte("/apisix/routes/0123"))
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.Len(t, resp.Kvs, 1)
	assert.Equal(t, resp.Kvs[0].ModRevision, int64(3))
	assert.Equal(t, resp.Kvs[0].CreateRevision, int64(3))
	assert.Equal(t, resp.Kvs[0].Key, []byte("/apisix/routes/0123"))
	var route2 apisix.Route
	assert.Nil(t, proto.Unmarshal(resp.Kvs[0].Value, &route2))
	assert.Equal(t, route.Uris, route2.Uris)
	assert.Equal(t, route.Name, route2.Name)
	assert.Equal(t, route.Id.GetStrVal(), route2.Id.GetStrVal())

	resp, err = e.findExactKey([]byte("/apisix/upstreams/00003"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	ups := &apisix.Upstream{
		Name: "ups1",
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "00003",
			},
		},
		Timeout: &apisix.Upstream_Timeout{
			Connect: 5,
			Send:    5,
			Read:    5,
		},
	}
	fr.rev += 88
	assert.Nil(t, e.cache.Upstream().Insert(ups))

	resp, err = e.findExactKey([]byte("/apisix/upstreams/00003"))
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.Len(t, resp.Kvs, 1)
	assert.Equal(t, resp.Kvs[0].ModRevision, int64(91))
	assert.Equal(t, resp.Kvs[0].CreateRevision, int64(91))
	assert.Equal(t, resp.Kvs[0].Key, []byte("/apisix/upstreams/00003"))

	var ups2 apisix.Upstream
	assert.Nil(t, proto.Unmarshal(resp.Kvs[0].Value, &ups2))
	assert.Equal(t, ups.Name, ups2.Name)
	assert.Equal(t, ups.Id.GetStrVal(), ups2.Id.GetStrVal())
	assert.Equal(t, ups.Timeout, ups2.Timeout)
}

func TestFindAllKeys(t *testing.T) {
	fr := &fakeRevisioner{rev: 3}
	e := &etcdV3{
		metaCache:  make(map[string]meta),
		revisioner: fr,
		keyPrefix:  "/apisix",
		cache:      cache.NewInMemoryCache(),
		logger:     log.DefaultLogger,
	}

	// Key prefix not match
	resp, err := e.findAllKeys([]byte("/kubernetes/routes"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	// Key should be like /apisix/routes, without the specific id.
	resp, err = e.findAllKeys([]byte("/apisix/routes/001"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	resp, err = e.findAllKeys([]byte("/apisix/others"))
	assert.Nil(t, resp, nil)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)

	fr.rev = 89

	r1 := &apisix.Route{
		Name: "/apisix/routes/1",
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "1",
			},
		},
	}
	r2 := &apisix.Route{
		Name: "/apisix/routes/2",
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "2",
			},
		},
	}
	assert.Nil(t, e.cache.Route().Insert(r1))
	assert.Nil(t, e.cache.Route().Insert(r2))
	resp, err = e.findAllKeys([]byte("/apisix/routes"))
	assert.Nil(t, err)
	assert.Len(t, resp.Kvs, 2)
	assert.Equal(t, resp.Kvs[0].CreateRevision, int64(89))
	assert.Equal(t, resp.Kvs[0].ModRevision, int64(89))
	assert.Equal(t, resp.Kvs[1].CreateRevision, int64(89))
	assert.Equal(t, resp.Kvs[1].ModRevision, int64(89))

	u1 := &apisix.Upstream{
		Name: "/apisix/upstreams/1",
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "1",
			},
		},
	}
	fr.rev++
	assert.Nil(t, e.cache.Upstream().Insert(u1))
	resp, err = e.findAllKeys([]byte("/apisix/upstreams"))
	assert.Nil(t, err)
	assert.Len(t, resp.Kvs, 1)
	assert.Equal(t, resp.Kvs[0].CreateRevision, int64(90))
	assert.Equal(t, resp.Kvs[0].ModRevision, int64(90))
}

func TestRangeRequest(t *testing.T) {
	fr := &fakeRevisioner{rev: 3}
	e := &etcdV3{
		metaCache:  make(map[string]meta),
		revisioner: fr,
		keyPrefix:  "/apisix",
		cache:      cache.NewInMemoryCache(),
		logger:     log.DefaultLogger,
	}

	rr := &etcdserverpb.RangeRequest{
		Key:      []byte("/apisix/routes/1"),
		RangeEnd: nil,
	}
	resp, err := e.Range(context.Background(), rr)
	assert.Nil(t, resp)
	assert.Equal(t, err, rpctypes.ErrKeyNotFound)
	r1 := &apisix.Route{
		Name: "/apisix/routes/1",
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "1",
			},
		},
	}
	r2 := &apisix.Route{
		Name: "/apisix/routes/2",
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "2",
			},
		},
	}
	assert.Nil(t, e.cache.Route().Insert(r1))
	assert.Nil(t, e.cache.Route().Insert(r2))

	resp, err = e.Range(context.Background(), rr)
	assert.Len(t, resp.Kvs, 1)
	assert.Equal(t, resp.Kvs[0].Key, []byte("/apisix/routes/1"))
	assert.Nil(t, err)

	rr.Key = []byte("/apisix/routes")
	rr.RangeEnd = []byte("/apisix/routet")
	resp, err = e.Range(context.Background(), rr)
	assert.Len(t, resp.Kvs, 2)
	assert.Equal(t, resp.Kvs[0].Key, []byte("/apisix/routes/1"))
	assert.Equal(t, resp.Kvs[1].Key, []byte("/apisix/routes/2"))
	assert.Nil(t, err)

}

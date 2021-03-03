package etcdv3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"

	"github.com/api7/apisix-mesh-agent/pkg/log"
)

func TestCheckRangeRequestConformance(t *testing.T) {
	e := &etcdV3{
		logger: log.DefaultLogger,
	}
	r := &etcdserverpb.RangeRequest{}

	// Empty key
	assert.Equal(t, e.checkRangeRequestConformance(r), rpctypes.ErrEmptyKey)

	// Unsupported range query.
	r.Key = []byte("/apisix/aaaa")
	r.RangeEnd = []byte("/apisix/route/xxx")
	assert.Equal(t, e.checkRangeRequestConformance(r), rpctypes.ErrKeyNotFound)
	r.RangeEnd = nil

	// Limitations.
	r.Limit = 11
	assert.Equal(t, e.checkRangeRequestConformance(r), rpctypes.ErrNotCapable)
	r.Limit = 0

	// Sort
	r.SortOrder = etcdserverpb.RangeRequest_ASCEND
	assert.Equal(t, e.checkRangeRequestConformance(r), rpctypes.ErrNotCapable)
	r.SortOrder = etcdserverpb.RangeRequest_NONE

	// Revision
	r.MaxCreateRevision = 1333
	assert.Equal(t, e.checkRangeRequestConformance(r), rpctypes.ErrNotCapable)

	r.MaxCreateRevision = 0
	assert.Nil(t, e.checkRangeRequestConformance(r))
}

func TestCheckWatchRequestConformance(t *testing.T) {
	e := &etcdV3{
		logger:    log.DefaultLogger,
		keyPrefix: "/apisix",
	}
	r := &etcdserverpb.WatchRequest{
		RequestUnion: &etcdserverpb.WatchRequest_CancelRequest{},
	}
	// WatchCancelRequest
	assert.Nil(t, e.checkWatchRequestConformance(r))
	// WatchProgressRequest
	r.RequestUnion = &etcdserverpb.WatchRequest_ProgressRequest{}
	assert.Equal(t, e.checkWatchRequestConformance(r), rpctypes.ErrNotCapable)
	// Empty CreateRequest
	r.RequestUnion = &etcdserverpb.WatchRequest_CreateRequest{}
	assert.Nil(t, e.checkWatchRequestConformance(r))
	// Empty key
	r.RequestUnion = &etcdserverpb.WatchRequest_CreateRequest{
		CreateRequest: &etcdserverpb.WatchCreateRequest{},
	}
	assert.Equal(t, e.checkWatchRequestConformance(r), rpctypes.ErrEmptyKey)

	// Bad Key and RandEnd combination.
	r.RequestUnion = &etcdserverpb.WatchRequest_CreateRequest{
		CreateRequest: &etcdserverpb.WatchCreateRequest{
			Key:      []byte("/apisix/unknowns"),
			RangeEnd: []byte("/apisix/unknownt"),
		},
	}
	assert.Equal(t, e.checkWatchRequestConformance(r), rpctypes.ErrKeyNotFound)

	// PrevKv
	r.RequestUnion = &etcdserverpb.WatchRequest_CreateRequest{
		CreateRequest: &etcdserverpb.WatchCreateRequest{
			Key:      []byte("/apisix/routes"),
			RangeEnd: []byte("/apisix/routet"),
			PrevKv:   true,
		},
	}
	assert.Equal(t, e.checkWatchRequestConformance(r), rpctypes.ErrNotCapable)

	// ProgressNotify
	r.RequestUnion = &etcdserverpb.WatchRequest_CreateRequest{
		CreateRequest: &etcdserverpb.WatchCreateRequest{
			Key:            []byte("/apisix/routes"),
			RangeEnd:       []byte("/apisix/routet"),
			ProgressNotify: true,
		},
	}
	assert.Equal(t, e.checkWatchRequestConformance(r), rpctypes.ErrNotCapable)
	// Fragment
	r.RequestUnion = &etcdserverpb.WatchRequest_CreateRequest{
		CreateRequest: &etcdserverpb.WatchCreateRequest{
			Key:      []byte("/apisix/routes"),
			RangeEnd: []byte("/apisix/routet"),
			Fragment: true,
		},
	}
	assert.Equal(t, e.checkWatchRequestConformance(r), rpctypes.ErrNotCapable)
}

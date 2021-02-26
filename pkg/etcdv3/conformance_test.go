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
	assert.Equal(t, e.checkRangeRequestConformance(r), rpctypes.ErrNotCapable)
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

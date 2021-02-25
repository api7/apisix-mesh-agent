package etcdv3

import (
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/log"
)

func (e *etcdV3) checkRangeRequestConformance(r *etcdserverpb.RangeRequest) error {
	if len(r.Key) == 0 {
		return rpctypes.ErrEmptyKey
	}
	if !(r.RangeEnd == nil ||
		(string(r.Key) == "/apisix/routes" && string(r.RangeEnd) == "/apisix/routet") ||
		(string(r.Key) == "/apisix/upstreams" && string(r.RangeEnd) == "/apisix/upstreamt")) {

		log.Warnw("RangeRequest with unsupported key and range_end combination",
			zap.String("key", string(r.Key)),
			zap.String("range_end", string(r.RangeEnd)),
		)
		return rpctypes.ErrNotCapable
	}
	if r.Limit != 0 {
		log.Warnw("RangeRequest with unsupported non-zero limit",
			zap.Int64("limit", r.Limit),
		)
		return rpctypes.ErrNotCapable
	}
	if r.SortOrder != etcdserverpb.RangeRequest_NONE {
		log.Warnw("RangeRequest requires sorting is not supported yet",
			zap.String("sort_order", r.SortOrder.String()),
		)
		return rpctypes.ErrNotCapable
	}
	if r.Revision > 0 || r.MinCreateRevision > 0 || r.MaxCreateRevision > 0 || r.MinModRevision > 0 || r.MaxModRevision > 0 {
		log.Warnw("RangeRequest with specific revisions is not supported yet",
			zap.Int64("revision", r.Revision),
			zap.Int64("min_create_revision", r.MinCreateRevision),
			zap.Int64("max_create_revision", r.MaxCreateRevision),
			zap.Int64("min_mod_revision", r.MinModRevision),
			zap.Int64("max_mod_revision", r.MaxModRevision),
		)
		return rpctypes.ErrNotCapable
	}
	return nil
}

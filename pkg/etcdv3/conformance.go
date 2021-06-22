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
	key := string(r.Key)
	randEnd := string(r.RangeEnd)
	if !(r.RangeEnd == nil ||
		(key == e.keyPrefix+"/routes" && randEnd == e.keyPrefix+"/routet") || // resty.etcd uses key+1 as range end
		(key == e.keyPrefix+"/upstreams" && randEnd == e.keyPrefix+"/upstreamt")) {

		log.Warnw("RangeRequest with unsupported key and range_end combination",
			zap.String("key", string(r.Key)),
			zap.String("range_end", string(r.RangeEnd)),
		)
		return rpctypes.ErrKeyNotFound
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

func (e *etcdV3) checkWatchRequestConformance(r *etcdserverpb.WatchRequest) error {
	switch wr := r.RequestUnion.(type) {
	case *etcdserverpb.WatchRequest_CancelRequest:
		return nil
	case *etcdserverpb.WatchRequest_CreateRequest:
		if wr.CreateRequest == nil {
			return nil
		}
		key := string(wr.CreateRequest.Key)
		rangeEnd := string(wr.CreateRequest.RangeEnd)
		if len(key) == 0 {
			return rpctypes.ErrEmptyKey
		}
		if !((key == e.keyPrefix+"/routes" && rangeEnd == e.keyPrefix+"/routet") ||
			(key == e.keyPrefix+"/upstreams" && rangeEnd == e.keyPrefix+"/upstreamt")) {

			log.Warnw("WatchCreateRequest with unsupported key and range_end combination",
				zap.String("key", string(wr.CreateRequest.Key)),
				zap.String("range_end", string(wr.CreateRequest.RangeEnd)),
				zap.Any("watch_create_request", wr),
			)
			return rpctypes.ErrKeyNotFound
		}
		if wr.CreateRequest.PrevKv {
			log.Warnw("WatchCreateRequest enables prev_kv, which is not supported yet",
				zap.Any("watch_create_request", wr),
			)
			return rpctypes.ErrNotCapable
		}
		if wr.CreateRequest.ProgressNotify {
			log.Warnw("WatchCreateRequest enables progress notify, which is not supported yet",
				zap.Any("watch_create_request", wr),
			)
			return rpctypes.ErrNotCapable
		}
		if wr.CreateRequest.Fragment {
			log.Warnw("WatchCreateRequest enables fragmented is not supported yet",
				zap.Any("watch_create_request", wr),
			)
			return rpctypes.ErrNotCapable
		}
	case *etcdserverpb.WatchRequest_ProgressRequest:
		log.Warnw("WatchProgressRequest is not supported yet",
			zap.Any("watch_progress_request", wr),
		)
		return rpctypes.ErrNotCapable
	}
	return nil
}

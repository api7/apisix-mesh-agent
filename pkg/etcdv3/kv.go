package etcdv3

import (
	"context"
	"encoding/json"
	"strings"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
)

var (
	_errInternalError   = status.New(codes.Internal, "internal error").Err()
	_emptyRangeResponse = &etcdserverpb.RangeResponse{
		Header: &etcdserverpb.ResponseHeader{},
	}
)

// Range implements etcdserverpb.KVServer.Range method.
func (e *etcdV3) Range(ctx context.Context, r *etcdserverpb.RangeRequest) (*etcdserverpb.RangeResponse, error) {
	e.logger.Debugw("received RangeRequest",
		zap.Any("range_request", r),
	)
	if err := e.checkRangeRequestConformance(r); err != nil {
		if err == rpctypes.ErrKeyNotFound {
			_emptyRangeResponse.Header.Revision = e.revisioner.Revision()
			return _emptyRangeResponse, nil
		}
		return nil, err
	}
	var (
		resp *etcdserverpb.RangeResponse
		err  error
	)
	if r.RangeEnd == nil {
		resp, err = e.findExactKey(r.Key)
	} else {
		resp, err = e.findAllKeys(r.Key)
	}
	if err != nil {
		if err != rpctypes.ErrKeyNotFound {
			return nil, err
		}
		_emptyRangeResponse.Header.Revision = e.revisioner.Revision()
		return _emptyRangeResponse, nil
	}
	if r.KeysOnly {
		for _, kv := range resp.Kvs {
			kv.Value = nil
		}
	}
	e.logger.Debugw("RangeRequest response",
		zap.Any("response", resp),
	)
	return resp, nil
}

// Put implements etcdserverpb.KVServer.Put method.
func (e *etcdV3) Put(ctx context.Context, r *etcdserverpb.PutRequest) (*etcdserverpb.PutResponse, error) {
	e.logger.Warnw("PutRequest received",
		zap.Any("put_request", r),
	)
	return &etcdserverpb.PutResponse{
		Header: &etcdserverpb.ResponseHeader{
			Revision: e.revisioner.Revision(),
		},
	}, nil
}

// DeleteRange implements etcdserverpb.KVServer.DeleteRange method.
func (e *etcdV3) DeleteRange(ctx context.Context, r *etcdserverpb.DeleteRangeRequest) (*etcdserverpb.DeleteRangeResponse, error) {
	e.logger.Warnw("DeleteRange received, which is not supported yet")
	return nil, rpctypes.ErrNotCapable
}

// Txn implements etcdserverpb.KVServer.Txn method.
func (e *etcdV3) Txn(ctx context.Context, r *etcdserverpb.TxnRequest) (*etcdserverpb.TxnResponse, error) {
	e.logger.Warnw("Txn received, which is not supported yet")
	return nil, rpctypes.ErrNotCapable
}

// Compact implements etcdserverpb..Compact method.
func (e *etcdV3) Compact(ctx context.Context, r *etcdserverpb.CompactionRequest) (*etcdserverpb.CompactionResponse, error) {
	e.logger.Warnw("Compact received, which is not supported yet")
	return nil, rpctypes.ErrNotCapable
}

func (e *etcdV3) composeKeyValue(key []byte, value []byte) *mvccpb.KeyValue {
	e.metaMu.RLock()
	m, ok := e.metaCache[string(key)]
	e.metaMu.RUnlock()
	if !ok {
		rev := e.revisioner.Revision()
		m = meta{
			createRevision: rev,
			modRevision:    rev,
		}
		e.metaMu.Lock()
		e.metaCache[string(key)] = m
		e.metaMu.Unlock()
	}

	return &mvccpb.KeyValue{
		Key:            key,
		CreateRevision: m.createRevision,
		ModRevision:    m.modRevision,
		Value:          value,
	}
}

func (e *etcdV3) findExactKey(key []byte) (*etcdserverpb.RangeResponse, error) {
	tempKey := string(key)
	if !strings.HasPrefix(tempKey, e.keyPrefix) {
		return nil, rpctypes.ErrKeyNotFound
	}
	tempKey = strings.TrimPrefix(tempKey, e.keyPrefix)
	parts := strings.Split(tempKey, "/")
	if len(parts) != 3 || parts[0] != "" {
		return nil, rpctypes.ErrKeyNotFound
	}
	var (
		value []byte
	)
	switch parts[1] {
	case "routes":
		e.logger.Debugw("request for route",
			zap.String("route_id", parts[2]),
		)
		route, err := e.cache.Route().Get(parts[2])
		if err != nil {
			if err == cache.ErrObjectNotFound {
				return nil, rpctypes.ErrKeyNotFound
			}
			return nil, _errInternalError
		}
		value, err = json.Marshal(route)
		if err != nil {
			e.logger.Errorw("failed to marshal route",
				zap.Any("route", route),
				zap.Error(err),
			)
			return nil, _errInternalError
		}
	case "upstreams":
		e.logger.Debugw("request for upstream",
			zap.String("upstream_id", parts[2]),
		)
		ups, err := e.cache.Upstream().Get(parts[2])
		if err != nil {
			if err == cache.ErrObjectNotFound {
				return nil, rpctypes.ErrKeyNotFound
			}
			return nil, _errInternalError
		}
		value, err = json.Marshal(ups)
		if err != nil {
			e.logger.Errorw("failed to marshal upstream",
				zap.Any("upstream", ups),
				zap.Error(err),
			)
			return nil, _errInternalError
		}
	default:
		e.logger.Warnw("request for unknown resources",
			zap.String("key", string(key)),
		)
		return nil, rpctypes.ErrKeyNotFound
	}
	return &etcdserverpb.RangeResponse{
		Header: &etcdserverpb.ResponseHeader{},
		Kvs: []*mvccpb.KeyValue{
			e.composeKeyValue(key, value),
		},
		Count: 1,
	}, nil
}

func (e *etcdV3) findAllKeys(key []byte) (*etcdserverpb.RangeResponse, error) {
	tempKey := string(key)
	if !strings.HasPrefix(tempKey, e.keyPrefix) {
		return nil, rpctypes.ErrKeyNotFound
	}
	tempKey = strings.TrimPrefix(tempKey, e.keyPrefix)
	parts := strings.Split(tempKey, "/")
	if len(parts) != 2 || parts[0] != "" {
		return nil, rpctypes.ErrKeyNotFound
	}
	var kvs []*mvccpb.KeyValue
	switch parts[1] {
	case "routes":
		routes, err := e.cache.Route().List()
		if err != nil {
			e.logger.Errorw("failed to list routes",
				zap.Error(err),
			)
			return nil, _errInternalError
		}
		for _, r := range routes {
			itemKey := e.keyPrefix + "/routes/" + r.Id
			value, err := json.Marshal(r)
			if err != nil {
				e.logger.Errorw("failed to marshal route",
					zap.Error(err),
					zap.Any("route", r),
				)
				return nil, _errInternalError
			}
			kvs = append(kvs, e.composeKeyValue([]byte(itemKey), value))
		}
	case "upstreams":
		upstreams, err := e.cache.Upstream().List()
		if err != nil {
			e.logger.Errorw("failed to list upstreams",
				zap.Error(err),
			)
			return nil, _errInternalError
		}
		for _, u := range upstreams {
			itemKey := e.keyPrefix + "/upstreams/" + u.Id
			value, err := json.Marshal(u)
			if err != nil {
				e.logger.Errorw("failed to marshal upstream",
					zap.Error(err),
					zap.Any("upstream", u),
				)
				return nil, _errInternalError
			}
			kvs = append(kvs, e.composeKeyValue([]byte(itemKey), value))
		}
	default:
		return nil, rpctypes.ErrKeyNotFound
	}
	return &etcdserverpb.RangeResponse{
		Header: &etcdserverpb.ResponseHeader{},
		Kvs:    kvs,
		Count:  int64(len(kvs)),
	}, nil
}

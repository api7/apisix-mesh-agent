package cache

import (
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

type upstream struct {
	mu sync.RWMutex
	// TODO optimize the store if the performance of map
	// is unbearable.
	store map[string]*apisix.Upstream
}

func newUpstream() Upstream {
	return &upstream{
		store: make(map[string]*apisix.Upstream),
	}
}

func (r *upstream) Get(id string) (*apisix.Upstream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	obj, ok := r.store[id]
	if !ok {
		return nil, ErrObjectNotFound
	}
	// Never return the original one to avoid race conditions.
	return proto.Clone(obj).(*apisix.Upstream), nil
}

func (r *upstream) List() ([]*apisix.Upstream, error) {
	var objs []*apisix.Upstream
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, obj := range r.store {
		objs = append(objs, proto.Clone(obj).(*apisix.Upstream))
	}
	return objs, nil
}

func (r *upstream) Insert(obj *apisix.Upstream) error {
	obj = proto.Clone(obj).(*apisix.Upstream)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[obj.Id] = obj
	return nil
}

func (r *upstream) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.store[id]
	if !ok {
		return ErrObjectNotFound
	}
	delete(r.store, id)
	return nil
}

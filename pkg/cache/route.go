package cache

import (
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

type route struct {
	mu sync.RWMutex
	// TODO optimize the store if the performance of map
	// is unbearable.
	store map[string]*apisix.Route
}

func newRoute() Route {
	return &route{
		store: make(map[string]*apisix.Route),
	}
}

func (r *route) Get(id string) (*apisix.Route, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	obj, ok := r.store[id]
	if !ok {
		return nil, ErrObjectNotFound
	}
	// Never return the original one to avoid race conditions.
	return proto.Clone(obj).(*apisix.Route), nil
}

func (r *route) List() ([]*apisix.Route, error) {
	var objs []*apisix.Route
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, obj := range r.store {
		objs = append(objs, proto.Clone(obj).(*apisix.Route))
	}
	return objs, nil
}

func (r *route) Insert(obj *apisix.Route) error {
	obj = proto.Clone(obj).(*apisix.Route)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[obj.Id.GetStrVal()] = obj
	return nil
}

func (r *route) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.store[id]
	if !ok {
		return ErrObjectNotFound
	}
	delete(r.store, id)
	return nil
}

package cache

import (
	"errors"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

var (
	// ErrObjectNotFound means the target object is not found
	// from the cache.
	ErrObjectNotFound = errors.New("object not found")
)

// Cache defines what capabilities a cache solution should provide.
type Cache interface {
	// Route returns the route exclusive cache object.
	Route() Route
	// Upstream returns the upstream exclusive cache object.
	Upstream() Upstream
}

// Route defines the exclusive behaviors for apisix.Route.
type Route interface {
	// Get the apisix.Route by its id. In case of the object not found,
	// ErrObjectNotFound is given.
	Get(string) (*apisix.Route, error)
	// List lists all apisix.Route.
	List() ([]*apisix.Route, error)
	// Insert inserts or updates an apisix.Route object, indexed by its id.
	Insert(*apisix.Route) error
	// Delete deletes the apisix.Route object by the id. In case of object not
	// exist, ErrObjectNotFound is given.
	Delete(string) error
}

// Upstream defines the exclusive behaviors for apisix.Upstream.
type Upstream interface {
	// Get the apisix.Upstream by its id. In case of the object not found,
	// ErrObjectNotFound is given.
	Get(string) (*apisix.Upstream, error)
	// List lists all apisix.Upstream.
	List() ([]*apisix.Upstream, error)
	// Insert creates or updates an apisix.Upstream object, indexed by its id.
	Insert(*apisix.Upstream) error
	// Delete deletes the apisix.Upstream object by the id. In case of object not
	// exist, ErrObjectNotFound is given.
	Delete(string) error
}

type cache struct {
	route    Route
	upstream Upstream
}

// NewInMemoryCache creates a Cache object which stores all data in memory.
func NewInMemoryCache() Cache {
	return &cache{
		route:    newRoute(),
		upstream: newUpstream(),
	}
}

func (c *cache) Route() Route {
	return c.route
}

func (c *cache) Upstream() Upstream {
	return c.upstream
}

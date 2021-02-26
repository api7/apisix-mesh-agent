package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestInMemoryCache(t *testing.T) {
	c := NewInMemoryCache()
	assert.NotNil(t, c)

	ups := &apisix.Upstream{
		Id: "1",
	}
	r := &apisix.Route{
		Id: "1",
	}

	assert.Nil(t, c.Route().Insert(r))
	assert.Nil(t, c.Upstream().Insert(ups))

	rr, err := c.Route().Get("1")
	assert.Nil(t, err)
	assert.Equal(t, rr.GetId(), "1")

	uu, err := c.Upstream().Get("1")
	assert.Nil(t, err)
	assert.Equal(t, uu.GetId(), "1")
}

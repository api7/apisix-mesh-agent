package file

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestManifestSize(t *testing.T) {
	m := &manifest{
		Routes: []*apisix.Route{
			{}, {},
		},
		Upstreams: []*apisix.Upstream{
			{}, {},
		},
	}
	assert.Equal(t, m.size(), 4)
}

func TestManifestEvents(t *testing.T) {
	m := &manifest{
		Routes: []*apisix.Route{
			{}, {},
		},
		Upstreams: []*apisix.Upstream{
			{}, {},
		},
	}
	evs := m.events(types.EventAdd)
	assert.Len(t, evs, 4)
	assert.NotNil(t, evs[0].Object)
	assert.Nil(t, evs[0].Tombstone)
	assert.Equal(t, evs[0].Type, types.EventAdd)

	evs = m.events(types.EventUpdate)
	assert.Len(t, evs, 4)
	assert.NotNil(t, evs[0].Object)
	assert.Nil(t, evs[0].Tombstone)
	assert.Equal(t, evs[0].Type, types.EventUpdate)

	evs = m.events(types.EventDelete)
	assert.Len(t, evs, 4)
	assert.Nil(t, evs[0].Object)
	assert.NotNil(t, evs[0].Tombstone)
	assert.Equal(t, evs[0].Type, types.EventDelete)
}

func TestManifestDiffFrom(t *testing.T) {
	m := &manifest{
		Routes: []*apisix.Route{
			{
				Id: "1",
			},
			{
				Id: "2",
			},
		},
		Upstreams: []*apisix.Upstream{
			{
				Id: "1",
			},
			{
				Id: "2",
			},
		},
	}
	m2 := &manifest{
		Routes: []*apisix.Route{
			{
				Id:   "2",
				Uris: []string{"/foo"},
			},
			{
				Id: "3",
			},
		},
		Upstreams: []*apisix.Upstream{
			{
				Id: "1",
			},
		},
	}
	a, d, u := m.diffFrom(m2)
	assert.Equal(t, a.size(), 1)
	assert.Equal(t, a.Routes[0].Id, "3")

	assert.Equal(t, d.size(), 2)
	assert.Equal(t, d.Routes[0].Id, "1")
	assert.Equal(t, d.Upstreams[0].Id, "2")

	assert.Equal(t, u.size(), 1)
	assert.Equal(t, u.Routes[0].Id, "2")
	assert.Equal(t, u.Routes[0].Uris, []string{"/foo"})
}

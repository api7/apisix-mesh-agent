package sidecar

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/cache"
	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestReflectToCache(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.XDSWatchFiles = append(cfg.XDSWatchFiles, "/tmp")
	cfg.GRPCListen = "127.0.0.1:10001"
	s, err := NewSidecar(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, s)

	events := []types.Event{
		{
			Type: types.EventAdd,
			Object: &apisix.Route{
				Id: "1",
			},
		},
		{
			Type: types.EventAdd,
			Object: &apisix.Route{
				Id: "2",
			},
		},
		{
			Type: types.EventUpdate,
			Object: &apisix.Upstream{
				Id: "133",
			},
		},
		{
			Type: types.EventDelete,
			Tombstone: &apisix.Upstream{
				Id: "21",
			},
		},
	}
	err = s.cache.Upstream().Insert(&apisix.Upstream{Id: "21"})
	assert.Nil(t, err)
	s.reflectToCache(events)
	r1, err := s.cache.Route().Get("1")
	assert.NotNil(t, r1)
	assert.Nil(t, err)

	r2, err := s.cache.Route().Get("2")
	assert.NotNil(t, r2)
	assert.Nil(t, err)

	u1, err := s.cache.Upstream().Get("133")
	assert.NotNil(t, u1)
	assert.Nil(t, err)

	u2, err := s.cache.Upstream().Get("21")
	assert.Nil(t, u2)
	assert.Equal(t, err, cache.ErrObjectNotFound)
}

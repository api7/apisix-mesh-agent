package apisix

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestCompareRoutes(t *testing.T) {
	r1 := []*apisix.Route{
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "1"},
			},
		},
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "2"},
			},
		},
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "3"},
			},
		},
	}

	added, deleted, updated := CompareRoutes(r1, nil)
	assert.Nil(t, added)
	assert.Nil(t, updated)
	assert.Equal(t, deleted, r1)

	added, deleted, updated = CompareRoutes(nil, r1)
	assert.Equal(t, added, r1)
	assert.Nil(t, updated)
	assert.Nil(t, deleted)

	r2 := []*apisix.Route{
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "1"},
			},
		},
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "4"},
			},
		},
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "3"},
			},
			Uris: []string{"/foo*"},
		},
	}

	added, deleted, updated = CompareRoutes(r1, r2)
	assert.Equal(t, added, []*apisix.Route{
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "4"},
			},
		},
	})
	assert.Equal(t, deleted, []*apisix.Route{
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "2"},
			},
		},
	})
	assert.Equal(t, updated[0].Id.GetStrVal(), "3")
	assert.Equal(t, updated[0].Uris, []string{"/foo*"})
}

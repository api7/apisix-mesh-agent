package apisix

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestCompareUpstreams(t *testing.T) {
	u1 := []*apisix.Upstream{
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

	added, deleted, updated := CompareUpstreams(u1, nil)
	assert.Nil(t, added)
	assert.Nil(t, updated)
	assert.Equal(t, deleted, u1)

	added, deleted, updated = CompareUpstreams(nil, u1)
	assert.Equal(t, added, u1)
	assert.Nil(t, updated)
	assert.Nil(t, deleted)

	u2 := []*apisix.Upstream{
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
			Retries: 3,
		},
	}

	added, deleted, updated = CompareUpstreams(u1, u2)
	assert.Equal(t, added, []*apisix.Upstream{
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "4"},
			},
		},
	})
	assert.Equal(t, deleted, []*apisix.Upstream{
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{StrVal: "2"},
			},
		},
	})
	assert.Equal(t, updated[0].Id.GetStrVal(), "3")
	assert.Equal(t, updated[0].Retries, int32(3))
}

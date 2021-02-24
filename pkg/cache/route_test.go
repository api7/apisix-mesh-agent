package cache

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func TestRoute(t *testing.T) {
	r := newRoute()
	assert.NotNil(t, r)

	// Not found
	obj, err := r.Get("1")
	assert.Nil(t, obj)
	assert.Equal(t, err, ErrObjectNotFound)
	assert.Equal(t, r.Delete("1"), ErrObjectNotFound)

	route1 := &apisix.Route{
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "1",
			},
		},
	}
	assert.Nil(t, r.Insert(route1))

	obj, err = r.Get("1")
	assert.Nil(t, err)
	assert.Equal(t, obj.Id.GetStrVal(), "1")

	// Update
	obj.Name = "Vivian"
	assert.Nil(t, r.Insert(obj))
	obj, err = r.Get("1")
	assert.Nil(t, err)
	assert.Equal(t, obj.Id.GetStrVal(), "1")
	assert.Equal(t, obj.GetName(), "Vivian")

	// Delete
	assert.Nil(t, r.Delete("1"))
	assert.Equal(t, r.Delete("1"), ErrObjectNotFound)
	obj, err = r.Get("1")
	assert.Nil(t, obj)
	assert.Error(t, err, ErrObjectNotFound)
}

func TestRouteList(t *testing.T) {
	objs := []*apisix.Route{
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{
					StrVal: "1",
				},
			},
		},
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{
					StrVal: "2",
				},
			},
		},
		{
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{
					StrVal: "3",
				},
			},
		},
	}
	r := newRoute()
	assert.NotNil(t, r)
	for _, obj := range objs {
		assert.Nil(t, r.Insert(obj))
	}
	list, err := r.List()
	assert.Nil(t, err)
	assert.Len(t, list, 3)

	var ids []string
	for _, elem := range list {
		ids = append(ids, elem.GetId().GetStrVal())
	}
	sort.Strings(ids)
	assert.Equal(t, ids[0], "1")
	assert.Equal(t, ids[1], "2")
	assert.Equal(t, ids[2], "3")
}

func TestRouteObjectClone(t *testing.T) {
	route1 := &apisix.Route{
		Id: &apisix.ID{
			OneofId: &apisix.ID_StrVal{
				StrVal: "1",
			},
		},
	}
	r := newRoute()
	assert.NotNil(t, r)
	assert.Nil(t, r.Insert(route1))

	obj, err := r.Get("1")
	assert.Nil(t, err)

	obj.Name = "alex"
	obj, err = r.Get("1")
	assert.Nil(t, err)
	assert.Equal(t, obj.Name, "")
}

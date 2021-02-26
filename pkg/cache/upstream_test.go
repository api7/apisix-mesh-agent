package cache

import (
	"sort"
	"testing"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"

	"github.com/stretchr/testify/assert"
)

func TestUpstream(t *testing.T) {
	u := newUpstream()
	assert.NotNil(t, u)

	// Not found
	obj, err := u.Get("1")
	assert.Nil(t, obj)
	assert.Equal(t, err, ErrObjectNotFound)
	assert.Equal(t, u.Delete("1"), ErrObjectNotFound)

	ups1 := &apisix.Upstream{
		Id: "1",
	}
	assert.Nil(t, u.Insert(ups1))

	obj, err = u.Get("1")
	assert.Nil(t, err)
	assert.Equal(t, obj.Id, "1")

	// Update
	obj.Name = "Vivian"
	assert.Nil(t, u.Insert(obj))
	obj, err = u.Get("1")
	assert.Nil(t, err)
	assert.Equal(t, obj.Id, "1")
	assert.Equal(t, obj.GetName(), "Vivian")

	// Delete
	assert.Nil(t, u.Delete("1"))
	assert.Equal(t, u.Delete("1"), ErrObjectNotFound)
	obj, err = u.Get("1")
	assert.Nil(t, obj)
	assert.Error(t, err, ErrObjectNotFound)
}

func TestUpstreamList(t *testing.T) {
	objs := []*apisix.Upstream{
		{
			Id: "1",
		},
		{
			Id: "2",
		},
		{
			Id: "3",
		},
	}
	u := newUpstream()
	assert.NotNil(t, u)
	for _, obj := range objs {
		assert.Nil(t, u.Insert(obj))
	}
	list, err := u.List()
	assert.Nil(t, err)
	assert.Len(t, list, 3)

	var ids []string
	for _, elem := range list {
		ids = append(ids, elem.GetId())
	}
	sort.Strings(ids)
	assert.Equal(t, ids[0], "1")
	assert.Equal(t, ids[1], "2")
	assert.Equal(t, ids[2], "3")
}

func TestUpstreamObjectClone(t *testing.T) {
	ups1 := &apisix.Upstream{
		Id: "1",
	}
	u := newUpstream()
	assert.NotNil(t, u)
	assert.Nil(t, u.Insert(ups1))

	obj, err := u.Get("1")
	assert.Nil(t, err)

	obj.Name = "alex"
	obj, err = u.Get("1")
	assert.Nil(t, err)
	assert.Equal(t, obj.Name, "")
}

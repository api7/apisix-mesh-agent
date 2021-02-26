package apisix

import (
	"google.golang.org/protobuf/proto"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

// CompareUpstreams diffs two apisix.Upstreams array and finds the new adds, updates
// and deleted ones. Note it stands on the first apisix.Upstream array's point
// of view.
func CompareUpstreams(u1, u2 []*apisix.Upstream) (added, deleted, updated []*apisix.Upstream) {
	if u1 == nil {
		return u2, nil, nil
	}
	if u2 == nil {
		return nil, u1, nil
	}
	u1Map := make(map[string]*apisix.Upstream)
	u2Map := make(map[string]*apisix.Upstream)
	for _, u := range u1 {
		u1Map[u.GetId()] = u
	}
	for _, u := range u2 {
		u2Map[u.GetId()] = u
	}
	for _, u := range u2 {
		if _, ok := u1Map[u.GetId()]; !ok {
			added = append(added, u)
		}
	}
	for _, uo := range u1 {
		if un, ok := u2Map[uo.GetId()]; !ok {
			deleted = append(deleted, uo)
		} else {
			if !proto.Equal(uo, un) {
				updated = append(updated, un)
			}
		}
	}
	return
}

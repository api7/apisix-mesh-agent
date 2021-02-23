package apisix

import (
	"google.golang.org/protobuf/proto"

	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

// CompareRoutes diffs two apisix.Route array and finds the new adds, updates
// and deleted ones. Note it stands on the first apisix.Route array's point
// of view.
func CompareRoutes(r1, r2 []*apisix.Route) (added, deleted, updated []*apisix.Route) {
	if r1 == nil {
		return r2, nil, nil
	}
	if r2 == nil {
		return nil, r1, nil
	}

	r1Map := make(map[string]*apisix.Route)
	r2Map := make(map[string]*apisix.Route)
	for _, r := range r1 {
		r1Map[r.Id.GetStrVal()] = r
	}
	for _, r := range r2 {
		r2Map[r.Id.GetStrVal()] = r
	}
	for _, r := range r2 {
		if _, ok := r1Map[r.Id.GetStrVal()]; !ok {
			added = append(added, r)
		}
	}
	for _, ro := range r1 {
		if rn, ok := r2Map[ro.Id.GetStrVal()]; !ok {
			deleted = append(deleted, ro)
		} else {
			if !proto.Equal(ro, rn) {
				updated = append(updated, rn)
			}
		}
	}
	return
}

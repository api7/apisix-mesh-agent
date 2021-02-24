package file

import (
	apisixutil "github.com/api7/apisix-mesh-agent/pkg/apisix"
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

type manifest struct {
	// Fields are named to be exportable so they can
	// be shown by logger.
	Routes    []*apisix.Route
	Upstreams []*apisix.Upstream
}

// diffFrom checks the difference between m and m2 from m's point of view.
func (m *manifest) diffFrom(m2 *manifest) (*manifest, *manifest, *manifest) {
	var (
		added   manifest
		updated manifest
		deleted manifest
	)

	a, d, u := apisixutil.CompareRoutes(m.Routes, m2.Routes)
	added.Routes = append(added.Routes, a...)
	updated.Routes = append(updated.Routes, u...)
	deleted.Routes = append(deleted.Routes, d...)

	au, du, uu := apisixutil.CompareUpstreams(m.Upstreams, m2.Upstreams)
	added.Upstreams = append(added.Upstreams, au...)
	updated.Upstreams = append(updated.Upstreams, uu...)
	deleted.Upstreams = append(deleted.Upstreams, du...)

	return &added, &deleted, &updated
}

func (m *manifest) size() int {
	return len(m.Upstreams) + len(m.Routes)
}

func (m *manifest) events(evType types.EventType) []types.Event {
	var events []types.Event
	for _, r := range m.Routes {
		if evType == types.EventDelete {
			events = append(events, types.Event{
				Type:      types.EventDelete,
				Tombstone: r,
			})
		} else {
			events = append(events, types.Event{
				Type:   evType,
				Object: r,
			})
		}
	}
	for _, u := range m.Upstreams {
		if evType == types.EventDelete {
			events = append(events, types.Event{
				Type:      types.EventDelete,
				Tombstone: u,
			})
		} else {
			events = append(events, types.Event{
				Type:   evType,
				Object: u,
			})
		}
	}
	return events
}

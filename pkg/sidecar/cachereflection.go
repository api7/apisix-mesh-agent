package sidecar

import (
	"errors"

	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

var (
	_errUnknownEventObject = errors.New("unknown event object type")
)

func (s *Sidecar) reflectToCache(events []types.Event) {
	for _, ev := range events {
		var err error
		switch ev.Type {
		case types.EventAdd, types.EventUpdate:
			switch obj := ev.Object.(type) {
			case *apisix.Route:
				s.logger.Debugw("insert route cache",
					zap.Any("route", obj),
					zap.String("event", string(ev.Type)),
				)
				err = s.cache.Route().Insert(obj)
			case *apisix.Upstream:
				s.logger.Debugw("insert upstream cache",
					zap.Any("upstream", obj),
					zap.String("event", string(ev.Type)),
				)
				err = s.cache.Upstream().Insert(obj)
			default:
				err = _errUnknownEventObject
			}
		default: // types.EventDelete
			switch obj := ev.Tombstone.(type) {
			case *apisix.Route:
				s.logger.Debugw("delete route cache",
					zap.Any("route", obj),
					zap.String("event", string(ev.Type)),
				)
				err = s.cache.Route().Delete(obj.GetId().GetStrVal())
			case *apisix.Upstream:
				s.logger.Debugw("delete upstream cache",
					zap.Any("upstream", obj),
					zap.String("event", string(ev.Type)),
				)
				err = s.cache.Upstream().Delete(obj.GetId().GetStrVal())
			default:
				err = _errUnknownEventObject
			}
		}
		if err != nil {
			s.logger.Errorw("failed to reflect event to cache",
				zap.Any("event", ev),
				zap.Error(err),
			)
		}
	}
}

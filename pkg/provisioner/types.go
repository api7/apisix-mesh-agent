package provisioner

import (
	"github.com/api7/apisix-mesh-agent/pkg/types"
)

// Provisioner provisions config event.
// The source type can be xDS or UDPA or whatever anything else.
type Provisioner interface {
	// Channel returns a readonly channel where caller can get events.
	Channel() <-chan []types.Event
	// Run launches the provisioner.
	Run(chan struct{}) error
}

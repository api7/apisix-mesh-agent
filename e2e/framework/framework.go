package framework

import "github.com/api7/apisix-mesh-agent/e2e/framework/controlplane"

// Framework is the framework of apisix-mesh-agent e2e tests.
type Framework struct {
	cp controlplane.ControlPlane
}

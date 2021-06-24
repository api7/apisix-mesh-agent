package e2e

import (
	"os"
	"testing"

	"github.com/onsi/ginkgo"

	_ "github.com/api7/apisix-mesh-agent/e2e/suites"
	_ "github.com/api7/apisix-mesh-agent/e2e/suites/nacos"
)

// TestE2ESuites is the entry of apisix-mesh-agent e2e suites.
func TestE2ESuites(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("APISIX_MESH_AGENT_E2E_HOME", pwd); err != nil {
		panic(err)
	}

	ginkgo.RunSpecs(t, "apisix-mesh-agent e2e test cases")
}

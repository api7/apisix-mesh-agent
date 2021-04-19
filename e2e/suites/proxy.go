package suites

import (
	"fmt"

	"github.com/api7/apisix-mesh-agent/e2e/framework"
	"github.com/onsi/ginkgo"
)

var _ = ginkgo.Describe("basic functions", func() {
	framework.NewDefaultFramework()
	ginkgo.It("test 1", func() {
		fmt.Println("123")
	})
})

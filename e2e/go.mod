module github.com/api7/apisix-mesh-agent/e2e

go 1.16

replace github.com/api7/apisix-mesh-agent => ../

require (
	github.com/api7/apisix-mesh-agent v0.0.0-00010101000000-000000000000
	github.com/gruntwork-io/terratest v0.32.15
	github.com/onsi/ginkgo v1.14.2
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
)

module github.com/api7/apisix-mesh-agent/e2e

go 1.16

replace github.com/api7/apisix-mesh-agent => ../

require (
	github.com/api7/apisix-mesh-agent v0.0.0-00010101000000-000000000000
	github.com/gavv/httpexpect/v2 v2.2.0
	github.com/gruntwork-io/terratest v0.32.15
	github.com/nacos-group/nacos-sdk-go v1.0.8
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.10.5
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20210608053332-aa57babbf139 // indirect
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
)

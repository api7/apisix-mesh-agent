package suites

import (
	"fmt"
	"net/http"

	"github.com/onsi/gomega"

	"github.com/api7/apisix-mesh-agent/e2e/framework"
	"github.com/onsi/ginkgo"
)

var _ = ginkgo.Describe("[basic proxy functions]", func() {
	g := gomega.NewWithT(ginkgo.GinkgoT())
	f, err := framework.NewDefaultFramework()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	ginkgo.It("nginx -> httpbin", func() {
		template := `
	server {
	listen 80;
	server_name httpbin.org;
	location / {
		proxy_pass http://%s;
		proxy_set_header Host %s;
		proxy_http_version 1.1;
		proxy_set_header Connection "";
	}
}
`
		fqdn := f.GetHttpBinServiceFQDN()
		snippet := fmt.Sprintf(template, fqdn, fqdn)
		g.Expect(f.CreateConfigMap("nginx-httpbin", "httpbin.conf", snippet)).ShouldNot(gomega.HaveOccurred())
		g.Expect(f.DeployNginxWithConfigMapVolume("nginx-httpbin")).ShouldNot(gomega.HaveOccurred())
		g.Expect(f.DeploySpringboardWithSpecificProxyTarget("nginx")).ShouldNot(gomega.HaveOccurred())

		expect, err := f.NewHTTPClientToSpringboard()
		g.Expect(err).ShouldNot(gomega.HaveOccurred())

		resp := expect.GET("/ip").WithHeader("Host", fqdn).Expect()
		// Hit the default route the cluster outbound|80||httpbin.<namespace>.svc.cluster.local
		resp.Status(http.StatusOK)
		// The first Via header was added by nginx's sidecar;
		// The second Via header was added by httpbin's sidecar;
		resp.Headers().Value("Via").Array().Equal([]string{"APISIX", "APISIX"})
		resp.Body().Contains("origin")
	})
})

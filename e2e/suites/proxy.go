package suites

import (
	"fmt"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/api7/apisix-mesh-agent/e2e/framework"
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

		time.Sleep(time.Second * 10)
		resp := expect.GET("/ip").WithHeader("Host", fqdn).Expect()
		if resp.Raw().StatusCode != http.StatusOK {
			ginkgo.GinkgoT().Log("status code is %v, please check logs", resp.Raw().StatusCode)
			time.Sleep(time.Hour * 1000)
		}
		// Hit the default route the cluster outbound|80||httpbin.<namespace>.svc.cluster.local
		resp.Status(http.StatusOK)
		// The first Via header was added by nginx's sidecar;
		// The second Via header was added by httpbin's sidecar;
		resp.Headers().Value("Via").Array().Equal([]string{"APISIX", "APISIX"})
		resp.Body().Contains("origin")
	})
})

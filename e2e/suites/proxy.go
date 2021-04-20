package suites

import (
	"fmt"

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

		expect, err := f.NewHTTPClientToNginxService()
		g.Expect(err).ShouldNot(gomega.HaveOccurred())

		resp := expect.GET("/status/200").WithHeader("Host", fqdn).Expect()
		resp.Status(404)
		// Request was terminated by APISIX itself.
		resp.Header("Server").Contains("APISIX")
		resp.Body().Contains("404 Route Not Found")

		vs := `
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: httpbin
spec:
  hosts:
  - %s
  http:
  - name: "httpbin-route"
    route:
    - destination:
        host: %s
`
		err = f.CreateResourceFromString(fmt.Sprintf(vs, fqdn, fqdn))
		g.Expect(err).ShouldNot(gomega.HaveOccurred())

		resp = expect.GET("/ip").WithHeader("Host", fqdn).Expect()
		resp.Status(200)
		// Inbound APISIX will add a via header which value is APISIX.
		// As we use the tunnel to access nginx, traffic won't be intercepted to the inbound APISIX.
		// So here only one Via header will be appended.
		resp.Header("Via").Equal("APISIX")
		resp.Body().Contains("origin")
	})
})

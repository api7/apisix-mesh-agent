package nacos

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/api7/apisix-mesh-agent/e2e/framework"
)

var _ = ginkgo.Describe("[nacos provisioner basic proxy functions]", func() {
	g := gomega.NewWithT(ginkgo.GinkgoT())

	var (
		f     *framework.Framework
		nacos *framework.NacosInstallation
		err   error
	)

	if os.Getenv("GITHUB_ACTIONS") == "" {
		f, err = framework.NewNacosFramework()
		g.Expect(err).ShouldNot(gomega.HaveOccurred())

		nacos, err = framework.NewNacos()
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
	}
	// Clear configs
	ginkgo.BeforeEach(func() {
		if os.Getenv("GITHUB_ACTIONS") != "" {
			ginkgo.Skip("Skipped in GitHub Actions")
		}

		err = nacos.DeleteConfig(&framework.NacosConfig{
			DataId: "cfg.routes",
		})
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = nacos.DeleteConfig(&framework.NacosConfig{
			DataId: "cfg.upstreams",
		})
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.AfterEach(func() {
		err = nacos.ClearAllConfig()
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.AfterSuite(func() {
		if !nacos.Preinstalled {
			err = nacos.Uninstall()
			g.Expect(err).ShouldNot(gomega.HaveOccurred())
		}
	})

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
		route := fmt.Sprintf(`[{
	"id": "1",
	"methods": ["GET"],
	"hosts": ["%v"],
	"uris": ["/*"],
	"upstream_id": "1"
}]`, fqdn)

		upstream := fmt.Sprintf(`[{
	"id": "1",
	"type": "roundrobin",
	"nodes": [{
		"host": "%v",
		"port": 80,
		"weight": 1
	}]
}]`, fqdn)

		snippet := fmt.Sprintf(template, fqdn, fqdn)
		g.Expect(f.CreateConfigMap("nginx-httpbin", "httpbin.conf", snippet)).ShouldNot(gomega.HaveOccurred())
		g.Expect(f.DeployNginxWithConfigMapVolume("nginx-httpbin")).ShouldNot(gomega.HaveOccurred())
		g.Expect(f.DeploySpringboardWithSpecificProxyTarget("nginx")).ShouldNot(gomega.HaveOccurred())

		expect, err := f.NewHTTPClientToSpringboard()
		g.Expect(err).ShouldNot(gomega.HaveOccurred())

		resp := expect.GET("/ip").WithHeader("Host", fqdn).Expect()
		// Hit the default route the cluster outbound|80||httpbin.<namespace>.svc.cluster.local
		resp.Status(http.StatusNotFound)

		err = nacos.PublishConfig(&framework.NacosConfig{
			DataId:  "cfg.routes",
			Content: route,
		})
		g.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = nacos.PublishConfig(&framework.NacosConfig{
			DataId:  "cfg.upstreams",
			Content: upstream,
		})
		g.Expect(err).ShouldNot(gomega.HaveOccurred())

		time.Sleep(time.Second * 3) // Wait for update

		resp = expect.GET("/ip").WithHeader("Host", fqdn).Expect()
		// Hit the default route the cluster outbound|80||httpbin.<namespace>.svc.cluster.local
		resp.Status(http.StatusOK)

		// The first Via header was added by nginx's sidecar;
		// The second Via header was added by httpbin's sidecar;
		resp.Headers().Value("Via").Array().Equal([]string{"APISIX", "APISIX"})
		resp.Body().Contains("origin")
	})
})

package framework

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/api7/apisix-mesh-agent/e2e/framework/controlplane"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Framework is the framework of apisix-mesh-agent e2e tests.
type Framework struct {
	opts        *Options
	namespace   string
	cp          controlplane.ControlPlane
	kubectlOpts *k8s.KubectlOptions

	// Public arguments to render manifests.
	HttpBinReplicas int
	NginxReplicas   int
	LocalRegistry   string
}

// Options contains options to customize the e2d framework.
type Options struct {
	Kubeconfig string
	// Control Plane type
	ControlPlane      string
	ControlPlaneImage string
	// The Helm Charts path to install the Control Plane.
	ControlPlaneChartsPath []string
}

func init() {
	gomega.RegisterFailHandler(ginkgo.Fail)
}

// NewDefaultFramework creates the framework with default options
func NewDefaultFramework() (*Framework, error) {
	e2eHome := os.Getenv("APISIX_MESH_AGENT_E2E_HOME")
	opts := &Options{
		Kubeconfig:        GetKubeconfig(),
		ControlPlane:      "istio",
		ControlPlaneImage: "localhost:5000/istio/pilot:1.9.1",

		ControlPlaneChartsPath: []string{
			filepath.Join(e2eHome, "charts/istio/base"),
			filepath.Join(e2eHome, "charts/istio/istio-discovery"),
		},
	}
	return NewFramework(opts)
}

// NewFramework creates the framework with the given options.
func NewFramework(opts *Options) (*Framework, error) {
	ns := randomizeNamespace()
	fw := &Framework{
		namespace: ns,
		kubectlOpts: &k8s.KubectlOptions{
			ConfigPath: opts.Kubeconfig,
			Namespace:  ns,
		},
		opts: opts,

		HttpBinReplicas: 1,
		NginxReplicas:   1,
		LocalRegistry:   "localhost:5000",
	}
	if len(opts.ControlPlaneChartsPath) == 0 {
		return nil, errors.New("no specific control plane charts")
	}
	switch opts.ControlPlane {
	case "istio":
		istioOpts := &controlplane.IstioOptions{
			IstioImage: opts.ControlPlaneImage,
			Kubeconfig: opts.Kubeconfig,
			Namespace:  fw.namespace,
			ChartsPath: opts.ControlPlaneChartsPath,
		}
		cp, err := controlplane.NewIstioControlPlane(istioOpts)
		if err != nil {
			return nil, err
		}
		fw.cp = cp
	default:
		return nil, errors.New("unknown control plane")
	}

	ginkgo.BeforeEach(fw.beforeEach)
	ginkgo.AfterEach(fw.afterEach)
	return fw, nil
}

// GetKubeconfig returns the kubeconfig file path.
// Order:
// env KUBECONFIG;
// ~/.kube/config;
// "" (in case in-cluster configuration will be used).
func GetKubeconfig() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		kubeconfig = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(kubeconfig); err != nil && !os.IsNotExist(err) {
			kubeconfig = ""
		}
	}
	return kubeconfig
}

func (f *Framework) deploy() {
	gomega.NewGomegaWithT(ginkgo.GinkgoT()).Expect(f.cp.Deploy()).ShouldNot(gomega.HaveOccurred())
	gomega.NewGomegaWithT(ginkgo.GinkgoT()).Expect(f.newHttpBin()).ShouldNot(gomega.HaveOccurred())
}

func randomizeNamespace() string {
	return fmt.Sprintf("apisix-mesh-agent-e2e-%d", time.Now().Nanosecond())
}

func (f *Framework) beforeEach() {
	err := k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace)
	gomega.NewGomegaWithT(ginkgo.GinkgoT()).Expect(err).ShouldNot(gomega.HaveOccurred())
	f.deploy()
}

func (f *Framework) afterEach() {
	err := k8s.DeleteNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace)
	gomega.NewGomegaWithT(ginkgo.GinkgoT()).Expect(err).ShouldNot(gomega.HaveOccurred())
}

func (f *Framework) renderManifest(manifest string) (string, error) {
	temp, err := template.New("manifest").Parse(manifest)
	if err != nil {
		return "", err
	}

	artifact := new(strings.Builder)
	if err := temp.Execute(artifact, f); err != nil {
		return "", err
	}
	return artifact.String(), nil
}

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Steps:    8,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}

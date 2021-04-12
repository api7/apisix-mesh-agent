package framework

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/api7/apisix-mesh-agent/e2e/framework/controlplane"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
)

// Framework is the framework of apisix-mesh-agent e2e tests.
type Framework struct {
	cpNamespace string
	namespace   string
	cp          controlplane.ControlPlane
	kubectlOpts *k8s.KubectlOptions
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

// NewDefaultFramework creates the framework with default options
func NewDefaultFramework() (*Framework, error) {
	opts := &Options{
		Kubeconfig:        GetKubeconfig(),
		ControlPlane:      "istio",
		ControlPlaneImage: "istio/pilot:1.9.1",

		ControlPlaneChartsPath: []string{
			"../charts/istio/base",
			"../charts/istio/istio-discovery",
		},
	}
	return NewFramework(opts)
}

// NewFramework creates the framework with the given options.
func NewFramework(opts *Options) (*Framework, error) {
	fw := &Framework{
		namespace:   randomizeNamespace(),
		cpNamespace: randomizeCPNamespace(),
		kubectlOpts: &k8s.KubectlOptions{
			ConfigPath: opts.Kubeconfig,
			Namespace:  "",
			Env:        nil,
		},
	}
	if len(opts.ControlPlaneChartsPath) == 0 {
		return nil, errors.New("no specific control plane charts")
	}
	switch opts.ControlPlane {
	case "istio":
		istioOpts := &controlplane.IstioOptions{
			IstioImage: opts.ControlPlaneImage,
			Kubeconfig: opts.Kubeconfig,
			Namespace:  fw.cpNamespace,
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

// Deploy deployes all components in the framework.
func (f *Framework) Deploy() error {
	// Create CP namespace
	if err := k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.cpNamespace); err != nil {
		return err
	}
	if err := f.cp.Deploy(); err != nil {
		return err
	}
	if err := k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace); err != nil {
		return err
	}
	return nil
}

func randomizeNamespace() string {
	return fmt.Sprintf("apisix-mesh-agent-e2e-%d", time.Now().Nanosecond())
}

func randomizeCPNamespace() string {
	return fmt.Sprintf("apisix-mesh-agent-e2e-cp-%d", time.Now().Nanosecond())
}

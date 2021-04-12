package controlplane

import (
	"bytes"
	"errors"
	"os/exec"
	"time"

	"github.com/onsi/ginkgo"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/api7/apisix-mesh-agent/pkg/log"
)

var (
	_helm              = "helm"
	_defaultKubeconfig = "~/.kube/config"
)

type istio struct {
	base                   *exec.Cmd
	discovery              *exec.Cmd
	cleanupBase            *exec.Cmd
	cleanupDiscovery       *exec.Cmd
	baseStderr             *bytes.Buffer
	cleanupBaseStderr      *bytes.Buffer
	discoveryStderr        *bytes.Buffer
	cleanupDiscoveryStderr *bytes.Buffer

	logger *log.Logger

	options     *IstioOptions
	installCmds [][]string
	clusterIP   string
}

// IstioOptions contains options to customize Istio control plane.
type IstioOptions struct {
	// IstioImage is the image of Istiod, should be in:
	// <registry>/<repository>:<tag> format.
	IstioImage string
	// Namespace is the target namespace to install Istiod.
	Namespace string
	// ChartsPath is a directory that contains charts for Istio.
	// The first element should be the chart for istio-base and
	// the second is the istio-control.
	ChartsPath []string
	// Kubeconfig is the kube config file path.
	Kubeconfig string
}

// NewIstioControlPlane creates an istio control plane.
func NewIstioControlPlane(opts *IstioOptions) (ControlPlane, error) {
	if opts.Kubeconfig == "" {
		opts.Kubeconfig = _defaultKubeconfig
	}
	kc := opts.Kubeconfig
	image := "istio/pilot:1.9.1"
	if opts.IstioImage != "" {
		image = opts.IstioImage
	}
	if opts.Namespace == "" {
		return nil, errors.New("unspecific namespace")
	}
	base := exec.Command(_helm,
		"install", "istio-base", "--namespace", opts.Namespace, "--kubeconfig", kc,
		"--set", "pilot.image="+image, "--set", "global.istioNamespace="+opts.Namespace,
		opts.ChartsPath[0])
	cleanupBase := exec.Command(_helm, "uninstall", "istio-base", "--namespace", opts.Namespace, "--kubeconfig", kc)
	discovery := exec.Command(_helm, "install", "istio-control", "--namespace", opts.Namespace, "--kubeconfig", kc, opts.ChartsPath[1])
	cleanupDiscovery := exec.Command(_helm, "uninstall", "istio-control", "--namespace", opts.Namespace, "--kubeconfig", kc)

	baseStderr := bytes.NewBuffer(nil)
	cleanupBaseStderr := bytes.NewBuffer(nil)
	discoveryStderr := bytes.NewBuffer(nil)
	cleanupDiscoveryStderr := bytes.NewBuffer(nil)

	base.Stderr = baseStderr
	cleanupBase.Stderr = cleanupBaseStderr
	discovery.Stderr = discoveryStderr
	cleanupDiscovery.Stderr = cleanupDiscoveryStderr

	logger, err := log.NewLogger(
		log.WithContext("istio"),
		log.WithLogLevel("error"),
	)
	if err != nil {
		return nil, err
	}

	return &istio{
		logger:                 logger,
		base:                   base,
		discovery:              discovery,
		cleanupBase:            cleanupBase,
		cleanupDiscovery:       cleanupDiscovery,
		options:                opts,
		baseStderr:             baseStderr,
		cleanupBaseStderr:      cleanupBaseStderr,
		discoveryStderr:        discoveryStderr,
		cleanupDiscoveryStderr: cleanupDiscoveryStderr,
	}, nil
}

func (cp *istio) Namespace() string {
	return cp.options.Namespace
}

func (cp *istio) Type() string {
	return "istio"
}

func (cp *istio) Addr() string {
	return "grpc://" + cp.clusterIP + ":15010"
}

func (cp *istio) Deploy() error {
	err := cp.base.Run()
	if err != nil {
		log.Errorw("failed to run istio-base install command",
			zap.String("command", cp.base.String()),
			zap.Error(err),
			zap.String("stderr", cp.baseStderr.String()),
		)
		return err
	}
	err = cp.discovery.Run()
	if err != nil {
		log.Errorw("failed to run istio-control install command",
			zap.String("command", cp.discovery.String()),
			zap.String("stderr", cp.discoveryStderr.String()),
		)
		return err
	}

	ctlOpts := &k8s.KubectlOptions{
		ConfigPath: cp.options.Kubeconfig,
		Namespace:  cp.options.Namespace,
	}

	var (
		svc *corev1.Service
	)

	condFunc := func() (bool, error) {
		svc, err = k8s.GetServiceE(ginkgo.GinkgoT(), ctlOpts, "istiod")
		if err != nil {
			return false, err
		}
		return k8s.IsServiceAvailable(svc), nil
	}

	if err := wait.PollImmediate(3*time.Second, 15*time.Second, condFunc); err != nil {
		return err
	}

	cp.clusterIP = svc.Spec.ClusterIP
	return nil
}

func (cp *istio) Uninstall() error {
	err := cp.cleanupBase.Run()
	if err != nil {
		log.Errorw("failed to uninstall istio-base",
			zap.Error(err),
			zap.String("stderr", cp.cleanupBaseStderr.String()),
		)
		return err
	}
	err = cp.cleanupDiscovery.Run()
	if err != nil {
		log.Errorw("failed to uninstall istio-control",
			zap.Error(err),
			zap.String("stderr", cp.cleanupDiscoveryStderr.String()),
		)
		return err
	}
	return nil
}

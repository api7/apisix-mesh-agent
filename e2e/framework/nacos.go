package framework

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/onsi/ginkgo"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	e2econst "github.com/api7/apisix-mesh-agent/e2e/framework/constant"
	"github.com/api7/apisix-mesh-agent/pkg/log"
)

func nacosNamespace() string {
	return "nacos-e2e"
}

type NacosConfig struct {
	DataId  string
	Content string
}

type NacosInstallation struct {
	Preinstalled bool
	KubectlOpts  *k8s.KubectlOptions

	Namespace   string
	ChartPath   string
	ReleaseName string

	Tunnel       *k8s.Tunnel
	Endpoint     string
	ConfigClient config_client.IConfigClient

	mu   sync.Mutex
	cfgs map[string]*NacosConfig
}

// NewNacos installs nacos
func NewNacos() (*NacosInstallation, error) {
	nacos := &NacosInstallation{
		KubectlOpts: &k8s.KubectlOptions{
			ConfigPath: GetKubeconfig(),
			Namespace:  nacosNamespace(),
		},
		Namespace:   nacosNamespace(),
		ChartPath:   filepath.Join(e2econst.E2eHome, "charts/nacos"),
		ReleaseName: "nacos-e2e",
		cfgs:        make(map[string]*NacosConfig),
	}

	err := k8s.CreateNamespaceE(ginkgo.GinkgoT(), nacos.KubectlOpts, nacos.Namespace)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Errorw("failed to create namespace nacos",
				zap.Error(err),
			)
			return nil, err
		}
	}

	_, err = k8s.GetPodE(ginkgo.GinkgoT(), nacos.KubectlOpts, nacos.ReleaseName+"-0")
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}

		nacos.Preinstalled = false
		// install
		install := exec.Command(e2econst.Helm, "install", nacos.ReleaseName,
			"--namespace", nacos.Namespace, "--kubeconfig", GetKubeconfig(), nacos.ChartPath)
		installErr := bytes.NewBuffer(nil)
		install.Stderr = installErr
		err = install.Run()
		if err != nil {
			log.Errorw("failed to run nacos install command",
				zap.String("command", install.String()),
				zap.Error(err),
				zap.String("stderr", installErr.String()),
			)
			return nil, err
		}
	} else {
		nacos.Preinstalled = true
	}

	condFunc := func() (bool, error) {
		pod, err := k8s.GetPodE(ginkgo.GinkgoT(), nacos.KubectlOpts, nacos.ReleaseName+"-0")
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, err
			} else {
				log.Errorw("failed to get nacos pod",
					zap.Error(err),
				)
			}
		}
		return k8s.IsPodAvailable(pod), nil
	}

	time.Sleep(time.Second * 2) // Wait for pod creation
	if err := wait.PollImmediate(5*time.Second, 3*time.Minute, condFunc); err != nil {
		log.Errorw("failed to wait nacos pod available",
			zap.Error(err),
		)
		return nil, err
	}

	tunnel := k8s.NewTunnel(nacos.KubectlOpts, k8s.ResourceTypeService, "nacos-cs", 18848, 8848)
	if err := tunnel.ForwardPortE(ginkgo.GinkgoT()); err != nil {
		log.Errorw("failed to create nacos tunnel",
			zap.Error(err),
		)
		return nil, err
	}
	nacos.Tunnel = tunnel
	nacos.Endpoint = tunnel.Endpoint()

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      "localhost",
			ContextPath: "/nacos",
			Port:        18848,
			Scheme:      "http",
		},
	}
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ServerConfigs: serverConfigs,
		},
	)
	nacos.ConfigClient = configClient

	return nacos, nil
}

// PublishConfig publishes config to nacos
func (n *NacosInstallation) PublishConfig(cfg *NacosConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	_, err := n.ConfigClient.PublishConfig(vo.ConfigParam{
		DataId:  cfg.DataId,
		Group:   "org.apache.apisix",
		Content: cfg.Content,
	})
	if err != nil {
		return err
	}
	n.cfgs[cfg.DataId] = cfg
	return nil
}

// CheckConfig checks if existed config matches target
func (n *NacosInstallation) CheckConfig(target *NacosConfig) error {
	got, err := n.ConfigClient.GetConfig(vo.ConfigParam{
		DataId: target.DataId,
		Group:  "org.apache.apisix",
	})
	if err != nil {
		return err
	}
	if got != target.Content {
		return errors.New(fmt.Sprintf("nacos config %v mismatch, expect: %v, actual: %v", target.DataId, target.Content, got))
	}

	return nil
}

// DeleteConfig deletes target config
func (n *NacosInstallation) DeleteConfig(target *NacosConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	_, err := n.ConfigClient.DeleteConfig(vo.ConfigParam{
		DataId: target.DataId,
		Group:  "org.apache.apisix",
	})

	delete(n.cfgs, target.DataId)
	return err
}

// ClearAllConfig deletes all published config
func (n *NacosInstallation) ClearAllConfig() error {
	for _, v := range n.cfgs {
		err := n.DeleteConfig(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *NacosInstallation) Uninstall() error {
	uninstall := exec.Command(e2econst.Helm, "uninstall", n.ReleaseName, "--namespace", n.Namespace, "--kubeconfig", GetKubeconfig())
	return uninstall.Run()
}

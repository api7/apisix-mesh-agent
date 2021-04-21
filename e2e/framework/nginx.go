package framework

import (
	"net/http"
	"net/url"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	_nginxManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: {{ .NginxReplicas }}
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      name: nginx
      labels:
        app: nginx
    spec:
{{ if .NginxVolumeConfigMap }}
      volumes:
      - name: conf
        configMap:
          name: {{ .NginxVolumeConfigMap }}
{{ end }}
      containers:
      - name: nginx
        image: {{ .LocalRegistry }}/nginx:1.19.3
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          protocol: TCP
          name: http
{{ if .NginxVolumeConfigMap }}
        volumeMounts:
        - name: conf
          mountPath: /etc/nginx/conf.d
{{ end }}
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
`
)

// DeployNginxWithConfigMapVolume deploys Nginx with an extra volume (ConfigMap type).
func (f *Framework) DeployNginxWithConfigMapVolume(cm string) error {
	f.NginxVolumeConfigMap = cm
	defer func() { f.NginxVolumeConfigMap = "" }()
	if err := f.newNginx(); err != nil {
		return err
	}
	if err := f.waitUntilAppNginxPodsReady(); err != nil {
		return err
	}
	return nil
}

// NewHTTPClientToSpringboard creates a http client which sends requests to
// springboard.
func (f *Framework) NewHTTPClientToSpringboard() (*httpexpect.Expect, error) {
	endpoint, err := f.buildTunnelToSpringboardService()
	if err != nil {
		return nil, err
	}
	u := url.URL{
		Scheme: "http",
		Host:   endpoint,
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: http.DefaultTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(httpexpect.NewAssertReporter(ginkgo.GinkgoT())),
	}), nil
}

func (f *Framework) buildTunnelToSpringboardService() (string, error) {
	tunnel := k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, "springboard", 12384, 80)
	if err := tunnel.ForwardPortE(ginkgo.GinkgoT()); err != nil {
		return "", err
	}
	f.tunnels = append(f.tunnels, tunnel)
	return tunnel.Endpoint(), nil
}

func (f *Framework) newNginx() error {
	artifact, err := f.renderManifest(_nginxManifest)
	if err != nil {
		return err
	}
	if err := k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact); err != nil {
		return err
	}

	return nil
}

func (f *Framework) waitUntilAppNginxPodsReady() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=nginx",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(ginkgo.GinkgoT(), f.kubectlOpts, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no nginx pods created")
			return false, nil
		}
		for _, pod := range items {
			found := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}
				found = true
				if cond.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}

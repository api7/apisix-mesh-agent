package framework

import (
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	_springboardManifest = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: springboard
data:
  proxy.conf: |
    server {
    	listen 80 reuseport;
    	location / {
    		proxy_http_version 1.1;
    		proxy_set_header Connection "";
    		proxy_pass http://{{ .SpringboardTarget }};
    		proxy_set_header Host $http_host;
    	}
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: springboard
spec:
  replicas: 1
  selector:
    matchLabels:
      app: springboard
  template:
    metadata:
      name: springboard
      labels:
        app: springboard
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      volumes:
      - name: conf
        configMap:
          name: springboard
      containers:
      - name: springboard
        image: {{ .LocalRegistry }}/nginx:1.19.3
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          protocol: TCP
          name: http
        volumeMounts:
        - name: conf
          mountPath: /etc/nginx/conf.d
---
apiVersion: v1
kind: Service
metadata:
  name: springboard
spec:
  selector:
    app: springboard
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
`
)

// DeploySpringboardWithSpecificProxyTarget deploys
func (f *Framework) DeploySpringboardWithSpecificProxyTarget(target string) error {
	f.SpringboardTarget = target
	defer func() {
		f.SpringboardTarget = ""
	}()
	artifact, err := f.renderManifest(_springboardManifest)
	if err != nil {
		return err
	}
	if err := k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact); err != nil {
		return err
	}
	if err := f.waitUntilAllSpringboardPodsReady(); err != nil {
		return err
	}

	return nil
}

func (f *Framework) waitUntilAllSpringboardPodsReady() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=springboard",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(ginkgo.GinkgoT(), f.kubectlOpts, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no springboard pods created")
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

package framework

import (
	"context"
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	_httpbinManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  labels:
    app: httpbin
spec:
  replicas: {{ .HttpBinReplicas }}
  selector:
    matchLabels:
      app: httpbin
  template:
    metadata:
      labels:
        app: httpbin
    spec:
      containers:
      - name: httpbin
        image: {{ .LocalRegistry }}/kennethreitz/httpbin
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          protocol: TCP
          name: http
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  selector:
    app: httpbin
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
`
)

func (f *Framework) newHttpBin() error {
	artifact, err := f.renderManifest(_httpbinManifest)
	if err != nil {
		return err
	}
	if err := k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact); err != nil {
		return err
	}
	return f.waitUntilAllHttpBinPodsReady()
}

func (f *Framework) waitUntilAllHttpBinPodsReady() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=httpbin",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(ginkgo.GinkgoT(), f.kubectlOpts, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no httpbin pods created")
			clientset, err := k8s.GetKubernetesClientFromOptionsE(ginkgo.GinkgoT(), f.kubectlOpts)
			if err != nil {
				return false, err
			}

			deployments, err := clientset.AppsV1().Deployments(f.kubectlOpts.Namespace).List(context.Background(), opts)
			if err != nil {
				return false, err
			}
			if len(deployments.Items) == 0 {
				ginkgo.GinkgoT().Log("no httpbin deployment created")
				return false, nil
			}
			for _, deployment := range deployments.Items {
				for _, cond := range deployment.Status.Conditions {
					ginkgo.GinkgoT().Logf("deployment %v: %v", deployment.Name, cond.String())
				}
			}
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

// GetHttpBinServiceFQDN returns the FQDN description for HttpBin service.
func (f *Framework) GetHttpBinServiceFQDN() string {
	return fmt.Sprintf("httpbin.%s.svc.cluster.local", f.namespace)
}

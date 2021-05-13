Istio Mesh
==========

This post gives the guide how to integrate apisix-mesh-agent into Istio mesh.

Prerequisites
-------------

### Prepare the Kubernetes cluster

Just use any [Kubernetes](https://kubernetes.io/) cluster that you want, if you don't have an existing one in your hand, we recommend you to use [Kind](https://kind.sigs.k8s.io/) to build a Kubernetes cluster for development quickly, you can run the following commands to set up and clean a Kubernetes cluster with a [Docker Registry](https://docs.docker.com/registry/#:~:text=The%20Registry%20is%20a%20stateless,under%20the%20permissive%20Apache%20license.).

```shell
cd /path/to/apisix-mesh-agent
make kind-up
make kind-reset
```

### Install Helm

In this post, we use [Helm 3](https://helm.io) to install [Istio](https://istio.io). You should download the desired Istio release version to your local environment. In this document, we use [istio/1.9.1](https://github.com/istio/istio/releases/tag/1.9.1).

### Create Istio Root Namespace

In this post, we use the typical `istio-system` as the istio root namespace.

```shell
kubectl create namespace istio-system
```

Build and Push Image
--------------------

```shell
export DOCKER_IMAGE_TAG=dev
export DOCKER_IMAGE_REGISTRY=localhost:5000
cd /path/to/apisix-mesh-agent
make build-image
docker tag api7/apisix-mesh-agent:$DOCKER_IMAGE_TAG $DOCKER_IMAGE_REGISTRY/api7/apisix-mesh-agent:$DOCKER_IMAGE_TAG
docker push $DOCKER_IMAGE_REGISTRY/api7/apisix-mesh-agent:$DOCKER_IMAGE_TAG
```

The above commands build the image firstly and push the image to the target image registry (change the `DOCKER_IMAGE_REGISTRY` to your desired one). You should have [docker](https://www.docker.com/) installed in the running environment.

> Note:
>
> 1. You should change the value of DOCKER_IMAGE_REGISTRY to the actual image registry address that you're using.
>
> 2. Your image registry should be accessible from the Kubernetes cluster.

Install Istio-base Chart
-------------------------

```shell
cd /path/to/istio/manifests
helm install istio-base \
	--namespace istio-system \
	./charts/base
```

istio-base chart contains several resources which are required for running `istiod`.

> Before you execute the above commands, be sure you've cloned [istio](https://istio.io/) to your local.

Install istio-control Chart
----------------------------

```shell
export ISTIO_RELEASE=1.9.1
cd /path/to/istio/manifests
cp /path/to/apisix-mesh-agent/manifests/istio/injection-template.yaml charts/istio-control/istio-discovery/files/
helm install istio-discovery \
	--namespace istio-system \
	--set pilot.image=istio/pilot:$ISTIO_RELEASE \
	--set global.proxy.privileged=true \
	--set global.proxy_init.hub=$DOCKER_IMAGE_REGISTRY \
	--set global.proxy_init.image=api7/apisix-mesh-agent \
	--set global.proxy_init.tag=dev \
	--set global.proxy.hub=$DOCKER_IMAGE_REGISTRY \
	--set global.proxy.image=api7/apisix-mesh-agent \
	--set global.proxy.tag=dev \
	./charts/istio-control/istio-discovery
```

We changed the injection template to [injection-template.yaml](../manifests/istio/injection-template.yaml) as we want to change the sidecar from [Envoy](https://www.envoyproxy.io/) to apisix-mesh-agent.

> Please make sure memory is enough as by default Istios requests `2G` memory, if that's expensive in your Kubernetes cluster, changing the resources configuration by specifying: `--set pilot.resources.requests.memory=<reasonable memory size>`.

Test
----

```shell
kubectl create namespace test
kubectl run nginx --image=nginx -n test --port 80
```

Wait for a while and check out the pod status, the sidecar container should be injected into the nginx pod.

```shell
kubectl get pods -n test
NAME    READY   STATUS    RESTARTS   AGE
nginx   2/2     Running   0          53s
```

For further learning, please read [tiny-service-mesh-scenario](./examples/tiny-service-mesh-scnario.md), so you can know how to verify this mesh by sending requests.

Uninstall
---------

```shell
helm uninstall istio-discovery --namespace istio-system
helm uninstall istio-base --namespace istio-system
kubectl delete namespace istio-system
```

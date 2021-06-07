E2E Test Suites
===============

All E2E test suites are run in your local environment, but all related components are run in a Kuberentes cluster, we recommend you to use [Kind](https://kind.sigs.k8s.io/) and we provide some simple comands
to create the Kubernetes cluster by kind quickly.

Workflow
---------

The e2e framework sets up hooks when running each [ginkgo.Describe](https://pkg.go.dev/github.com/onsi/ginkgo#Describe) block, the `BeforeEach` hook will do the following things before the test case can be run:

1. Create two namespaces, one for service mesh control plane (like [Istio](https://istio.io)), the other for apps.
2. Deploy the control plane, now Istio is in use, it uses [modified charts](./charts) to replace [Envoy](https://www.envoyproxy.io/) by apisix-mesh-agent.
3. Label the app namespace so Pods inside it can be injected by control plane.
4. Deploy the httpbin pod.

Extra components might be deployed inside the test case, such as deploying a Pod as the springboard to send requests.

How to run all the e2e test suites
-----------------------------------

```shell
make e2e-test
```

You can pass the variable `E2E_CONCURRENCY` to control the concurrency.

How can I focus on one test case
---------------------------------

Edit the target test case, changing the `ginkgo.It` to `ginkgo.FIt` or
`ginkgo.Describe` to `ginkgo.FDescribe`, then executing `make e2e-test`.

What if the legacies are remaining due to aborted debugging
------------------------------------------------------------

Just run the following command:

```shell
make cleanup-e2e-legacies
```

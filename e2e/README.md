E2E Test Suites
===============

All E2E test suites are run in your local environment, but all related components are run in a Kuberentes cluster, we recommend you to use [Kind](https://kind.sigs.k8s.io/) and we provide some simple comands
to create the Kubernetes cluster by kind quickly.

How to run all the e2e test suites?
-----------------------------------

```shell
make e2e-test
```

You can pass the variable `E2E_CONCURRENCY` to control the concurrency.

How can I focus on one test case?
---------------------------------

Edit the target test case, changing the `gingko.It` to `ginkgo.FIt` or
`ginkgo.Describe` to `ginkgo.FDescribe`, then executing `make e2e-test`.

What if the legacies are remaining due to aborted debugging?
------------------------------------------------------------

Just run the following command:

```shell
make cleanup-e2e-legacies
```

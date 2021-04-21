default: help
.PHONY: default

### help:                 Show Makefile rules
.PHONY: help
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'

VERSION ?= 0.0.0

GITSHA ?= $(shell git rev-parse --short=7 HEAD)
PWD ?= $(shell pwd)
DATE ?= $(shell date +%s)
DOCKER_IMAGE_TAG ?= dev
ISTIO_IMAGE ?= istio/pilot:1.9.1
NGINX_IMAGE ?= nginx:1.19.3
HTTPBIN_IMAGE ?= kennethreitz/httpbin
E2E_IMAGE_REGISTRY ?= localhost:5000
E2E_CONCURRENCY ?= 1

VERSYM=github.com/api7/apisix-mesh-agent/pkg/version._version
GITSHASYM=github.com/api7/apisix-mesh-agent/pkg/version._gitRevision
TIMESTAMPSYM=github.com/api7/apisix-mesh-agent/pkg/version._timestamp
GO_LDFLAGS ?= "-X=$(VERSYM)=$(VERSION) -X=$(GITSHASYM)=$(GITSHA) -X=$(TIMESTAMPSYM)=$(DATE)"

### build:                Build apisix-mesh-agent
.PHONY: build
build:
	go build \
		-o apisix-mesh-agent \
		-ldflags $(GO_LDFLAGS) \
		main.go

### lint:                 Do static lint check
.PHONY: lint
lint:
	golangci-lint run

### gofmt:                Format all go codes
.PHONY: gofmt
gofmt:
	find . -type f -name "*.go" | xargs gofmt -w -s

### build-image:          Build image
.PHONY: build-image
build-image:
	docker build \
		-t api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG) \
		--build-arg ENABLE_PROXY=true \
		--build-arg LUAROCKS_SERVER=https://luarocks.cn .

### unit-test:            Run unit test cases
.PHONY: unit-test
unit-test:
	go test -cover -coverprofile=coverage.txt ./...

### kind-reset:           Delete the kind k8s cluster
.PHONY: kind-reset
kind-reset:
	kind delete cluster --name apisix

### kind-up:             Create a k8s cluster by kind
.PHONY: kind-up
kind-up:
	./scripts/kind-with-registry.sh

### e2e-test:            Run e2e test suites
.PHONY: e2e-test
e2e-test: kind-up prepare-e2e-env
	APISIX_MESH_AGENT_E2E_HOME=$(shell pwd)/e2e \
		cd e2e && \
		ginkgo -cover -coverprofile=coverage.txt -r --randomizeSuites --randomizeAllSpecs --trace -p --nodes=$(E2E_CONCURRENCY)

### prepare-e2e-env:      Prepare the environment for running e2e test suites
.PHONY: prepare-e2e-env
prepare-e2e-env: cleanup-e2e-legacies build-image
	docker pull $(ISTIO_IMAGE)
	docker tag $(ISTIO_IMAGE) $(E2E_IMAGE_REGISTRY)/$(ISTIO_IMAGE)
	docker push $(E2E_IMAGE_REGISTRY)/$(ISTIO_IMAGE)

	docker pull $(NGINX_IMAGE)
	docker tag $(NGINX_IMAGE) $(E2E_IMAGE_REGISTRY)/$(NGINX_IMAGE)
	docker push $(E2E_IMAGE_REGISTRY)/$(NGINX_IMAGE)

	docker pull $(HTTPBIN_IMAGE)
	docker tag $(HTTPBIN_IMAGE) $(E2E_IMAGE_REGISTRY)/$(HTTPBIN_IMAGE)
	docker push $(E2E_IMAGE_REGISTRY)/$(HTTPBIN_IMAGE)

	docker tag api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG) $(E2E_IMAGE_REGISTRY)/api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG)
	docker push $(E2E_IMAGE_REGISTRY)/api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG)

### cleanup-e2e-legacies: Cleanup the e2e suites running legacies
.PHONY: cleanup-e2e-legacies
cleanup-e2e-legacies:
	kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io | grep istio | awk '{print $$1}' | xargs kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io || true
	kubectl get mutatingwebhookconfigurations.admissionregistration.k8s.io | grep istio | awk '{print $$1}' | xargs kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io || true
	kubectl get ns | grep apisix-mesh-agent | awk '{print $$1}' | xargs  kubectl delete ns || true

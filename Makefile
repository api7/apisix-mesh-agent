default: help

### help:                 Show Makefile rules
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
E2E_IMAGE_REGISTRY ?= localhost:5000

VERSYM=github.com/api7/apisix-mesh-agent/pkg/version._version
GITSHASYM=github.com/api7/apisix-mesh-agent/pkg/version._gitRevision
TIMESTAMPSYM=github.com/api7/apisix-mesh-agent/pkg/version._timestamp
GO_LDFLAGS ?= "-X=$(VERSYM)=$(VERSION) -X=$(GITSHASYM)=$(GITSHA) -X=$(TIMESTAMPSYM)=$(DATE)"

### build:                Build apisix-mesh-agent
build:
	go build \
		-o apisix-mesh-agent \
		-ldflags $(GO_LDFLAGS) \
		main.go

### lint:                 Do static lint check
lint:
	golangci-lint run

### gofmt:                Format all go codes
gofmt:
	find . -type f -name "*.go" | xargs gofmt -w -s

build-image:
	docker build \
		-t api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG) \
		--build-arg ENABLE_PROXY=true \
		--build-arg LUAROCKS_SERVER=https://luarocks.cn .

### unit-test:            Run unit test cases
unit-test:
	go test -cover -coverprofile=coverage.txt ./...

### prepare-e2e-env:      Prepare the environment for running e2e test suites
prepare-e2e-env: cleanup-e2e-legacies build-image
	docker pull $(ISTIO_IMAGE)
	docker tag $(ISTIO_IMAGE) $(E2E_IMAGE_REGISTRY)/$(ISTIO_IMAGE)
	docker push $(E2E_IMAGE_REGISTRY)/$(ISTIO_IMAGE)

	docker pull $(NGINX_IMAGE)
	docker tag $(NGINX_IMAGE) $(E2E_IMAGE_REGISTRY)/$(NGINX_IMAGE)
	docker push $(E2E_IMAGE_REGISTRY)/$(NGINX_IMAGE)

	docker tag api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG) $(E2E_IMAGE_REGISTRY)/api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG)
	docker push $(E2E_IMAGE_REGISTRY)/api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG)

### cleanup-e2e-legacies: Cleanup the e2e suites running legacies
cleanup-e2e-legacies:
	kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io | grep istio | awk '{print $$1}' | xargs kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io
	kubectl get mutatingwebhookconfigurations.admissionregistration.k8s.io | grep istio | awk '{print $$1}' | xargs kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io
	kubectl get ns | grep apisix-mesh-agent | awk '{print $$1}' | xargs  kubectl delete ns

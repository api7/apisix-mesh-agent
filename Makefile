default: help

### help:             Show Makefile rules
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'

VERSION ?= 0.0.0

GITSHA ?= $(shell git rev-parse --short=7 HEAD)
PWD ?= $(shell pwd)
DATE ?= $(shell date +%s)
DOCKER_IMAGE_TAG ?= dev

VERSYM=github.com/api7/apisix-mesh-agent/pkg/version._version
GITSHASYM=github.com/api7/apisix-mesh-agent/pkg/version._gitRevision
TIMESTAMPSYM=github.com/api7/apisix-mesh-agent/pkg/version._timestamp
GO_LDFLAGS ?= "-X=$(VERSYM)=$(VERSION) -X=$(GITSHASYM)=$(GITSHA) -X=$(TIMESTAMPSYM)=$(DATE)"

### build:            Build apisix-mesh-agent
build:
	go build \
		-o apisix-mesh-agent \
		-ldflags $(GO_LDFLAGS) \
		main.go

### lint:             Do static lint check
lint:
	golangci-lint run

### gofmt:            Format all go codes
gofmt:
	find . -type f -name "*.go" | xargs gofmt -w -s

build-image:
	docker build -t api7/apisix-mesh-agent:$(DOCKER_IMAGE_TAG) --build-arg ENABLE_PROXY=true .

### unit-test:        Run unit test cases
unit-test:
	go test -cover -coverprofile=coverage.txt ./...

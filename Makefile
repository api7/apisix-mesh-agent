# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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

### unit-test:        Run unit test cases
unit-test:
	go test -cover -coverprofile=coverage.txt ./...

### license-check:    Do Apache License Header check
license-check:
ifeq ("$(wildcard .actions/openwhisk-utilities/scancode/scanCode.py)", "")
	git clone https://github.com/apache/openwhisk-utilities.git .actions/openwhisk-utilities
	cp .actions/ASF* .actions/openwhisk-utilities/scancode/
endif
	.actions/openwhisk-utilities/scancode/scanCode.py --config .actions/ASF-Release.cfg ./

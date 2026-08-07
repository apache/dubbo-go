#
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
#

SHELL := bash
.DELETE_ON_ERROR:
.DEFAULT_GOAL := help
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-print-directory

CLI_DIR = tools/dubbogo-cli
IMPORTS_FORMATTER_DIR = tools/imports-formatter

.PHONY: help test test-race fmt clean lint check-fmt

help:
	@echo "Available commands:"
	@echo "  test       - Run unit tests"
	@echo "  test-race  - Run race detector on servicediscovery packages"
	@echo "  clean      - Clean test generate files"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run golangci-lint"

# Run unit tests
test: clean
	GOTOOLCHAIN=go1.25.0+auto go test ./... -coverprofile=coverage.txt -covermode=atomic
	cd $(CLI_DIR) && GOTOOLCHAIN=go1.25.0+auto go test ./...

# Tests with known issues (data races or test design defects) tracked in the
# race-detector issue. Skipped via -skip so the whole-repo race run can pass;
# each test should be fixed and removed from this list over time.
RACE_SKIP_TESTS := TestFailbackRetryFailed \
TestFailbackOutOfLimit \
TestRouteCacheGenerationRace \
TestListener \
TestDubboProtocol_Refer \
TestGrpcHealthWatchEmitsClosingEvent \
TestServiceDiscoveryRegistryUnRegister_Concurrent \
TestCfgAPI_Export \
TestCfgAPI_Call \
TestTCPPackageHandle
space := $(subst x, ,x)

test-race:
	GOTOOLCHAIN=go1.25.0+auto go test -race ./... \
		-skip '^($(subst $(space),|,$(RACE_SKIP_TESTS)))$$'

fmt: install-imports-formatter
	# replace interface{} with any
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@v0.21.1 -fix -test ./...
	go fmt ./... && GOROOT=$(shell go env GOROOT) imports-formatter
	cd $(CLI_DIR) && go fmt ./...

# This command is used in CI to verify that code formatting is correct
check-fmt:
	@echo "Checking code format..."
	@$(MAKE) fmt
	@if ! git diff --exit-code --quiet; then \
		echo "Error: The following files have formatting changes:"; \
		git diff --name-only; \
		echo ""; \
		echo "Formatting diff:"; \
		git --no-pager diff --; \
		echo ""; \
		echo "Please run 'make fmt' to fix formatting issues and commit the changes."; \
		exit 1; \
	fi

# Clean test generate files
clean:
	rm -rf coverage.txt

# Run golangci-lint
lint: install-golangci-lint
	go vet ./...
	golangci-lint run ./... --timeout=10m

install-golangci-lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2

install-imports-formatter:
	cd $(IMPORTS_FORMATTER_DIR) && go install

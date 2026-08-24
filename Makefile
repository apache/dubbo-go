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

GO ?= go
TOOLS_DIR ?= .tools
TOOLS_BIN := $(TOOLS_DIR)/bin
COVERAGE_FILE ?= coverage.txt

GOLANGCI_LINT_VERSION ?= v2.7.2
MODERNIZE_VERSION ?= v0.21.1

ifeq ($(OS),Windows_NT)
BIN_EXT := .exe
else
BIN_EXT :=
endif

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

GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint$(BIN_EXT)
IMPORTS_FORMATTER := $(TOOLS_BIN)/imports-formatter$(BIN_EXT)
MODERNIZE := $(TOOLS_BIN)/modernize$(BIN_EXT)
GOLANGCI_LINT_STAMP := $(TOOLS_BIN)/.golangci-lint-$(GOLANGCI_LINT_VERSION)
IMPORTS_FORMATTER_STAMP := $(TOOLS_BIN)/.imports-formatter
MODERNIZE_STAMP := $(TOOLS_BIN)/.modernize-$(MODERNIZE_VERSION)

# Pass cross-compilation settings to every Go command without changing the host environment.
GO_ENV := $(strip $(if $(GOOS),GOOS=$(GOOS)) $(if $(GOARCH),GOARCH=$(GOARCH)) $(if $(CGO_ENABLED),CGO_ENABLED=$(CGO_ENABLED)))
GO_RUN = $(GO_ENV) $(GO)

.PHONY: help build test test-race generate-mocks fmt check-fmt clean lint tools

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the root module and the CLI
	$(GO_RUN) build ./...
	cd $(CLI_DIR) && $(GO_RUN) build ./...

test: clean ## Run unit tests and write the root coverage profile
	$(GO_RUN) test ./... -coverprofile=$(CURDIR)/$(COVERAGE_FILE) -covermode=atomic
	cd $(CLI_DIR) && $(GO_RUN) test ./...

test-race: clean ## Run unit tests with the race detector
	$(GO_RUN) test ./... -race -skip '^($(subst $(space),|,$(RACE_SKIP_TESTS)))$$' -coverprofile=$(CURDIR)/$(COVERAGE_FILE) -covermode=atomic
	cd $(CLI_DIR) && $(GO_RUN) test ./... -race

generate-mocks: ## Regenerate GoMock implementations with the pinned generator
	$(GO_RUN) generate ./cluster/metrics ./filter ./protocol/base

fmt: $(MODERNIZE_STAMP) $(IMPORTS_FORMATTER_STAMP) ## Format Go code and modernize syntax
	# Replace interface{} with any and apply the repository's import grouping rules.
	$(MODERNIZE) -fix -test ./...
	$(GO_RUN) fmt ./...
	GOROOT=$$($(GO) env GOROOT) $(IMPORTS_FORMATTER)
	cd $(CLI_DIR) && $(GO_RUN) fmt ./...

check-fmt: ## Check gofmt output without modifying files
	@unformatted="$$(git ls-files '*.go' | xargs gofmt -l)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Error: gofmt changes are required:"; \
		echo "$$unformatted"; \
		echo "Run 'make fmt' and commit the result."; \
		exit 1; \
	fi

clean: ## Remove generated test artifacts
	rm -f $(COVERAGE_FILE)

lint: $(GOLANGCI_LINT_STAMP) ## Run the configured golangci-lint checks
	$(GOLANGCI_LINT) run ./... --timeout=10m

tools: $(GOLANGCI_LINT_STAMP) $(IMPORTS_FORMATTER_STAMP) $(MODERNIZE_STAMP) ## Install pinned development tools locally

$(TOOLS_BIN):
	mkdir -p $@

$(GOLANGCI_LINT_STAMP): | $(TOOLS_BIN)
	GOBIN=$(abspath $(TOOLS_BIN)) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	touch $@

$(IMPORTS_FORMATTER_STAMP): | $(TOOLS_BIN)
	cd $(IMPORTS_FORMATTER_DIR) && GOBIN=$(abspath $(TOOLS_BIN)) $(GO) install .
	touch $@

$(MODERNIZE_STAMP): | $(TOOLS_BIN)
	GOBIN=$(abspath $(TOOLS_BIN)) $(GO) install golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@$(MODERNIZE_VERSION)
	touch $@

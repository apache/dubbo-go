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

.PHONY: help test fmt clean lint

help:
	@echo "Available commands:"
	@echo "  test       - Run unit tests"
	@echo "  clean      - Clean test generate files"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run golangci-lint"

# Run unit tests
test: clean
	go test ./... -coverprofile=coverage.txt -covermode=atomic
	cd $(CLI_DIR) && go test ./...

fmt: install-imports-formatter
	# replace interface{} with any
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -category=efaceany -fix -test ./...
	go fmt ./... && GOROOT=$(shell go env GOROOT) imports-formatter
	cd $(CLI_DIR) && go fmt ./...

# Clean test generate files
clean:
	rm -rf coverage.txt

# Run golangci-lint
lint: install-golangci-lint
	go vet ./...
	golangci-lint run ./... --timeout=10m

install-golangci-lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0

install-imports-formatter:
	cd $(IMPORTS_FORMATTER_DIR) && go install

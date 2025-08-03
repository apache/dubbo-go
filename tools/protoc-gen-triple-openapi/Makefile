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
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# Variables
BINARY_NAME=protoc-gen-triple-openapi
PROTO_FILE=./example/greet.proto
OUTPUT_DIR=./example

.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build     Build the binary"
	@echo "  generate  Generate OpenAPI spec from the example proto file"
	@echo "  clean     Remove generated files"
	@echo "  help      Show this help message"

.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

.PHONY: generate
generate: build
	@echo "Generating OpenAPI spec from $(PROTO_FILE)..."
	protoc --plugin=$(BINARY_NAME)=./$(BINARY_NAME) --triple-openapi_out=. $(PROTO_FILE)
	protoc --plugin=$(BINARY_NAME)=./$(BINARY_NAME) --triple-openapi_out=. --triple-openapi_opt=format=json $(PROTO_FILE)

.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME) $(OUTPUT_DIR)/greet.triple.openapi.yaml $(OUTPUT_DIR)/greet.triple.openapi.json

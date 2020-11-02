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

VERSION ?= latest

GO = go
GO_PATH = $$($(GO) env GOPATH)
GO_OS = $$($(GO) env GOOS)
GO_BUILD = $(GO) build
GO_GET = $(GO) get
GO_TEST = $(GO) test
GO_BUILD_FLAGS = -v
GO_BUILD_LDFLAGS = -X main.version=$(VERSION)

GO_LICENSE_CHECKER = $(GO_PATH)/bin/license-header-checker
GO_LICENSE_CHECKER_DIR = license-header-checker-$(GO_OS)
LICENSE_DIR = /tmp/tools/license/license.txt

PLATFORMS := windows linux darwin
os = $(word 1, $@)
ARCH = amd64

SHELL = /bin/bash

prepare:
	$(GO_LICENSE_CHECKER) -version || (wget https://github.com/lsm-dev/license-header-checker/releases/download/v1.1.0/$(GO_LICENSE_CHECKER_DIR).zip -O $(GO_LICENSE_CHECKER_DIR).zip && unzip -o $(GO_LICENSE_CHECKER_DIR).zip && cp $(GO_LICENSE_CHECKER_DIR)/64bits/license-header-checker $(GO_PATH)/bin/)
	ls $(LICENSE_DIR) || wget -O $(LICENSE_DIR) https://github.com/dubbogo/resources/raw/master/tools/license/license.txt
	#./before_ut.sh

.PHONE: test
test: clean lint
	$(GO_TEST) ./... -coverprofile=coverage.txt -covermode=atomic

deps: prepare
	$(GO_GET) -v -t -d ./...

.PHONY: license
license: clean prepare
	$(GO_LICENSE_CHECKER) -v -a -r -i vendor $(LICENSE_DIR) . go  && [[ -z `git status -s` ]]

.PHONY: verify
verify: clean license lint test

.PHONY: clean
clean: prepare
	-rm -rf coverage.txt

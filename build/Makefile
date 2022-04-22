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

# DO NOT RUN
# The makefile is for ci test and has dependencies. Do not run it locally.
# If you want to run the unit tests, run command `go test ./...` directly.

VERSION ?= latest

GO = go
GO_PATH = $(shell $(GO) env GOPATH)
GO_OS = $(shell $(GO) env GOOS)
ifeq ($(GO_OS), darwin)
    GO_OS = mac
endif
GO_BUILD = $(GO) build
GO_GET = $(GO) get
GO_TEST = $(GO) test
GO_BUILD_FLAGS = -v
GO_BUILD_LDFLAGS = -X main.version=$(VERSION)

GO_LICENSE_CHECKER_DIR = license-header-checker-$(GO_OS)
GO_LICENSE_CHECKER = $(GO_PATH)/bin/license-header-checker
LICENSE_DIR = /tmp/tools/license

ARCH = amd64
# for add zookeeper fatjar
ZK_TEST_LIST=config_center/zookeeper registry/zookeeper cluster/router/chain cluster/router/condition cluster/router/tag  metadata/report/zookeeper
ZK_JAR_NAME=zookeeper-3.4.9-fatjar.jar
ZK_FATJAR_BASE=/zookeeper-4unittest/contrib/fatjar
ZK_JAR_PATH=remoting/zookeeper$(ZK_FATJAR_BASE)
ZK_JAR=$(ZK_JAR_PATH)/$(ZK_JAR_NAME)

SHELL = /bin/bash

prepareLic:
	echo 'The makefile is for ci test and has dependencies. Do not run it locally. If you want to run the unit tests, run command `go test ./...` directly.'
	$(GO_LICENSE_CHECKER) -version || (wget https://github.com/lsm-dev/license-header-checker/releases/download/v1.2.0/$(GO_LICENSE_CHECKER_DIR).zip -O $(GO_LICENSE_CHECKER_DIR).zip && unzip -o $(GO_LICENSE_CHECKER_DIR).zip && mkdir -p $(GO_PATH)/bin/ && cp $(GO_LICENSE_CHECKER_DIR)/64bit/license-header-checker $(GO_PATH)/bin/)
	ls /tmp/tools/license/license.txt || wget -P $(LICENSE_DIR) https://github.com/dubbogo/resources/raw/master/tools/license/license.txt

prepare: prepareLic

.PHONE: test
test: clean
	$(GO_TEST) ./... -coverprofile=coverage.txt -covermode=atomic

deps: prepare
	$(GO_GET) -v -t -d ./...

.PHONY: license
license: clean prepareLic
	$(GO_LICENSE_CHECKER) -v -a -r -i vendor $(LICENSE_DIR)/license.txt . go && [[ -z `git status -s` ]]
install: install
	$(GO_BUILD)

.PHONY: verify
verify: clean license test

.PHONY: clean
clean: prepare
	rm -rf coverage.txt
	rm -rf license-header-checker*

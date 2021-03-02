#
#Licensed to the Apache Software Foundation (ASF) under one or more
#contributor license agreements.  See the NOTICE file distributed with
#this work for additional information regarding copyright ownership.
#The ASF licenses this file to You under the Apache License, Version 2.0
#(the "License"); you may not use this file except in compliance with
#the License.  You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
#

FROM golang

WORKDIR /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-server

ENV CONF_PROVIDER_FILE_PATH "server.yml"
ENV APP_LOG_CONF_FILE "log.yml"

ARG PR_ORIGIN_REPO
ARG PR_ORIGIN_COMMITID

ADD . /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-server
# update dubbo-go to current commit id
RUN test ${PR_ORIGIN_REPO} && echo "github.com/apache/dubbo-go will be replace to github.com/${PR_ORIGIN_REPO}@${PR_ORIGIN_COMMITID}" || echo 'go get github.com/apache/dubbo-go@develop'
RUN test ${PR_ORIGIN_REPO} && go mod edit  -replace=github.com/apache/dubbo-go=github.com/${PR_ORIGIN_REPO}@${PR_ORIGIN_COMMITID} || go get -u github.com/apache/dubbo-go@develop

ENV GO111MODULE on

RUN go mod tidy && go install github.com/apache/dubbo-go/test/integrate/dubbo/go-server

CMD go-server

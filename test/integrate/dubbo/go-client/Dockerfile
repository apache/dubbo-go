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

WORKDIR /go/src/dubbo.apache.org/dubbo-go/v3/test/integrate/dubbo/go-client

ENV CONF_CONSUMER_FILE_PATH "client.yml"
ENV APP_LOG_CONF_FILE "log.yml"

# for pull_request, it is pr origin repository
# for merge int develop/master, it is github.com/apache/dubbo-go
ARG REPO
# for pull_request, it is pr origin commit id
# for merge int develop/master, it is the commit id that trigger the ci workflow
ARG COMMITID

ADD . /go/src/dubbo.apache.org/dubbo-go/v3/test/integrate/dubbo/go-client

# update dubbo-go to current commit id
RUN test ${REPO} && echo "dubbo.apache.org/dubbo-go/v3 will be replace to github.com/${REPO}/v3@${COMMITID}"
RUN test ${REPO} && go mod edit -replace=dubbo.apache.org/dubbo-go/v3=github.com/${REPO}/v3@${COMMITID}

ENV GO111MODULE on

RUN go mod tidy && go install dubbo.apache.org/dubbo-go/v3/test/integrate/dubbo/go-client

CMD go-client
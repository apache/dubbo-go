#
#  Licensed to the Apache Software Foundation (ASF) under one or more
#  contributor license agreements.  See the NOTICE file distributed with
#  this work for additional information regarding copyright ownership.
#  The ASF licenses this file to You under the Apache License, Version 2.0
#  (the "License"); you may not use this file except in compliance with
#  the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

#!/bin/bash

set -e
set -x

echo 'start integrate-test'

# set root workspace
ROOT_DIR=$(pwd)
echo "integrate-test root work-space -> ${ROOT_DIR}"

# show all github-env
echo "github current commit id  -> $2"
echo "github pull request branch -> ${GITHUB_REF}"
echo "github pull request slug -> ${GITHUB_REPOSITORY}"
echo "github pull request repo slug -> ${GITHUB_REPOSITORY}"
echo "github pull request actor -> ${GITHUB_ACTOR}"
echo "github pull request repo param -> $1"
echo "github pull request base branch -> $3"
echo "github pull request head branch -> ${GITHUB_HEAD_REF}"

samples_testing() {
    echo "use dubbo-go-samples $3 branch for integration testing"
    git clone -b "$3" https://github.com/apache/dubbo-go-samples.git samples && cd samples

    # update dubbo-go to current commit id
    go mod edit -replace=dubbo.apache.org/dubbo-go/v3=github.com/"$1"/v3@"$2"

    # start integrate test
    ./start_integrate_test.sh
}

local_testing() {
    echo "use test/integrate/dubbo for integration testing"
    # default use zk as registry
    #start zookeeper registry insecure listen in [:]:2181
    docker run -d --network host zookeeper
    echo "zookeeper listen in [:]2181"

    # build go-server image
    cd ./test/integrate/dubbo/go-server
    docker build . -t ci-provider --build-arg REPO="$1" --build-arg COMMITID="$2"
    cd "${ROOT_DIR}"
    docker run -d --network host ci-provider
}

# check dubbo-go-samples corresponding branch
res=$(git ls-remote --heads https://github.com/apache/dubbo-go-samples.git "$3" | wc -l)
if [ "$res" -eq "1" ]; then
    samples_testing "$@"
else
    local_testing "$@"
fi

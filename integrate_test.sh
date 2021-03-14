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

# show all travis-env
echo "travis current commit id  -> $2"
echo "travis pull request branch -> ${GITHUB_REF}"
echo "travis pull request slug -> ${GITHUB_REPOSITORY}"
echo "travis pull request repo slug -> ${GITHUB_REPOSITORY}"
echo "travis pull request actor -> ${GITHUB_ACTOR}"
echo "travis pull request repo param -> $1"


# #start etcd registry  insecure listen in [:]:2379
# docker run -d --network host k8s.gcr.io/etcd:3.3.10 etcd
# echo "etcdv3 listen in [:]2379"

# #start consul registry insecure listen in [:]:8500
# docker run -d --network host consul
# echo "consul listen in [:]8500"

# #start nacos registry insecure listen in [:]:8848
# docker run -d --network host nacos/nacos-server:latest
# echo "ncacos listen in [:]8848"

# default use zk as registry
#start zookeeper registry insecure listen in [:]:2181
docker run -d --network host zookeeper
echo "zookeeper listen in [:]2181"

# build go-server image
cd ./test/integrate/dubbo/go-server
docker build . -t  ci-provider --build-arg PR_ORIGIN_REPO=$1 --build-arg PR_ORIGIN_COMMITID=$2
cd ${ROOT_DIR}
docker run -d --network host ci-provider

# build go-client image
cd ./test/integrate/dubbo/go-client
docker build . -t  ci-consumer --build-arg PR_ORIGIN_REPO=$1 --build-arg PR_ORIGIN_COMMITID=$2
cd ${ROOT_DIR}
# run provider
# check consumer status
docker run -i --network host ci-consumer

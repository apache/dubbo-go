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

echo "start integrate-test: repo = $1, SHA = $2, branch = $3"

# set root workspace
ROOT_DIR=$(pwd)
echo "integrate-test root work-space -> ${ROOT_DIR}"

echo "use dubbo-go-samples $3 branch for integration testing"
git clone -b $3 https://github.com/apache/dubbo-go-samples.git samples && cd samples

# update dubbo-go to current commit id
if [ "$1" == "apache/dubbo-go" ]; then
    go mod edit -replace=dubbo.apache.org/dubbo-go/v3=dubbo.apache.org/dubbo-go/v3@"$2"
else
    go mod edit -replace=dubbo.apache.org/dubbo-go/v3=github.com/"$1"/v3@"$2"
fi

go mod tidy

# start integrate test
./start_integrate_test.sh

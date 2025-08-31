#!/bin/bash
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

go build

for dir in ./test/correctly/*/; do
    cd "$dir" || exit 1

    dir_name=$(basename "$dir")

    protoc --go_out=. --go_opt=paths=source_relative --plugin=protoc-gen-go-triple=../../../protoc-gen-go-triple --go-triple_out=. ./proto/greet.proto
    go mod tidy

    go vet ./proto/*.go
    result=$?

    if [ $result -ne 0 ]; then
        echo "go vet found issues in $dir_name."
        exit $result
    fi

    echo "No issues found in $dir_name."
    cd - || exit 1
done
#!/bin/bash
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

set -e

BASE_DIR=$(cd "$(dirname "$0")/.." && pwd)
PROTO_DIR="$BASE_DIR/proto"
OUT_DIR="$PROTO_DIR/benchmark_gen"

mkdir -p "$OUT_DIR"

echo "[INFO] 生成protobuf代码..."
protoc --go_out="$OUT_DIR" --go_opt=paths=source_relative "$PROTO_DIR/benchmark.proto"

echo "[INFO] 生成triple代码..."
protoc --go-triple_out="$OUT_DIR" "$PROTO_DIR/benchmark.proto"

echo "[INFO] 代码生成完成"
ls -la "$OUT_DIR"

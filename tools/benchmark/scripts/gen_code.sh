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
PROJECT_ROOT=$(cd "$BASE_DIR/../.." && pwd)
PROTO_DIR="$BASE_DIR/proto"
OUT_DIR="$PROTO_DIR"
PLUGIN_DIR="$PROJECT_ROOT/tools/protoc-gen-go-triple"

mkdir -p "$OUT_DIR"

echo "[INFO] Building protoc-gen-go-triple plugin..."
cd "$PLUGIN_DIR"
go build -o protoc-gen-go-triple .
cd -

echo "[INFO] Generating protobuf code..."
protoc --proto_path="$PROTO_DIR" --go_out="$OUT_DIR" --go_opt=paths=source_relative "benchmark.proto"

echo "[INFO] Generating triple code using protoc-gen-go-triple..."
protoc --proto_path="$PROTO_DIR" \
  --plugin=protoc-gen-go-triple="$PLUGIN_DIR/protoc-gen-go-triple" \
  --go-triple_out="$OUT_DIR" \
  --go-triple_opt=paths=source_relative \
  "benchmark.proto"

echo "[INFO] Cleaning up plugin..."
rm -f "$PLUGIN_DIR/protoc-gen-go-triple"

echo "[INFO] Code generation completed"
ls -la "$OUT_DIR"

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
LOG_DIR="$BASE_DIR/logs"

mkdir -p "$LOG_DIR"

FRAMEWORK="${1:-dubbo-go}"
PAYLOAD="${2:-1024}"
SERIALIZATION="${3:-protobuf}"
COMPRESSION="${4:-none}"
CONCURRENCY="${5:-100}"
CALL_MODE="${6:-unary}"

echo "========================================"
echo "   Dubbo-Go Benchmark - Single Test"
echo "========================================"
echo "框架:         $FRAMEWORK"
echo "报文大小:     $PAYLOAD bytes"
echo "序列化:       $SERIALIZATION"
echo "压缩:         $COMPRESSION"
echo "并发数:       $CONCURRENCY"
echo "调用模式:     $CALL_MODE"
echo "========================================"

echo "[INFO] 编译服务端..."
case "$FRAMEWORK" in
    dubbo-go)
        cd "$BASE_DIR/server/dubbo-go"
        go build -o benchmark-dubbo-go main.go
        SERVER_BIN="$BASE_DIR/server/dubbo-go/benchmark-dubbo-go"
        SERVER_PORT=20000
        ;;
    grpc)
        cd "$BASE_DIR/server/grpc"
        go build -o benchmark-grpc main.go
        SERVER_BIN="$BASE_DIR/server/grpc/benchmark-grpc"
        SERVER_PORT=50051
        ;;
    *)
        echo "[ERROR] 不支持的框架: $FRAMEWORK"
        exit 1
        ;;
esac

echo "[INFO] 编译客户端..."
cd "$BASE_DIR/client"
go build -o benchmark-client main.go

LOG_FILE="$LOG_DIR/${FRAMEWORK}_${PAYLOAD}_${SERIALIZATION}_${COMPRESSION}_${CONCURRENCY}_${CALL_MODE}.log"

echo ""
echo "[INFO] 启动服务端..."
case "$FRAMEWORK" in
    dubbo-go)
        "$SERVER_BIN" --serialization "$SERIALIZATION" --compression "$COMPRESSION" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
        ;;
    grpc)
        "$SERVER_BIN" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
        ;;
esac
pid=$!
echo "[INFO] 服务端PID: $pid"

sleep 3

echo ""
echo "[INFO] 开始压测..."
"$BASE_DIR/client/benchmark-client" \
    --framework "$FRAMEWORK" \
    --payload "$PAYLOAD" \
    --serialization "$SERIALIZATION" \
    --compression "$COMPRESSION" \
    --concurrency "$CONCURRENCY" \
    --mode "$CALL_MODE" \
    --pid "$pid"

echo ""
echo "[INFO] 停止服务端..."
kill "$pid" 2>/dev/null || true
sleep 2
if kill -0 "$pid" 2>/dev/null; then
    kill -9 "$pid" 2>/dev/null || true
fi

echo ""
echo "========================================"
echo "   测试完成!"
echo "========================================"
echo "日志位置: $LOG_FILE"

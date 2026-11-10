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
SEPARATOR="========================================"

mkdir -p "$LOG_DIR"

cleanup() {
    echo "[INFO] Cleaning up resources..."
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        kill "$SERVER_PID" 2>/dev/null || true
        sleep 2
        if kill -0 "$SERVER_PID" 2>/dev/null; then
            kill -9 "$SERVER_PID" 2>/dev/null || true
        fi
    fi
}

trap cleanup EXIT INT TERM

FRAMEWORK="${1:-dubbo-go}"
PAYLOAD="${2:-1024}"
SERIALIZATION="${3:-protobuf}"
COMPRESSION="${4:-none}"
CONCURRENCY="${5:-100}"
CALL_MODE="${6:-unary}"

wait_for_port() {
    local port=$1
    local timeout=${2:-30}
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if nc -z localhost "$port" 2>/dev/null; then
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    return 1
}

echo "$SEPARATOR"
echo "   Dubbo-Go Benchmark - Single Test"
echo "$SEPARATOR"
echo "Framework:         $FRAMEWORK"
echo "Payload Size:      $PAYLOAD bytes"
echo "Serialization:     $SERIALIZATION"
echo "Compression:       $COMPRESSION"
echo "Concurrency:       $CONCURRENCY"
echo "Call Mode:         $CALL_MODE"
echo "$SEPARATOR"

echo "[INFO] Compiling server..."
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
    dubbo-java)
        cd "$BASE_DIR/server/dubbo-java"
        mvn clean package -DskipTests -q
        SERVER_BIN="$BASE_DIR/server/dubbo-java/target/benchmark-dubbo-java.jar"
        SERVER_PORT=20001
        ;;
    *)
        echo "[ERROR] Unsupported framework: $FRAMEWORK"
        exit 1
        ;;
esac

echo "[INFO] Compiling client..."
cd "$BASE_DIR/client"
go build -o benchmark-client main.go

LOG_FILE="$LOG_DIR/${FRAMEWORK}_${PAYLOAD}_${SERIALIZATION}_${COMPRESSION}_${CONCURRENCY}_${CALL_MODE}.log"

echo ""
echo "[INFO] Starting server..."
case "$FRAMEWORK" in
    dubbo-go)
        "$SERVER_BIN" --serialization "$SERIALIZATION" --compression "$COMPRESSION" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
        ;;
    grpc)
        "$SERVER_BIN" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
        ;;
    dubbo-java)
        java -jar "$SERVER_BIN" > "$LOG_FILE.server.log" 2>&1 &
        ;;
    *)
        echo "[ERROR] Unsupported framework: $FRAMEWORK"
        exit 1
        ;;
esac
SERVER_PID=$!
echo "[INFO] Server PID: $SERVER_PID"

echo "[INFO] Waiting for server to be ready on port $SERVER_PORT..."
if ! wait_for_port "$SERVER_PORT" 30; then
    echo "[ERROR] Server failed to start within 30 seconds"
    kill "$SERVER_PID" 2>/dev/null || true
    exit 1
fi
echo "[INFO] Server is ready"

echo ""
echo "[INFO] Starting benchmark..."
"$BASE_DIR/client/benchmark-client" \
    --framework "$FRAMEWORK" \
    --payload "$PAYLOAD" \
    --serialization "$SERIALIZATION" \
    --compression "$COMPRESSION" \
    --concurrency "$CONCURRENCY" \
    --mode "$CALL_MODE" \
    --pid "$SERVER_PID" \
    --output "$BASE_DIR/data"

echo ""
echo "$SEPARATOR"
echo "   Test completed!"
echo "$SEPARATOR"
echo "Log location: $LOG_FILE"

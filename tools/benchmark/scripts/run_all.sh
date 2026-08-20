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
REPORT_DIR="$BASE_DIR/report"
DATA_DIR="$BASE_DIR/data"
SEPARATOR="========================================"

mkdir -p "$LOG_DIR"
mkdir -p "$REPORT_DIR"
mkdir -p "$DATA_DIR"

cleanup_server() {
    local pid=$1
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        sleep 2
        if kill -0 "$pid" 2>/dev/null; then
            kill -9 "$pid" 2>/dev/null || true
        fi
    fi
}

echo "$SEPARATOR"
echo "   Dubbo-Go Benchmark - Full Test Suite"
echo "$SEPARATOR"

echo "[INFO] Checking environment dependencies..."

if ! command -v go > /dev/null 2>&1; then
    echo "[ERROR] Go not installed, please install Go 1.25+"
    exit 1
fi

echo "[INFO] Environment check passed"

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

echo ""
echo "[INFO] Compiling Dubbo-Go server..."
cd "$BASE_DIR/server/dubbo-go"
go build -o benchmark-dubbo-go main.go

echo "[INFO] Compiling gRPC server..."
cd "$BASE_DIR/server/grpc"
go build -o benchmark-grpc main.go

echo "[INFO] Compiling Dubbo-Java server..."
cd "$BASE_DIR/server/dubbo-java"
mvn clean package -DskipTests -q

echo ""
echo "[INFO] Compiling benchmark client..."
cd "$BASE_DIR/client"
go build -o benchmark-client main.go

echo ""
echo "[INFO] Starting full benchmark suite..."

FRAMEWORKS=("dubbo-go" "grpc" "dubbo-java")
PAYLOADS=("128" "1024" "16384" "1048576")
SERIALIZATIONS=("protobuf")
COMPRESSIONS=("none")
CONCURRENCY=("50" "100")
CALL_MODES=("unary")

for framework in "${FRAMEWORKS[@]}"; do
    echo ""
    echo "[INFO] ==== Testing framework: $framework ===="
    
    case "$framework" in
        dubbo-go)
            SERVER_BIN="$BASE_DIR/server/dubbo-go/benchmark-dubbo-go"
            SERVER_PORT=20000
            ;;
        grpc)
            SERVER_BIN="$BASE_DIR/server/grpc/benchmark-grpc"
            SERVER_PORT=50051
            ;;
        dubbo-java)
            SERVER_BIN="$BASE_DIR/server/dubbo-java/target/benchmark-dubbo-java.jar"
            SERVER_PORT=20001
            ;;
        *)
            echo "[WARNING] Skipping unknown framework: $framework"
            continue
            ;;
    esac

    for payload in "${PAYLOADS[@]}"; do
        for serialization in "${SERIALIZATIONS[@]}"; do
            for compression in "${COMPRESSIONS[@]}"; do
                for concurrency in "${CONCURRENCY[@]}"; do
                    for mode in "${CALL_MODES[@]}"; do
                        echo ""
                        echo "--------------------------------------------------------"
                        echo "Test case: $framework | $payload bytes | $serialization | $compression | $concurrency concurrency | $mode"
                        echo "--------------------------------------------------------"

                        LOG_FILE="$LOG_DIR/${framework}_${payload}_${serialization}_${compression}_${concurrency}_${mode}.log"

                        echo "[INFO] Starting server..."
                        case "$framework" in
                            dubbo-go)
                                "$SERVER_BIN" --serialization "$serialization" --compression "$compression" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
                                ;;
                            grpc)
                                "$SERVER_BIN" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
                                ;;
                            dubbo-java)
                                java -jar "$SERVER_BIN" > "$LOG_FILE.server.log" 2>&1 &
                                ;;
                            *)
                                echo "[ERROR] Unsupported framework: $framework"
                                exit 1
                                ;;
                        esac
                        SERVER_PID=$!

                        trap "cleanup_server $SERVER_PID" EXIT INT TERM

                        echo "[INFO] Waiting for server to be ready on port $SERVER_PORT..."
                        if ! wait_for_port "$SERVER_PORT" 30; then
                            echo "[ERROR] Server failed to start within 30 seconds"
                            cleanup_server "$SERVER_PID"
                            trap - EXIT INT TERM
                            continue
                        fi
                        echo "[INFO] Server is ready"

                        echo "[INFO] Starting benchmark..."
                        "$BASE_DIR/client/benchmark-client" \
                            --framework "$framework" \
                            --payload "$payload" \
                            --serialization "$serialization" \
                            --compression "$compression" \
                            --concurrency "$concurrency" \
                            --mode "$mode" \
                            --pid "$SERVER_PID" \
                            --output "$DATA_DIR" \
                            > "$LOG_FILE" 2>&1

                        echo "[INFO] Test case completed, log saved to $LOG_FILE"

                        cleanup_server "$SERVER_PID"
                        trap - EXIT INT TERM
                    done
                done
            done
        done
    done
done

echo ""
echo "$SEPARATOR"
echo "   Benchmark completed!"
echo "$SEPARATOR"
echo "Log location: $LOG_DIR/"
echo "Data location: $DATA_DIR/"

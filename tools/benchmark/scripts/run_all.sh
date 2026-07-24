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
PID_FILE="/tmp/benchmark_server.pid"
SEPARATOR="========================================"

mkdir -p "$LOG_DIR"
mkdir -p "$REPORT_DIR"
mkdir -p "$DATA_DIR"

echo "$SEPARATOR"
echo "   Dubbo-Go Benchmark - Full Test Suite"
echo "$SEPARATOR"

echo "[INFO] checking environment dependencies..."

if ! command -v go > /dev/null 2>&1; then
    echo "[ERROR] Go is not installed, please install Go 1.23+"
    exit 1
fi

echo "[INFO] environment check passed"

cleanup() {
    echo "[INFO] cleaning up resources..."
    if [ -f "$PID_FILE" ]; then
        pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            sleep 2
            if kill -0 "$pid" 2>/dev/null; then
                kill -9 "$pid" 2>/dev/null || true
            fi
        fi
        rm -f "$PID_FILE"
    fi
}

trap cleanup EXIT

echo ""
echo "[INFO] building Dubbo-Go server..."
cd "$BASE_DIR/server/dubbo-go"
go build -o benchmark-dubbo-go main.go

echo "[INFO] building gRPC server..."
cd "$BASE_DIR/server/grpc"
go build -o benchmark-grpc main.go

echo ""
echo "[INFO] building benchmark client..."
cd "$BASE_DIR/client"
go build -o benchmark-client main.go

echo ""
echo "[INFO] starting full benchmark suite..."

FRAMEWORKS=("dubbo-go" "grpc")
PAYLOADS=("128" "1024" "16384" "1048576")
SERIALIZATIONS=("protobuf")
COMPRESSIONS=("none")
CONCURRENCY=("50" "100")
CALL_MODES=("unary")

for framework in "${FRAMEWORKS[@]}"; do
    echo ""
    echo "[INFO] ==== testing framework: $framework ===="
    
    case "$framework" in
        dubbo-go)
            SERVER_BIN="$BASE_DIR/server/dubbo-go/benchmark-dubbo-go"
            SERVER_PORT=20000
            ;;
        grpc)
            SERVER_BIN="$BASE_DIR/server/grpc/benchmark-grpc"
            SERVER_PORT=50051
            ;;
        *)
            echo "[WARNING] skipping unknown framework: $framework"
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
                        echo "Test scenario: $framework | $payload bytes | $serialization | $compression | $concurrency concurrency | $mode"
                        echo "--------------------------------------------------------"

                        LOG_FILE="$LOG_DIR/${framework}_${payload}_${serialization}_${compression}_${concurrency}_${mode}.log"
                        DATA_FILE="$DATA_DIR/${framework}_${payload}_${serialization}_${compression}_${concurrency}_${mode}.json"

                        echo "[INFO] starting server..."
                        case "$framework" in
                            dubbo-go)
                                "$SERVER_BIN" --serialization "$serialization" --compression "$compression" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
                                ;;
                            grpc)
                                "$SERVER_BIN" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
                                ;;
                            *)
                                echo "[ERROR] unsupported framework: $framework"
                                exit 1
                                ;;
                        esac
                        pid=$!
                        echo "$pid" > "$PID_FILE"

                        sleep 3

                        echo "[INFO] starting benchmark..."
                        "$BASE_DIR/client/benchmark-client" \
                            --framework "$framework" \
                            --payload "$payload" \
                            --serialization "$serialization" \
                            --compression "$compression" \
                            --concurrency "$concurrency" \
                            --mode "$mode" \
                            --pid "$pid" \
                            > "$LOG_FILE" 2>&1

                        echo "[INFO] scenario completed, logs saved to $LOG_FILE"

                        echo "[INFO] stopping server..."
                        kill "$pid" 2>/dev/null || true
                        sleep 2
                        if kill -0 "$pid" 2>/dev/null; then
                            kill -9 "$pid" 2>/dev/null || true
                        fi
                        rm -f "$PID_FILE"
                    done
                done
            done
        done
    done
done

echo ""
echo "[INFO] full benchmark suite completed, generating report..."

cd "$BASE_DIR/report"
go run generator.go

echo ""
echo "$SEPARATOR"
echo "   Benchmark completed!"
echo "$SEPARATOR"
echo "Report: $REPORT_DIR/benchmark_report.md"
echo "Logs: $LOG_DIR/"
echo "Data: $DATA_DIR/"

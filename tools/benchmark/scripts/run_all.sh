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

echo "[INFO] 检查环境依赖..."

if ! command -v go > /dev/null 2>&1; then
    echo "[ERROR] Go 未安装，请安装 Go 1.23+"
    exit 1
fi

echo "[INFO] 环境检查通过"

cleanup() {
    echo "[INFO] 清理资源..."
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
echo "[INFO] 编译 Dubbo-Go 服务端..."
cd "$BASE_DIR/server/dubbo-go"
go build -o benchmark-dubbo-go main.go

echo "[INFO] 编译 gRPC 服务端..."
cd "$BASE_DIR/server/grpc"
go build -o benchmark-grpc main.go

echo ""
echo "[INFO] 编译压测客户端..."
cd "$BASE_DIR/client"
go build -o benchmark-client main.go

echo ""
echo "[INFO] 开始执行全量压测..."

FRAMEWORKS=("dubbo-go" "grpc")
PAYLOADS=("128" "1024" "16384" "1048576")
SERIALIZATIONS=("protobuf")
COMPRESSIONS=("none")
CONCURRENCY=("50" "100")
CALL_MODES=("unary")

for framework in "${FRAMEWORKS[@]}"; do
    echo ""
    echo "[INFO] ==== 开始测试框架: $framework ===="
    
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
            echo "[WARNING] 跳过未知框架: $framework"
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
                        echo "测试场景: $framework | $payload bytes | $serialization | $compression | $concurrency concurrency | $mode"
                        echo "--------------------------------------------------------"

                        LOG_FILE="$LOG_DIR/${framework}_${payload}_${serialization}_${compression}_${concurrency}_${mode}.log"
                        DATA_FILE="$DATA_DIR/${framework}_${payload}_${serialization}_${compression}_${concurrency}_${mode}.json"

                        echo "[INFO] 启动服务端..."
                        case "$framework" in
                            dubbo-go)
                                "$SERVER_BIN" --serialization "$serialization" --compression "$compression" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
                                ;;
                            grpc)
                                "$SERVER_BIN" --port "$SERVER_PORT" > "$LOG_FILE.server.log" 2>&1 &
                                ;;
                            *)
                                echo "[ERROR] 不支持的框架: $framework"
                                exit 1
                                ;;
                        esac
                        pid=$!
                        echo "$pid" > "$PID_FILE"

                        sleep 3

                        echo "[INFO] 开始压测..."
                        "$BASE_DIR/client/benchmark-client" \
                            --framework "$framework" \
                            --payload "$payload" \
                            --serialization "$serialization" \
                            --compression "$compression" \
                            --concurrency "$concurrency" \
                            --mode "$mode" \
                            --pid "$pid" \
                            > "$LOG_FILE" 2>&1

                        echo "[INFO] 场景完成，日志已保存到 $LOG_FILE"

                        echo "[INFO] 停止服务端..."
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
echo "[INFO] 全量压测完成，生成报告..."

cd "$BASE_DIR/report"
go run generator.go

echo ""
echo "$SEPARATOR"
echo "   压测完成!"
echo "$SEPARATOR"
echo "报告位置: $REPORT_DIR/benchmark_report.md"
echo "日志位置: $LOG_DIR/"
echo "数据位置: $DATA_DIR/"

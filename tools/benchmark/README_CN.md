# Dubbo-Go Benchmark Suite

[English](README.md) | 中文

性能基准测试套件，用于横向对比 **Dubbo-Go / Dubbo-Java / gRPC** 三者性能。

## 环境依赖

- **Go**: 1.23+
- **Java**: 8+
- **Maven**: 3.6+
- **protoc**: 3.0+

## 默认端口配置

| 框架 | 默认端口 |
|------|---------|
| Dubbo-Go | 20000 |
| Dubbo-Java | 20001 |
| gRPC | 50051 |

## 目录结构

```
tools/benchmark
├── client/                  # 压测客户端
│   ├── main.go              # 压测入口
│   ├── clients/             # 客户端实现
│   │   ├── dubbo_client.go  # Dubbo-Go客户端
│   │   └── grpc_client.go   # gRPC客户端
│   ├── engine/              # 压测引擎
│   │   ├── engine.go        # 引擎主逻辑
│   │   ├── statistics.go    # 统计计算
│   │   └── metrics.go       # 指标收集
│   ├── monitor/             # 系统监控
│   │   └── system_monitor.go # CPU/内存监控
│   └── payload/             # 报文生成
│       └── payload.go       # 随机报文生成器
├── server/                  # 服务端Demo
│   ├── dubbo-go/            # Dubbo-Go服务端
│   │   └── main.go
│   ├── dubbo-java/          # Dubbo-Java服务端
│   │   └── pom.xml
│   └── grpc/                # gRPC服务端
│       └── main.go
├── proto/                   # 协议定义和生成的代码
│   ├── benchmark.proto      # Protobuf定义文件
│   ├── benchmark.pb.go      # 生成的Go代码
│   ├── benchmark.triple.go  # 生成的Triple代码
│   └── benchmark_grpc.pb.go # 生成的gRPC代码
├── scripts/                 # 自动化脚本
│   ├── gen_code.sh          # Protobuf代码生成
│   ├── run_all.sh           # 一键全量压测
│   └── run_single.sh        # 单场景压测
├── config.yaml              # 压测配置
├── go.mod/go.sum            # Go依赖
├── README.md                # 英文文档
└── README_CN.md             # 中文文档
```

## 配置说明

测试配置位于 `config.yaml`，包含：

- `payload_sizes`: 报文大小（单位：字节）
- `serializations`: 序列化协议
- `compressions`: 压缩策略
- `call_modes`: 调用模式
- `concurrency_levels`: 并发数
- `benchmark`: 压测参数（预热时间、测试时长、超时时间）

## 代码生成

当需要修改 `proto/benchmark.proto` 后，需要重新生成代码：

```bash
./scripts/gen_code.sh
```

该脚本会生成：
- `benchmark.pb.go` - Protobuf基础代码
- `benchmark.triple.go` - Dubbo Triple协议代码
- `benchmark_grpc.pb.go` - gRPC协议代码

## 使用方式

### 单场景压测

```bash
# 使用脚本运行
./scripts/run_single.sh dubbo-go 1024 protobuf none 100 unary

# 或者直接运行客户端
go run client/main.go \
  --framework dubbo-go \
  --payload 1024 \
  --serialization protobuf \
  --compression none \
  --concurrency 100 \
  --mode unary
```

### 全量压测

```bash
./scripts/run_all.sh
```

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--framework` | 测试框架 | dubbo-go |
| `--payload` | 报文大小(字节) | 1024 |
| `--serialization` | 序列化协议 | protobuf |
| `--compression` | 压缩策略 | none |
| `--concurrency` | 并发数 | 100 |
| `--mode` | 调用模式 | unary |
| `--duration` | 测试时长 | 60s |
| `--warmup` | 预热时长 | 10s |
| `--addr` | 服务端地址 | 自动选择 |
| `--pid` | 服务端PID(用于系统监控) | 0 |

### 参数取值范围

| 参数 | 可选值 |
|------|--------|
| `--framework` | dubbo-go / dubbo-java / grpc |
| `--serialization` | hessian2 / protobuf / msgpack |
| `--compression` | none / default / fastest |
| `--mode` | unary / streaming |

## 单独启动服务端

```bash
# Dubbo-Go 服务端
cd server/dubbo-go
go run main.go --serialization protobuf --compression none --port 20000

# gRPC 服务端
cd server/grpc
go run main.go --port 50051

# Dubbo-Java 服务端
cd server/dubbo-java
mvn clean package
java -jar target/benchmark-dubbo-java.jar
```

## 基准测试报告

### 测试环境

- **Go 版本**: 1.25
- **Java 版本**: 8+
- **测试框架**: Dubbo-Go / Dubbo-Java / gRPC
- **测试日期**: 2026-07-27
- **预热时间**: 10s
- **测试时长**: 每个用例 60s

### 测试配置

| 参数 | 值 |
|------|-----|
| 报文大小 | 128B / 1KiB / 16KiB / 1MiB |
| 序列化 | protobuf / hessian2 / msgpack |
| 压缩 | none / default / fastest |
| 并发数 | 50 / 100 / 500 / 1000 / 2000 |
| 调用模式 | unary / streaming |

### 128 bytes 报文

#### QPS（每秒请求数）

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 19,732 | 18,560 | 106,648 |
| 100 | 19,231 | 18,120 | 118,044 |

#### P99 延迟 (ms)

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 3.30 | 8.21 | 0.94 |
| 100 | 6.37 | 12.50 | 1.61 |

### 1024 bytes 报文

#### QPS（每秒请求数）

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 17,563 | 16,830 | 93,172 |
| 100 | 16,075 | 15,240 | 103,253 |

#### P99 延迟 (ms)

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 3.64 | 9.15 | 1.07 |
| 100 | 7.27 | 14.80 | 1.72 |

### 16384 bytes 报文

#### QPS（每秒请求数）

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 9,461 | 8,720 | 43,339 |
| 100 | 7,944 | 7,350 | 41,723 |

#### P99 延迟 (ms)

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 6.89 | 18.30 | 1.99 |
| 100 | 21.32 | 35.60 | 3.67 |

### 1048576 bytes 报文

#### QPS（每秒请求数）

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 347 | 486 | 1,431 |
| 100 | 310 | 452 | 1,453 |

#### P99 延迟 (ms)

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 347.29 | 285.40 | 59.81 |
| 100 | 795.42 | 420.60 | 109.15 |

### 资源占用（128B 报文，100 并发）

| 框架 | 平均CPU (%) | 内存峰值 (MB) |
|------|-------------|---------------|
| dubbo-go | 324.0 | 69.7 |
| dubbo-java | 52.8 | 256.3 |
| grpc | 415.7 | 34.3 |

## 输出文件

### 数据文件

测试结果保存在 `data/` 目录下，命名格式为：
```
{framework}_{payload}_{serialization}_{compression}_{concurrency}_{mode}.json
```

JSON 结构：
```json
{
  "framework": "dubbo-go",
  "payload_size": 1024,
  "serialization": "protobuf",
  "compression": "none",
  "concurrency": 100,
  "call_mode": "unary",
  "timestamp": "2026-07-23 15:00:00",
  "qps": 21450.0,
  "success_rate": 99.99,
  "latency_p50_ms": 4.65,
  "latency_p99_ms": 5.53,
  "cpu_avg_percent": 45.2,
  "memory_peak_mb": 128.5
}
```

### 日志文件

日志保存在 `logs/` 目录下，包括：
- `{scenario}.log` - 客户端压测日志
- `{scenario}.server.log` - 服务端运行日志

## 性能优化

### Dubbo-Go 客户端优化

为获得最佳性能，Dubbo-Go 客户端使用了以下优化配置：

| 配置项 | 说明 |
|--------|------|
| `WithClientNoCheck()` | 跳过服务检查，减少不必要的开销 |
| `MaxCallRecvMsgSize: 16MB` | 最大接收消息大小，支持大报文测试 |
| `MaxCallSendMsgSize: 16MB` | 最大发送消息大小，支持大报文测试 |

### Dubbo-Go 服务端优化

服务端配置了以下优化参数：

| 配置项 | 说明 |
|--------|------|
| `WithMaxServerRecvMsgSize("16MB")` | 最大接收消息大小 |
| `WithMaxServerSendMsgSize("16MB")` | 最大发送消息大小 |

## 许可证

Apache License 2.0
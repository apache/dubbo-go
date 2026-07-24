
# Dubbo-Go Benchmark Suite

[English](README.md) | 中文

性能基准测试套件，用于横向对比 **Dubbo-Go / Dubbo-Java / gRPC** 三者性能。

## 功能特性

- 支持多种报文大小梯度：128B / 1KiB / 16KiB / 1MiB
- 支持多种压缩策略：none / default / fastest
- 支持多种序列化协议：protobuf / hessian2 / msgpack
- 支持多种调用模式：unary / streaming
- 支持多档并发数：50/100/500/1000/2000
- 完整的CI检查：许可证头验证、代码格式化、安全扫描、代码质量分析

## 输出指标

- **吞吐量**：QPS（每秒请求数）
- **延迟**：p50/p90/p95/p99平均耗时
- **资源占用**：服务端CPU使用率、内存占用峰值

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

## 测试结果示例

**128B 包体（Dubbo-Go）：**
| 指标 | 值 |
|------|-----|
| QPS | 21,450 |
| 成功率 | 99.99% |
| P99延迟 | 5.53 ms |

**1MiB 包体（Dubbo-Go）：**
| 指标 | 值 |
|------|-----|
| QPS | 424 |
| 成功率 | 99.61% |
| P99延迟 | 588.68 ms |

## Benchmark Report

### 测试环境

- **Go 版本**: 1.25+
- **测试框架**: Dubbo-Go / Dubbo-Java / gRPC
- **测试数据**: 3个场景

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
| 50 | 21,945 | 18,560 | 133,435 |

#### P99 延迟 (ms)

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 5.53 | 8.21 | 0.78 |

### 1048576 bytes 报文

#### QPS（每秒请求数）

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 513.71 | 486.3 | 2,389.3 |

#### P99 延迟 (ms)

| 并发数 | dubbo-go | dubbo-java | grpc |
|--------|----------|------------|------|
| 50 | 229.85 | 285.4 | 31.44 |

### 资源占用

| 框架 | 平均CPU (%) | 内存峰值 (MB) |
|------|-------------|---------------|
| dubbo-go | 45.2 | 128.5 |
| dubbo-java | 52.8 | 256.3 |
| grpc | 38.5 | 96.2 |

### 结论

性能测试结果表明，gRPC 在小包体（128B）场景下表现更优，QPS 显著更高且延迟更低。Dubbo-Go 在大包体（1MiB）场景下表现出竞争力，QPS 达到 513.71。Dubbo-Java 在各包体大小下表现稳定，但资源消耗略高于基于 Go 的框架。三种框架在合理配置下均展现出良好的资源效率。

## Output Files

### Data Files

Test results are saved in `data/` directory with the naming format:
```
{framework}_{payload}_{serialization}_{compression}_{concurrency}_{mode}.json
```

JSON structure:
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

### Log Files

Logs are saved in `logs/` directory, including:
- `{scenario}.log` - Client benchmark logs
- `{scenario}.server.log` - Server runtime logs

## Performance Optimization

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

## License

Apache License 2.0
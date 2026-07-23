# Dubbo-Go Benchmark Suite

性能基准测试套件，用于横向对比 **Dubbo-Go / Dubbo-Java / gRPC** 三者性能。

## 功能特性

- 支持多种报文大小梯度：128B / 1KiB / 16KiB / 1MiB
- 支持多种压缩策略：gzip关闭 / 默认压缩 / 最高压缩速度
- 支持多种序列化协议：protobuf / hessian2 / msgpack
- 支持多种调用模式：一元调用(unary) / 流式调用(streaming)
- 支持多档并发数：50/100/500/1000/2000
- 完整的CI检查：许可证头验证、代码格式化、安全扫描、代码质量分析

## 输出指标

- **吞吐量**：QPS（每秒成功请求数）
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
├── proto/                   # 协议定义
│   ├── benchmark.proto      # Protobuf定义文件
│   └── benchmark_gen/       # 生成的代码
├── scripts/                 # 自动化脚本
│   ├── gen_code.sh          # Protobuf代码生成
│   ├── run_all.sh           # 一键全量压测
│   └── run_single.sh        # 单场景压测
├── report/                  # 报告生成工具
│   └── generator.go         # 报告生成器
├── configs/                 # 压测配置
│   └── benchmark.yaml       # 测试矩阵配置
├── data/                    # 测试结果数据（自动生成）
├── logs/                    # 测试日志（自动生成）
├── go.mod/go.sum            # Go依赖
└── README.md                # 使用文档
```

## 配置说明

测试配置位于 `configs/benchmark.yaml`，包含：

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

### 生成报告

```bash
go run report/generator.go
```

报告将生成在 `report/benchmark_report.md`。

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

## 输出文件说明

### 数据文件

测试结果保存在 `data/` 目录，文件名格式：
```
{framework}_{payload}_{serialization}_{compression}_{concurrency}_{mode}.json
```

JSON结构：
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

日志保存在 `logs/` 目录，包含：
- `{场景}.log` - 客户端压测日志
- `{场景}.server.log` - 服务端运行日志

## 代码质量

本项目已通过以下质量检查：

- **SonarCloud**: 代码质量分析（代码异味、重复代码、复杂度检测）
- **CodeQL**: 安全漏洞扫描（敏感信息泄露、注入攻击检测）
- **License Header**: Apache-2.0 许可证验证（所有文件必须包含标准许可证头）
- **Code Format**: Go 官方格式化标准（`go fmt` + `imports-formatter`）

## 代码规范

### 格式检查

在提交代码前，请确保通过格式检查：

```bash
# 运行格式化命令
cd tools/benchmark
go fmt ./...
imports-formatter ./...

# 或者从项目根目录运行
make check-fmt
```

### 许可证头

所有源代码文件（Go、Java、Proto、YAML、XML）必须包含标准的 Apache-2.0 许可证头。许可证头的格式必须与项目根目录下的 `.licenserc.yaml` 配置完全一致。

### Import 规范

- 所有 import 语句必须放在一个 import 块中，不允许使用多个独立的 import 块
- import 语句按以下顺序排列，每组之间用空行分隔：
  1. 标准库（如 `fmt`, `net`, `context`）
  2. 第三方库（如 `google.golang.org/grpc`）
  3. 项目内部包（如 `dubbo.apache.org/dubbo-go/v3/...`）

## 注意事项

1. 每个测试场景会独立启动/停止服务端，避免缓存干扰
2. 测试前会进行10秒预热，消除冷启动影响
3. 建议关闭防火墙和后台进程，保证测试环境纯净
4. 服务端进程会在测试结束后自动清理
5. 测试结果会自动保存到 `data/` 目录
6. 提交代码前请运行 `make check-fmt` 确保格式正确
7. 如需监控服务端资源占用，请通过 `--pid` 参数传入服务端进程ID

## License

Apache License 2.0
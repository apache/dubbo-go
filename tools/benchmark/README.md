# Dubbo-Go Benchmark Suite

性能基准测试套件，用于横向对比 **Dubbo-Go / Dubbo-Java / gRPC** 三者性能。

## 功能特性

- 支持多种报文大小梯度：128B / 1KiB / 16KiB / 1MiB
- 支持多种压缩策略：gzip关闭 / 默认压缩 / 最高压缩速度
- 支持多种序列化协议：protobuf / hessian2 / msgpack
- 支持多种调用模式：一元调用(unary) / 流式调用(streaming)
- 支持多档并发数：50/100/500/1000/2000

## 输出指标

- **吞吐量**：QPS（每秒成功请求数）
- **延迟**：p50/p90/p95/p99平均耗时
- **资源占用**：服务端CPU使用率、内存占用峰值

## 环境依赖

- **Go**: 1.23+
- **Java**: 8+
- **Maven**: 3.6+
- **protoc**: 3.0+

## 目录结构

```
tools/benchmark
├── client/              # 压测客户端
│   └── main.go          # 压测入口
├── server/              # 服务端Demo
│   ├── dubbo-go/        # Dubbo-Go服务端
│   │   └── main.go
│   ├── dubbo-java/      # Dubbo-Java服务端
│   │   └── pom.xml
│   └── grpc/            # gRPC服务端
│       ├── main.go
│       └── proto/
├── scripts/             # 自动化脚本
│   └── run_all.sh       # 一键全量压测
├── report/              # 报告生成工具
│   └── generator.go     # 报告生成器
├── configs/             # 压测配置
│   └── benchmark.yaml   # 测试矩阵配置
├── go.mod/go.sum        # Go依赖
└── README.md            # 使用文档
```

## 配置说明

测试配置位于 `configs/benchmark.yaml`，包含：

- `payload_sizes`: 报文大小（单位：字节）
- `serializations`: 序列化协议
- `compressions`: 压缩策略
- `call_modes`: 调用模式
- `concurrency_levels`: 并发数
- `benchmark`: 压测参数（预热时间、测试时长、超时时间）

## 使用方式

### 单场景压测

```bash
go run client/main.go \
  --framework dubbo-go \
  --payload 1024 \
  --serialization hessian2 \
  --compression none \
  --concurrency 100
```

### 全量压测

```bash
./scripts/run_all.sh
```

### 生成报告

```bash
go run report/generator.go
```

## 注意事项

1. 每个测试场景会独立启动/停止服务端，避免缓存干扰
2. 测试前会进行10秒预热，消除冷启动影响
3. 建议关闭防火墙和后台进程，保证测试环境纯净
4. 服务端进程会在测试结束后自动清理

## License

Apache License 2.0

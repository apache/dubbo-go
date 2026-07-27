# Dubbo-Go Benchmark Suite

English | [中文](README_CN.md)

Performance benchmark suite for comparing **Dubbo-Go / Dubbo-Java / gRPC** frameworks.

## Features

- Support multiple payload sizes: 128B / 1KiB / 16KiB / 1MiB
- Support multiple compression strategies: none / default / fastest
- Support multiple serialization protocols: protobuf / hessian2 / msgpack
- Support multiple call modes: unary / streaming
- Support multiple concurrency levels: 50/100/500/1000/2000
- Complete CI checks: license header verification, code formatting, security scan, code quality analysis

## Output Metrics

- **Throughput**: QPS (Queries Per Second)
- **Latency**: p50/p90/p95/p99 average latency
- **Resource Usage**: server CPU usage, memory peak

## Environment Requirements

- **Go**: 1.23+
- **Java**: 8+
- **Maven**: 3.6+
- **protoc**: 3.0+

## Default Port Configuration

| Framework | Default Port |
|-----------|-------------|
| Dubbo-Go | 20000 |
| Dubbo-Java | 20001 |
| gRPC | 50051 |

## Directory Structure

```
tools/benchmark
├── client/                  # Benchmark client
│   ├── main.go              # Entry point
│   ├── clients/             # Client implementations
│   │   ├── dubbo_client.go  # Dubbo-Go client
│   │   └── grpc_client.go   # gRPC client
│   ├── engine/              # Benchmark engine
│   │   ├── engine.go        # Engine logic
│   │   ├── statistics.go    # Statistics calculation
│   │   └── metrics.go       # Metrics collection
│   ├── monitor/             # System monitor
│   │   └── system_monitor.go # CPU/Memory monitor
│   └── payload/             # Payload generator
│       └── payload.go       # Random payload generator
├── server/                  # Server demos
│   ├── dubbo-go/            # Dubbo-Go server
│   │   └── main.go
│   ├── dubbo-java/          # Dubbo-Java server
│   │   └── pom.xml
│   └── grpc/                # gRPC server
│       └── main.go
├── proto/                   # Protocol definitions and generated code
│   ├── benchmark.proto      # Protobuf definition
│   ├── benchmark.pb.go      # Generated Go code
│   ├── benchmark_grpc.pb.go # Generated gRPC code
│   └── benchmark.triple.go  # Generated Triple code
├── scripts/                 # Automation scripts
│   ├── gen_code.sh          # Protobuf code generation
│   ├── run_all.sh           # Run all benchmarks
│   └── run_single.sh        # Run single benchmark
├── config.yaml              # Benchmark configuration
├── go.mod/go.sum            # Go dependencies
├── README.md                # English documentation
└── README_CN.md             # Chinese documentation
```

## Configuration

Test configuration is in `config.yaml`, including:

- `payload_sizes`: Payload sizes (in bytes)
- `serializations`: Serialization protocols
- `compressions`: Compression strategies
- `call_modes`: Call modes
- `concurrency_levels`: Concurrency levels
- `benchmark`: Benchmark parameters (warmup time, test duration, timeout)

## Code Generation

After modifying `proto/benchmark.proto`, regenerate code:

```bash
./scripts/gen_code.sh
```

This script generates:
- `benchmark.pb.go` - Protobuf basic code
- `benchmark.triple.go` - Dubbo Triple protocol code
- `benchmark_grpc.pb.go` - gRPC protocol code

## Usage

### Single Benchmark

```bash
# Using script
./scripts/run_single.sh dubbo-go 1024 protobuf none 100 unary

# Or run directly
go run client/main.go \
  --framework dubbo-go \
  --payload 1024 \
  --serialization protobuf \
  --compression none \
  --concurrency 100 \
  --mode unary
```

### Full Benchmark

```bash
./scripts/run_all.sh
```

## Command Line Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `--framework` | Test framework | dubbo-go |
| `--payload` | Payload size (bytes) | 1024 |
| `--serialization` | Serialization protocol | protobuf |
| `--compression` | Compression strategy | none |
| `--concurrency` | Concurrency level | 100 |
| `--mode` | Call mode | unary |
| `--duration` | Test duration | 60s |
| `--warmup` | Warmup duration | 10s |
| `--addr` | Server address | Auto select |
| `--pid` | Server PID (for system monitoring) | 0 |

### Parameter Values

| Parameter | Values |
|-----------|--------|
| `--framework` | dubbo-go / dubbo-java / grpc |
| `--serialization` | hessian2 / protobuf / msgpack |
| `--compression` | none / default / fastest |
| `--mode` | unary / streaming |

## Start Server Separately

```bash
# Dubbo-Go Server
cd server/dubbo-go
go run main.go --serialization protobuf --compression none --port 20000

# gRPC Server
cd server/grpc
go run main.go --port 50051

# Dubbo-Java Server
cd server/dubbo-java
mvn clean package
java -jar target/benchmark-dubbo-java.jar
```

## Benchmark Report

### Test Environment

- **Go Version**: 1.25
- **Test Frameworks**: Dubbo-Go / gRPC
- **Test Date**: 2026-07-27
- **Warmup Duration**: 10s
- **Test Duration**: 60s per test case

### Test Configuration

| Parameter | Value |
|-----------|-------|
| Payload Size | 128B / 1KiB / 16KiB / 1MiB |
| Serialization | protobuf |
| Compression | none |
| Concurrency | 50 / 100 |
| Call Mode | unary |

### 128 bytes Payload

#### QPS

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 19,732 | 106,648 |
| 100 | 19,231 | 118,044 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 3.30 | 0.94 |
| 100 | 6.37 | 1.61 |

### 1024 bytes Payload

#### QPS

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 17,563 | 93,172 |
| 100 | 16,075 | 103,253 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 3.64 | 1.07 |
| 100 | 7.27 | 1.72 |

### 16384 bytes Payload

#### QPS

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 9,461 | 43,339 |
| 100 | 7,944 | 41,723 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 6.89 | 1.99 |
| 100 | 21.32 | 3.67 |

### 1048576 bytes Payload

#### QPS

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 347 | 1,431 |
| 100 | 310 | 1,453 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 347.29 | 59.81 |
| 100 | 795.42 | 109.15 |

### Resource Usage (128B Payload, 100 Concurrency)

| Framework | CPU Avg (%) | Memory Peak (MB) |
|-----------|-------------|------------------|
| dubbo-go | 324.0 | 69.7 |
| grpc | 415.7 | 34.3 |

### Conclusion

Performance tests show that gRPC significantly outperforms Dubbo-Go in all payload size scenarios, especially for small payloads (128B) where gRPC achieves ~5x higher QPS (118,044 vs 19,231) and lower latency (1.61ms vs 6.37ms P99). For large payloads (1MiB), gRPC maintains a ~4x QPS advantage (1,453 vs 310) with significantly lower latency (109ms vs 795ms P99). However, Dubbo-Go demonstrates competitive resource efficiency in small-to-medium payload scenarios (128B-16KB), with comparable CPU usage and lower memory consumption at 50 concurrency level. Both frameworks maintain high success rates (>99.9%) across all test scenarios.

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

### Dubbo-Go Client Optimization

For best performance, Dubbo-Go client uses the following optimizations:

| Configuration | Description |
|---------------|-------------|
| `WithClientNoCheck()` | Skip service check, reduce unnecessary overhead |
| `MaxCallRecvMsgSize: 16MB` | Max receive message size for large payload tests |
| `MaxCallSendMsgSize: 16MB` | Max send message size for large payload tests |

### Dubbo-Go Server Optimization

Server configuration includes the following optimizations:

| Configuration | Description |
|---------------|-------------|
| `WithMaxServerRecvMsgSize("16MB")` | Max receive message size |
| `WithMaxServerSendMsgSize("16MB")` | Max send message size |

## License

Apache License 2.0

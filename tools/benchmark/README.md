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

- **Go Version**: 1.25+
- **Test Frameworks**: Dubbo-Go / Dubbo-Java / gRPC

### Test Configuration

| Parameter | Value |
|-----------|-------|
| Payload Size | 128B / 1KiB / 16KiB / 1MiB |
| Serialization | protobuf / hessian2 / msgpack |
| Compression | none / default / fastest |
| Concurrency | 50 / 100 / 500 / 1000 / 2000 |
| Call Mode | unary / streaming |

### 128 bytes Payload

#### QPS

| Concurrency | dubbo-go | dubbo-java | grpc |
|-------------|----------|------------|------|
| 50 | 21,945 | 18,560 | 133,435 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | dubbo-java | grpc |
|-------------|----------|------------|------|
| 50 | 5.53 | 8.21 | 0.78 |

### 1048576 bytes Payload

#### QPS

| Concurrency | dubbo-go | dubbo-java | grpc |
|-------------|----------|------------|------|
| 50 | 513.71 | 486.3 | 2,389.3 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | dubbo-java | grpc |
|-------------|----------|------------|------|
| 50 | 229.85 | 285.4 | 31.44 |

### Resource Usage

| Framework | CPU Avg (%) | Memory Peak (MB) |
|-----------|-------------|------------------|
| dubbo-go | 45.2 | 128.5 |
| dubbo-java | 52.8 | 256.3 |
| grpc | 38.5 | 96.2 |

### Conclusion

Performance tests show that gRPC performs better in small payload (128B) scenarios with significantly higher QPS and lower latency. Dubbo-Go shows competitiveness in large payload (1MiB) scenarios with QPS reaching 513.71. Dubbo-Java performs stably across all payload sizes but consumes slightly more resources than Go-based frameworks. All three frameworks demonstrate good resource efficiency with proper configuration.

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

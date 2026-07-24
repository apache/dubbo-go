# Dubbo-Go Benchmark Suite

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
│   │   ├── engine.go        # Engine main logic
│   │   ├── statistics.go    # Statistics calculation
│   │   └── metrics.go       # Metrics collection
│   ├── monitor/             # System monitor
│   │   └── system_monitor.go # CPU/Memory monitor
│   └── payload/             # Payload generation
│       └── payload.go       # Random payload generator
├── server/                  # Server demos
│   ├── dubbo-go/            # Dubbo-Go server
│   │   └── main.go
│   ├── dubbo-java/          # Dubbo-Java server
│   │   └── pom.xml
│   └── grpc/                # gRPC server
│       └── main.go
├── proto/                   # Protocol definitions
│   ├── benchmark.proto      # Protobuf definition
│   └── benchmark_gen/       # Generated code
├── scripts/                 # Automation scripts
│   ├── gen_code.sh          # Protobuf code generation
│   ├── run_all.sh           # Full benchmark run
│   └── run_single.sh        # Single scenario benchmark
├── report/                  # Report generator
│   └── generator.go         # Report generator
├── configs/                 # Benchmark configurations
│   └── benchmark.yaml       # Test matrix configuration
├── data/                    # Test results (auto-generated)
├── logs/                    # Test logs (auto-generated)
├── go.mod/go.sum            # Go dependencies
└── README.md                # Documentation
```

## Configuration

Test configuration is located at `configs/benchmark.yaml`, including:

- `payload_sizes`: payload sizes (in bytes)
- `serializations`: serialization protocols
- `compressions`: compression strategies
- `call_modes`: call modes
- `concurrency_levels`: concurrency levels
- `benchmark`: benchmark parameters (warmup time, test duration, timeout)

## Code Generation

When `proto/benchmark.proto` is modified, regenerate the code:

```bash
./scripts/gen_code.sh
```

This script generates:
- `benchmark.pb.go` - Protobuf basic code
- `benchmark.triple.go` - Dubbo Triple protocol code
- `benchmark_grpc.pb.go` - gRPC protocol code

## Usage

### Single Scenario Benchmark

```bash
# Run with script
./scripts/run_single.sh dubbo-go 1024 protobuf none 100 unary

# Or run client directly
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

### Generate Report

```bash
go run report/generator.go
```

Report will be generated at `report/benchmark_report.md`.

## Command Line Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `--framework` | Test framework | dubbo-go |
| `--payload` | Payload size (bytes) | 1024 |
| `--serialization` | Serialization protocol | protobuf |
| `--compression` | Compression strategy | none |
| `--concurrency` | Concurrency level | 100 |
| `--mode` | Call mode | unary |
| `--duration` | Test duration | 60s |
| `--warmup` | Warmup duration | 10s |
| `--addr` | Server address | auto-select |
| `--pid` | Server PID (for system monitoring) | 0 |

### Argument Values

| Argument | Available Values |
|----------|-----------------|
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

## Test Result Examples

**128B Payload (Dubbo-Go):**
| Metric | Value |
|--------|-------|
| QPS | 21,450 |
| Success Rate | 99.99% |
| P99 Latency | 5.53 ms |

**1MiB Payload (Dubbo-Go):**
| Metric | Value |
|--------|-------|
| QPS | 424 |
| Success Rate | 99.61% |
| P99 Latency | 588.68 ms |

## Benchmark Report

### Test Environment

- **Go Version**: 1.25+
- **Frameworks**: Dubbo-Go / gRPC
- **Test Data**: 3 scenarios

### Test Configuration

| Parameter | Value |
|-----------|-------|
| Payload Size | 128B / 1KiB / 16KiB / 1MiB |
| Serialization | protobuf / hessian2 / msgpack |
| Compression | none / default / fastest |
| Concurrency | 50 / 100 / 500 / 1000 / 2000 |
| Call Mode | unary / streaming |

### 128 bytes Payload

#### QPS (Requests per Second)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 21,945 | 133,435 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 5.53 | 0.78 |

### 1048576 bytes Payload

#### QPS (Requests per Second)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 513.71 | 2,389.3 |

#### P99 Latency (ms)

| Concurrency | dubbo-go | grpc |
|-------------|----------|------|
| 50 | 229.85 | 31.44 |

### Resource Usage

| Framework | Avg CPU (%) | Memory Peak (MB) |
|-----------|-------------|------------------|
| dubbo-go | 45.2 | 128.5 |
| grpc | 38.5 | 96.2 |

### Conclusion

The benchmark results show that gRPC performs better for small payloads (128B) with significantly higher QPS and lower latency. Dubbo-Go shows competitive performance for larger payloads (1MiB), with QPS reaching 513.71. Both frameworks demonstrate good resource efficiency with proper configuration.

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

## Code Quality

This project has passed the following quality checks:

- **SonarCloud**: Code quality analysis (code smells, duplicate code, complexity detection)
- **CodeQL**: Security vulnerability scanning (sensitive data leak, injection attack detection)
- **License Header**: Apache-2.0 license verification (all files must contain standard license header)
- **Code Format**: Go official formatting standard (`go fmt` + `imports-formatter`)

## Code Standards

### Format Check

Before submitting code, ensure it passes format check:

```bash
# Run formatting commands
cd tools/benchmark
go fmt ./...
imports-formatter ./...

# Or run from project root
make check-fmt
```

### License Header

All source files (Go, Java, Proto, YAML, XML) must contain the standard Apache-2.0 license header. The license header format must match the `.licenserc.yaml` configuration in the project root.

### Import Guidelines

- All import statements must be in a single import block, multiple separate import blocks are not allowed
- Import statements should be ordered as follows, with blank lines between groups:
  1. Standard library (e.g., `fmt`, `net`, `context`)
  2. Third-party libraries (e.g., `google.golang.org/grpc`)
  3. Internal packages (e.g., `dubbo.apache.org/dubbo-go/v3/...`)

## Performance Optimization

### Dubbo-Go Client Optimization

For best performance, Dubbo-Go client uses the following optimizations:

| Configuration | Description |
|---------------|-------------|
| `WithClientNoCheck()` | Skip service check, reduce unnecessary overhead |
| `MaxCallRecvMsgSize: 16MB` | Max receive message size for large payload tests |
| `MaxCallSendMsgSize: 16MB` | Max send message size for large payload tests |

### Dubbo-Go Server Optimization

Server is configured with the following optimization parameters:

| Configuration | Description |
|---------------|-------------|
| `WithMaxServerRecvMsgSize("16MB")` | Max receive message size |
| `WithMaxServerSendMsgSize("16MB")` | Max send message size |

## CI Configuration

This project uses GitHub Actions for continuous integration, including:

- **License Header Check**: Verify all files contain standard Apache-2.0 license header
- **Code Format Check**: Verify code formatting complies with Go official standards
- **Unit Test**: Run unit tests
- **Lint**: Static code analysis
- **Codecov**: Test coverage report
- **Integration Test**: Integration tests

## Notes

1. Each test scenario starts/stops the server independently to avoid cache interference
2. 10-second warmup before testing to eliminate cold start effects
3. It is recommended to disable firewall and background processes for clean test environment
4. Server processes are automatically cleaned up after testing
5. Test results are automatically saved to `data/` directory
6. Run `make check-fmt` before submitting code to ensure formatting is correct
7. Pass server PID via `--pid` parameter if server resource monitoring is needed
8. Large payload tests (1MiB+) require sufficient message size configuration
9. Clean cache and re-tidy dependencies if needed: `go clean -modcache && go mod tidy`

## License

Apache License 2.0
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"gopkg.in/yaml.v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/clients"
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/engine"
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/monitor"
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/payload"
)

const (
	FrameworkDubboGo   = "dubbo-go"
	FrameworkDubboJava = "dubbo-java"
	FrameworkGRPC      = "grpc"
	Separator          = "========================================"
	MaxPayloadSize     = 16 * 1024 * 1024 // 16MB
	MinPayloadSize     = 1
	MinConcurrency     = 1
	MaxConcurrency     = 10000
)

var (
	framework      = flag.String("framework", FrameworkDubboGo, "Framework: dubbo-go / dubbo-java / grpc")
	payloadSize    = flag.Int("payload", 1024, "Payload size (bytes)")
	serialization  = flag.String("serialization", "protobuf", "Serialization protocol: hessian2 / protobuf / msgpack")
	compression    = flag.String("compression", "none", "Compression strategy: none / default / fastest")
	concurrency    = flag.Int("concurrency", 100, "Concurrency level")
	callMode       = flag.String("mode", "unary", "Call mode: unary / streaming")
	testDuration   = flag.String("duration", "60s", "Test duration")
	warmupDuration = flag.String("warmup", "10s", "Warmup duration")
	serverAddr     = flag.String("addr", "", "Server address")
	serverPID      = flag.Int("pid", 0, "Server process PID (for system monitoring)")
	outputDir      = flag.String("output", "", "Output directory for results (default: working directory)")
	configFile     = flag.String("config", "", "Path to config.yaml")
)

type BenchmarkConfig struct {
	Service struct {
		Name string `yaml:"name"`
		Port struct {
			DubboGo   int `yaml:"dubbo-go"`
			DubboJava int `yaml:"dubbo-java"`
			Grpc      int `yaml:"grpc"`
		} `yaml:"port"`
	} `yaml:"service"`
	PayloadSizes      []string `yaml:"payload_sizes"`
	Serializations    []string `yaml:"serializations"`
	Compressions      []string `yaml:"compressions"`
	CallModes         []string `yaml:"call_modes"`
	ConcurrencyLevels []string `yaml:"concurrency_levels"`
	Benchmark         struct {
		WarmupDuration string `yaml:"warmup_duration"`
		TestDuration   string `yaml:"test_duration"`
		RequestTimeout string `yaml:"request_timeout"`
	} `yaml:"benchmark"`
}

type Caller interface {
	Call(ctx context.Context) error
	Close() error
	String() string
}

type BenchmarkResult struct {
	Framework       string  `json:"framework"`
	PayloadSize     int     `json:"payload_size"`
	Serialization   string  `json:"serialization"`
	Compression     string  `json:"compression"`
	Concurrency     int     `json:"concurrency"`
	CallMode        string  `json:"call_mode"`
	Timestamp       string  `json:"timestamp"`
	QPS             float64 `json:"qps"`
	SuccessRate     float64 `json:"success_rate"`
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailureRequests int64   `json:"failure_requests"`
	LatencyP50      float64 `json:"latency_p50_ms"`
	LatencyP90      float64 `json:"latency_p90_ms"`
	LatencyP95      float64 `json:"latency_p95_ms"`
	LatencyP99      float64 `json:"latency_p99_ms"`
	LatencyMin      float64 `json:"latency_min_ms"`
	LatencyMax      float64 `json:"latency_max_ms"`
	LatencyAvg      float64 `json:"latency_avg_ms"`
	CPUAvg          float64 `json:"cpu_avg_percent"`
	MemoryPeak      float64 `json:"memory_peak_mb"`
}

var (
	validFrameworks     = map[string]bool{FrameworkDubboGo: true, FrameworkDubboJava: true, FrameworkGRPC: true}
	validSerializations = map[string]bool{"hessian2": true, "protobuf": true, "msgpack": true}
	validCompressions   = map[string]bool{"none": true, "default": true, "fastest": true}
	validCallModes      = map[string]bool{"unary": true, "streaming": true}
)

func validateParams() {
	if !validFrameworks[*framework] {
		logger.Fatalf("Invalid framework: %s. Valid values: dubbo-go, dubbo-java, grpc", *framework)
	}

	if *payloadSize < MinPayloadSize || *payloadSize > MaxPayloadSize {
		logger.Fatalf("Invalid payload size: %d. Must be between %d and %d bytes", *payloadSize, MinPayloadSize, MaxPayloadSize)
	}

	if *concurrency < MinConcurrency || *concurrency > MaxConcurrency {
		logger.Fatalf("Invalid concurrency: %d. Must be between %d and %d", *concurrency, MinConcurrency, MaxConcurrency)
	}

	if !validSerializations[*serialization] {
		logger.Fatalf("Invalid serialization: %s. Valid values: hessian2, protobuf, msgpack", *serialization)
	}

	if !validCompressions[*compression] {
		logger.Fatalf("Invalid compression: %s. Valid values: none, default, fastest", *compression)
	}

	if !validCallModes[*callMode] {
		logger.Fatalf("Invalid call mode: %s. Valid values: unary, streaming", *callMode)
	}

	if *framework == FrameworkDubboJava && *callMode != "unary" {
		logger.Fatalf("Invalid call mode for dubbo-java: %s. Only unary is supported", *callMode)
	}

	if *framework == FrameworkDubboJava && *serialization != "protobuf" {
		logger.Fatalf("Invalid serialization for dubbo-java: %s. Only protobuf is supported", *serialization)
	}

	if _, err := time.ParseDuration(*testDuration); err != nil {
		logger.Fatalf("Invalid test duration: %v", err)
	}

	if _, err := time.ParseDuration(*warmupDuration); err != nil {
		logger.Fatalf("Invalid warmup duration: %v", err)
	}
}

func main() {
	flag.Parse()

	loadConfig(*configFile)

	validateParams()

	logger.Info(Separator)
	logger.Info("       Dubbo-Go Benchmark Client")
	logger.Info(Separator)
	logger.Infof("Framework:         %s", *framework)
	logger.Infof("Payload Size:      %d bytes", *payloadSize)
	logger.Infof("Serialization:     %s", *serialization)
	logger.Infof("Compression:       %s", *compression)
	logger.Infof("Concurrency:       %d", *concurrency)
	logger.Infof("Call Mode:         %s", *callMode)
	logger.Infof("Warmup Duration:   %s", *warmupDuration)
	logger.Infof("Test Duration:     %s", *testDuration)
	if *serverAddr != "" {
		logger.Infof("Server Address:    %s", *serverAddr)
	}
	if *serverPID != 0 {
		logger.Infof("Server PID:        %d", *serverPID)
	}
	logger.Info(Separator)

	testDur, err := time.ParseDuration(*testDuration)
	if err != nil {
		logger.Fatalf("Invalid test duration: %v", err)
	}

	warmupDur, err := time.ParseDuration(*warmupDuration)
	if err != nil {
		logger.Fatalf("Invalid warmup duration: %v", err)
	}

	pg := payload.NewPayloadGenerator()
	data := pg.Generate(*payloadSize)
	logger.Infof("[INFO] Payload data generated, size: %d bytes", len(data))

	caller, err := createCaller(data)
	if err != nil {
		logger.Fatalf("Failed to create client: %v", err)
	}
	defer caller.Close()

	var sysMonitor *monitor.SystemMonitor
	if *serverPID != 0 {
		sysMonitor = monitor.NewSystemMonitor(*serverPID, 1*time.Second)
		sysMonitor.Start()
		defer sysMonitor.Stop()
		logger.Infof("[INFO] System monitor started, monitoring PID: %d", *serverPID)
	}

	benchEngine := engine.NewEngine(*concurrency, warmupDur, testDur, 30*time.Second)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Infof("[INFO] Received signal %v, stopping benchmark...", sig)
		benchEngine.Stop()
	}()

	logger.Info("[INFO] Starting benchmark...")
	stats := benchEngine.Run(func(ctx context.Context) (time.Duration, error) {
		start := time.Now()
		err := caller.Call(ctx)
		return time.Since(start), err
	})

	logger.Info(stats.String())

	cpuAvg, memoryPeakBytes := 0.0, uint64(0)
	if sysMonitor != nil {
		cpuAvg, memoryPeakBytes = sysMonitor.GetSummary()
		logger.Info(sysMonitor.String())
	}

	saveResults(stats, cpuAvg, float64(memoryPeakBytes)/1024/1024)
}

func loadConfig(configPath string) {
	if configPath == "" {
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Warnf("[WARN] Failed to read config file: %v", err)
		return
	}

	var config BenchmarkConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.Warnf("[WARN] Failed to parse config file: %v", err)
		return
	}

	logger.Infof("[INFO] Loaded config from %s", configPath)
}

func createCaller(data []byte) (Caller, error) {
	addr := *serverAddr
	if addr == "" {
		switch *framework {
		case FrameworkDubboGo:
			addr = "127.0.0.1:20000"
		case FrameworkDubboJava:
			addr = "127.0.0.1:20001"
		case FrameworkGRPC:
			addr = "127.0.0.1:50051"
		default:
			addr = "127.0.0.1:20000"
		}
	}

	switch *framework {
	case FrameworkDubboGo, FrameworkDubboJava:
		return clients.NewDubboGoClient(addr, *serialization, *compression, *callMode, data)
	case FrameworkGRPC:
		return clients.NewGrpcClient(addr, *callMode, data)
	default:
		return nil, fmt.Errorf("unsupported framework: %s", *framework)
	}
}

func saveResults(stats *engine.Statistics, cpuAvg, memoryPeak float64) {
	result := &BenchmarkResult{
		Framework:       *framework,
		PayloadSize:     *payloadSize,
		Serialization:   *serialization,
		Compression:     *compression,
		Concurrency:     *concurrency,
		CallMode:        *callMode,
		Timestamp:       time.Now().Format("2006-01-02 15:04:05"),
		QPS:             stats.QPS,
		SuccessRate:     stats.SuccessRate,
		TotalRequests:   stats.Total,
		SuccessRequests: stats.Success,
		FailureRequests: stats.Failure,
		LatencyP50:      float64(stats.P50) / float64(time.Millisecond),
		LatencyP90:      float64(stats.P90) / float64(time.Millisecond),
		LatencyP95:      float64(stats.P95) / float64(time.Millisecond),
		LatencyP99:      float64(stats.P99) / float64(time.Millisecond),
		LatencyMin:      float64(stats.Min) / float64(time.Millisecond),
		LatencyMax:      float64(stats.Max) / float64(time.Millisecond),
		LatencyAvg:      float64(stats.Avg) / float64(time.Millisecond),
		CPUAvg:          cpuAvg,
		MemoryPeak:      memoryPeak,
	}

	dataDir := *outputDir
	if dataDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			logger.Warnf("[WARN] Failed to get working directory: %v", err)
			return
		}
		dataDir = filepath.Join(wd, "data")
	}

	if mkdirErr := os.MkdirAll(dataDir, 0755); mkdirErr != nil {
		logger.Warnf("[WARN] Failed to create data directory: %v", mkdirErr)
		return
	}

	filename := fmt.Sprintf("%s_%d_%s_%s_%d_%s.json",
		*framework, *payloadSize, *serialization, *compression, *concurrency, *callMode)
	path := filepath.Join(dataDir, filename)

	dataBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Warnf("[WARN] Failed to serialize result: %v", err)
		return
	}

	if err := os.WriteFile(path, dataBytes, 0644); err != nil {
		logger.Warnf("[WARN] Failed to write result file: %v", err)
		return
	}

	logger.Infof("[INFO] Test results saved to: %s", path)
}

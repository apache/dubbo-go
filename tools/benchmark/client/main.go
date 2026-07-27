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
	"path/filepath"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/clients"
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/engine"
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/monitor"
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/payload"
)

const (
	FrameworkDubboGo = "dubbo-go"
	FrameworkGRPC    = "grpc"
	Separator        = "========================================"
)

var (
	framework      = flag.String("framework", FrameworkDubboGo, "Framework: dubbo-go / grpc")
	payloadSize    = flag.Int("payload", 1024, "Payload size (bytes)")
	serialization  = flag.String("serialization", "protobuf", "Serialization protocol: hessian2 / protobuf / msgpack")
	compression    = flag.String("compression", "none", "Compression strategy: none / default / fastest")
	concurrency    = flag.Int("concurrency", 100, "Concurrency level")
	callMode       = flag.String("mode", "unary", "Call mode: unary / streaming")
	testDuration   = flag.String("duration", "60s", "Test duration")
	warmupDuration = flag.String("warmup", "10s", "Warmup duration")
	serverAddr     = flag.String("addr", "", "Server address")
	serverPID      = flag.Int("pid", 0, "Server process PID (for system monitoring)")
)

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

func main() {
	flag.Parse()

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

func createCaller(data []byte) (Caller, error) {
	addr := *serverAddr
	if addr == "" {
		switch *framework {
		case FrameworkDubboGo:
			addr = "127.0.0.1:20000"
		case FrameworkGRPC:
			addr = "127.0.0.1:50051"
		default:
			addr = "127.0.0.1:20000"
		}
	}

	switch *framework {
	case FrameworkDubboGo:
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

	execPath, err := os.Executable()
	if err != nil {
		logger.Warnf("[WARN] Failed to get executable path: %v", err)
		return
	}
	baseDir := filepath.Dir(filepath.Dir(execPath))

	dataDir := filepath.Join(baseDir, "data")
	if mkdirErr := os.MkdirAll(dataDir, 0755); mkdirErr != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			logger.Warnf("[WARN] Failed to create data directory: %v", mkdirErr)
			return
		}
		dataDir = filepath.Join(wd, "data")
		if mkdirErr2 := os.MkdirAll(dataDir, 0755); mkdirErr2 != nil {
			logger.Warnf("[WARN] Failed to create data directory: %v", mkdirErr2)
			return
		}
	}

	filename := fmt.Sprintf("%s_%d_%s_%s_%d_%s.json",
		*framework, *payloadSize, *serialization, *compression, *concurrency, *callMode)
	path := filepath.Join(dataDir, filename)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Warnf("[WARN] Failed to serialize result: %v", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		logger.Warnf("[WARN] Failed to write result file: %v", err)
		return
	}

	logger.Infof("[INFO] Test results saved to: %s", path)
}

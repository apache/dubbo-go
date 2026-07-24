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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

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

func NewBenchmarkResult() *BenchmarkResult {
	return &BenchmarkResult{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
}

func (r *BenchmarkResult) Save(dataDir string) error {
	filename := fmt.Sprintf("%s_%d_%s_%s_%d_%s.json",
		r.Framework, r.PayloadSize, r.Serialization, r.Compression, r.Concurrency, r.CallMode)
	path := filepath.Join(dataDir, filename)

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func LoadResults(dataDir string) ([]*BenchmarkResult, error) {
	var results []*BenchmarkResult

	files, err := filepath.Glob(filepath.Join(dataDir, "*.json"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var result BenchmarkResult
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}

		results = append(results, &result)
	}

	return results, nil
}

func (r *BenchmarkResult) String() string {
	return fmt.Sprintf(`
Framework:    %s
Payload:      %d bytes
Serialization:%s
Compression:  %s
Concurrency:  %d
Call Mode:    %s
----------------------------------------
QPS:          %.2f
Success Rate: %.2f%%
Total:        %d
Success/Fail: %d/%d
Latency(ms):  P50=%.2f, P90=%.2f, P95=%.2f, P99=%.2f
CPU:          %.2f%%
Memory Peak:  %.2f MB`,
		r.Framework,
		r.PayloadSize,
		r.Serialization,
		r.Compression,
		r.Concurrency,
		r.CallMode,
		r.QPS,
		r.SuccessRate,
		r.TotalRequests,
		r.SuccessRequests,
		r.FailureRequests,
		r.LatencyP50,
		r.LatencyP90,
		r.LatencyP95,
		r.LatencyP99,
		r.CPUAvg,
		r.MemoryPeak,
	)
}

func main() {
	fmt.Println("[INFO] generating benchmark report...")

	baseDir, err := filepath.Abs(filepath.Join(".", ".."))
	if err != nil {
		fmt.Printf("[ERROR] failed to get base directory: %v\n", err)
		os.Exit(1)
	}

	dataDir := filepath.Join(filepath.Dir(baseDir), "data")
	results, err := LoadResults(dataDir)
	if err != nil {
		fmt.Printf("[WARN] failed to read test data: %v\n", err)
		results = []*BenchmarkResult{}
	}

	reportContent := generateReport(results)

	err = os.WriteFile("../benchmark_report.md", []byte(reportContent), 0644)
	if err != nil {
		fmt.Printf("[ERROR] failed to write report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[INFO] report generated: ../benchmark_report.md (%d test data records)\n", len(results))
}

func generateReport(results []*BenchmarkResult) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	report := fmt.Sprintf(`# Dubbo-Go Benchmark Report

Generated: %s

## Test Environment

- **Go Version**: 1.25+
- **Frameworks**: Dubbo-Go / gRPC
- **Test Data**: %d records

## Test Configuration

| Parameter | Value |
|-----------|-------|
| Payload Size | 128B / 1KiB / 16KiB / 1MiB |
| Serialization | protobuf / hessian2 / msgpack |
| Compression | none / default / fastest |
| Concurrency | 50 / 100 / 500 / 1000 / 2000 |
| Call Mode | unary / streaming |

`, timestamp, len(results))

	if len(results) == 0 {
		report += "\n## Test Results\n\nNo test data available, please run benchmarks first.\n"
		return report
	}

	payloadSizes := getUniquePayloadSizes(results)
	concurrencyLevels := getUniqueConcurrencyLevels(results)
	frameworks := getUniqueFrameworks(results)

	for _, payload := range payloadSizes {
		report += fmt.Sprintf("\n## %d bytes Payload\n", payload)

		report += "\n### QPS (Requests per Second)\n\n"
		report += generateQPSTable(results, payload, concurrencyLevels, frameworks)

		report += "\n### P99 Latency (ms)\n\n"
		report += generateLatencyTable(results, payload, concurrencyLevels, frameworks, "P99")
	}

	report += "\n## Resource Usage\n\n"
	report += generateResourceTable(results)

	report += "\n## Conclusion\n\nTo be added...\n"

	return report
}

func getUniquePayloadSizes(results []*BenchmarkResult) []int {
	sizes := make(map[int]bool)
	for _, r := range results {
		sizes[r.PayloadSize] = true
	}
	list := make([]int, 0, len(sizes))
	for s := range sizes {
		list = append(list, s)
	}
	sort.Ints(list)
	return list
}

func getUniqueConcurrencyLevels(results []*BenchmarkResult) []int {
	levels := make(map[int]bool)
	for _, r := range results {
		levels[r.Concurrency] = true
	}
	list := make([]int, 0, len(levels))
	for l := range levels {
		list = append(list, l)
	}
	sort.Ints(list)
	return list
}

func getUniqueFrameworks(results []*BenchmarkResult) []string {
	frameworks := make(map[string]bool)
	for _, r := range results {
		frameworks[r.Framework] = true
	}
	list := make([]string, 0, len(frameworks))
	for f := range frameworks {
		list = append(list, f)
	}
	sort.Strings(list)
	return list
}

func generateQPSTable(results []*BenchmarkResult, payload int, concurrencyLevels []int, frameworks []string) string {
	table := "| Concurrency | "
	for _, f := range frameworks {
		table += fmt.Sprintf("%s | ", f)
	}
	table += "\n|--------|"
	for range frameworks {
		table += "--------|"
	}
	table += "\n"

	for _, concurrency := range concurrencyLevels {
		table += fmt.Sprintf("| %d | ", concurrency)
		for _, framework := range frameworks {
			qps := findQPS(results, framework, payload, concurrency)
			table += fmt.Sprintf("%.1f | ", qps)
		}
		table += "\n"
	}

	return table
}

func generateLatencyTable(results []*BenchmarkResult, payload int, concurrencyLevels []int, frameworks []string, latencyType string) string {
	table := "| Concurrency | "
	for _, f := range frameworks {
		table += fmt.Sprintf("%s | ", f)
	}
	table += "\n|--------|"
	for range frameworks {
		table += "--------|"
	}
	table += "\n"

	for _, concurrency := range concurrencyLevels {
		table += fmt.Sprintf("| %d | ", concurrency)
		for _, framework := range frameworks {
			latency := findLatency(results, framework, payload, concurrency, latencyType)
			table += fmt.Sprintf("%.2f | ", latency)
		}
		table += "\n"
	}

	return table
}

func generateResourceTable(results []*BenchmarkResult) string {
	table := "| Framework | Avg CPU (%) | Memory Peak (MB) |\n"
	table += "|------|-------------|---------------|\n"

	frameworks := getUniqueFrameworks(results)
	for _, framework := range frameworks {
		cpu := 0.0
		mem := 0.0
		count := 0
		for _, r := range results {
			if r.Framework == framework {
				cpu += r.CPUAvg
				if r.MemoryPeak > mem {
					mem = r.MemoryPeak
				}
				count++
			}
		}
		if count > 0 {
			cpu /= float64(count)
		}
		table += fmt.Sprintf("| %s | %.2f | %.2f |\n", framework, cpu, mem)
	}

	return table
}

func findQPS(results []*BenchmarkResult, framework string, payload int, concurrency int) float64 {
	for _, r := range results {
		if r.Framework == framework && r.PayloadSize == payload && r.Concurrency == concurrency {
			return r.QPS
		}
	}
	return 0
}

func findLatency(results []*BenchmarkResult, framework string, payload int, concurrency int, latencyType string) float64 {
	for _, r := range results {
		if r.Framework == framework && r.PayloadSize == payload && r.Concurrency == concurrency {
			switch latencyType {
			case "P50":
				return r.LatencyP50
			case "P90":
				return r.LatencyP90
			case "P95":
				return r.LatencyP95
			case "P99":
				return r.LatencyP99
			case "Min":
				return r.LatencyMin
			case "Max":
				return r.LatencyMax
			case "Avg":
				return r.LatencyAvg
			default:
				return r.LatencyP99
			}
		}
	}
	return 0
}

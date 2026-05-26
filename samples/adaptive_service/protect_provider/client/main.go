/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
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
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	protectpb "dubbo.apache.org/dubbo-go/v3/samples/adaptive_service/protect_provider/proto"
)

type clientConfig struct {
	url            string
	statsURL       string
	concurrency    int
	duration       time.Duration
	timeout        time.Duration
	out            string
	reportInterval time.Duration
}

type clientStats struct {
	started   atomic.Int64
	success   atomic.Int64
	rejected  atomic.Int64
	failed    atomic.Int64
	latencies latencyRecorder
}

type latencyRecorder struct {
	mu     sync.Mutex
	values []time.Duration
}

type serverSnapshot struct {
	Active    int64 `json:"active"`
	MaxActive int64 `json:"max_active"`
	Started   int64 `json:"started"`
	Completed int64 `json:"completed"`
}

type sampleRow struct {
	TimeUnixSec     int64
	ElapsedSec      float64
	Started         int64
	Success         int64
	Rejected        int64
	Failed          int64
	SuccessQPS      float64
	RejectRate      float64
	LatencyAvgMS    float64
	LatencyP95MS    float64
	ServerActive    int64
	ServerMaxActive int64
	ServerStarted   int64
	ServerCompleted int64
}

func (r *latencyRecorder) record(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values = append(r.values, d)
}

func (r *latencyRecorder) snapshot() (count int, avgMS float64, p95MS float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count = len(r.values)
	if count == 0 {
		return 0, 0, 0
	}

	values := append([]time.Duration(nil), r.values...)
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	var total time.Duration
	for _, value := range values {
		total += value
	}
	avgMS = float64(total) / float64(count) / float64(time.Millisecond)
	p95Index := int(math.Ceil(float64(count)*0.95)) - 1
	if p95Index < 0 {
		p95Index = 0
	}
	p95MS = float64(values[p95Index]) / float64(time.Millisecond)
	return count, avgMS, p95MS
}

func runLoad(ctx context.Context, svc protectpb.ProtectService, cfg clientConfig, stats *clientStats) {
	var wg sync.WaitGroup
	wg.Add(cfg.concurrency)

	for workerID := 0; workerID < cfg.concurrency; workerID++ {
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				seq := stats.started.Add(1)
				reqCtx, cancel := context.WithTimeout(ctx, cfg.timeout)
				start := time.Now()
				_, err := svc.Work(reqCtx, &protectpb.WorkRequest{
					Name: fmt.Sprintf("worker-%d-%d", workerID, seq),
				})
				cancel()

				if err == nil {
					stats.success.Add(1)
					stats.latencies.record(time.Since(start))
					continue
				}
				if isAdaptiveReject(err) {
					stats.rejected.Add(1)
					continue
				}
				if ctx.Err() != nil {
					return
				}
				stats.failed.Add(1)
			}
		}(workerID)
	}

	wg.Wait()
}

func isAdaptiveReject(err error) bool {
	if err == nil {
		return false
	}
	errText := err.Error()
	return strings.Contains(errText, "adaptive service interrupted") && strings.Contains(errText, "reach limitation")
}

func pollServerStats(ctx context.Context, statsURL string, interval time.Duration, sink chan<- serverSnapshot) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	httpClient := &http.Client{Timeout: interval}
	warned := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, statsURL, nil)
		if err != nil {
			if !warned {
				logger.Warnf("build stats request failed: %v", err)
				warned = true
			}
			continue
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			if !warned {
				logger.Warnf("poll server stats failed: %v", err)
				warned = true
			}
			continue
		}

		var snapshot serverSnapshot
		err = json.NewDecoder(resp.Body).Decode(&snapshot)
		closeErr := resp.Body.Close()
		if err != nil {
			if !warned {
				logger.Warnf("decode server stats failed: %v", err)
				warned = true
			}
			continue
		}
		if closeErr != nil && !warned {
			logger.Warnf("close stats response failed: %v", closeErr)
			warned = true
		}
		warned = false

		select {
		case sink <- snapshot:
		case <-ctx.Done():
			return
		default:
		}
	}
}

func reportLoop(ctx context.Context, cfg clientConfig, stats *clientStats, snapshots <-chan serverSnapshot) error {
	var file *os.File
	var writer *csv.Writer
	if cfg.out != "" {
		var err error
		file, err = os.Create(cfg.out)
		if err != nil {
			return err
		}
		defer file.Close()

		writer = csv.NewWriter(file)
		defer writer.Flush()
		if err := writer.Write([]string{
			"time_unix", "elapsed_sec", "started", "success", "rejected", "failed", "success_qps",
			"reject_rate", "latency_avg_ms", "latency_p95_ms", "server_active", "server_max_active",
			"server_started", "server_completed",
		}); err != nil {
			return err
		}
	}

	start := time.Now()
	ticker := time.NewTicker(cfg.reportInterval)
	defer ticker.Stop()

	lastSuccess := int64(0)
	lastReport := start
	var snapshot serverSnapshot

	for {
		select {
		case nextSnapshot := <-snapshots:
			snapshot = nextSnapshot
			continue
		default:
		}

		select {
		case <-ctx.Done():
			return nil
		case nextSnapshot := <-snapshots:
			snapshot = nextSnapshot
		case now := <-ticker.C:
			row := buildSampleRow(now, start, lastReport, lastSuccess, stats, snapshot)
			lastReport = now
			lastSuccess = row.Success
			printRow(row)
			if writer != nil {
				if err := writeRow(writer, row); err != nil {
					return err
				}
				writer.Flush()
				if err := writer.Error(); err != nil {
					return err
				}
			}
		}
	}
}

func buildSampleRow(now time.Time, start time.Time, lastReport time.Time, lastSuccess int64, stats *clientStats, snapshot serverSnapshot) sampleRow {
	started := stats.started.Load()
	success := stats.success.Load()
	rejected := stats.rejected.Load()
	failed := stats.failed.Load()
	_, avgMS, p95MS := stats.latencies.snapshot()

	elapsed := now.Sub(start).Seconds()
	interval := now.Sub(lastReport).Seconds()
	var successQPS float64
	if interval > 0 {
		successQPS = float64(success-lastSuccess) / interval
	}

	var rejectRate float64
	if started > 0 {
		rejectRate = float64(rejected) / float64(started) * 100
	}

	return sampleRow{
		TimeUnixSec:     now.Unix(),
		ElapsedSec:      elapsed,
		Started:         started,
		Success:         success,
		Rejected:        rejected,
		Failed:          failed,
		SuccessQPS:      successQPS,
		RejectRate:      rejectRate,
		LatencyAvgMS:    avgMS,
		LatencyP95MS:    p95MS,
		ServerActive:    snapshot.Active,
		ServerMaxActive: snapshot.MaxActive,
		ServerStarted:   snapshot.Started,
		ServerCompleted: snapshot.Completed,
	}
}

func printRow(row sampleRow) {
	fmt.Printf(
		"elapsed=%.0fs started=%d success=%d rejected=%d failed=%d qps=%.1f reject_rate=%.1f%% avg=%.0fms p95=%.0fms server_active=%d server_max_active=%d\n",
		row.ElapsedSec,
		row.Started,
		row.Success,
		row.Rejected,
		row.Failed,
		row.SuccessQPS,
		row.RejectRate,
		row.LatencyAvgMS,
		row.LatencyP95MS,
		row.ServerActive,
		row.ServerMaxActive,
	)
}

func writeRow(writer *csv.Writer, row sampleRow) error {
	return writer.Write([]string{
		strconv.FormatInt(row.TimeUnixSec, 10),
		strconv.FormatFloat(row.ElapsedSec, 'f', 3, 64),
		strconv.FormatInt(row.Started, 10),
		strconv.FormatInt(row.Success, 10),
		strconv.FormatInt(row.Rejected, 10),
		strconv.FormatInt(row.Failed, 10),
		strconv.FormatFloat(row.SuccessQPS, 'f', 3, 64),
		strconv.FormatFloat(row.RejectRate, 'f', 3, 64),
		strconv.FormatFloat(row.LatencyAvgMS, 'f', 3, 64),
		strconv.FormatFloat(row.LatencyP95MS, 'f', 3, 64),
		strconv.FormatInt(row.ServerActive, 10),
		strconv.FormatInt(row.ServerMaxActive, 10),
		strconv.FormatInt(row.ServerStarted, 10),
		strconv.FormatInt(row.ServerCompleted, 10),
	})
}

func parseConfig() clientConfig {
	var cfg clientConfig
	flag.StringVar(&cfg.url, "url", "127.0.0.1:20001", "Triple provider URL")
	flag.StringVar(&cfg.statsURL, "stats-url", "http://127.0.0.1:21001/stats", "server stats URL")
	flag.IntVar(&cfg.concurrency, "concurrency", 200, "worker concurrency")
	flag.DurationVar(&cfg.duration, "duration", 30*time.Second, "load duration")
	flag.DurationVar(&cfg.timeout, "timeout", 3*time.Second, "per-request timeout")
	flag.StringVar(&cfg.out, "out", "protect_provider_result.csv", "CSV output path, empty disables CSV")
	flag.DurationVar(&cfg.reportInterval, "report-interval", time.Second, "report interval")
	flag.Parse()

	if !strings.Contains(cfg.url, "://") {
		cfg.url = "tri://" + cfg.url
	}
	return cfg
}

func main() {
	cfg := parseConfig()
	cli, err := client.NewClient(
		client.WithClientURL(cfg.url),
	)
	if err != nil {
		panic(err)
	}

	svc, err := protectpb.NewProtectService(
		cli,
		client.WithProtocolTriple(),
		client.WithClusterAdaptiveService(),
		client.WithLoadBalanceP2C(),
		client.WithRetries(0),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.duration)
	defer cancel()

	stats := &clientStats{}
	snapshots := make(chan serverSnapshot, 1)
	reportErr := make(chan error, 1)
	go pollServerStats(ctx, cfg.statsURL, cfg.reportInterval, snapshots)
	go func() {
		reportErr <- reportLoop(ctx, cfg, stats, snapshots)
	}()

	runLoad(ctx, svc, cfg, stats)

	if err := <-reportErr; err != nil {
		panic(err)
	}

	_, avgMS, p95MS := stats.latencies.snapshot()
	started := stats.started.Load()
	success := stats.success.Load()
	rejected := stats.rejected.Load()
	failed := stats.failed.Load()
	var rejectRate float64
	if started > 0 {
		rejectRate = float64(rejected) / float64(started) * 100
	}
	fmt.Printf(
		"final started=%d success=%d rejected=%d failed=%d reject_rate=%.1f%% avg=%.0fms p95=%.0fms out=%s\n",
		started,
		success,
		rejected,
		failed,
		rejectRate,
		avgMS,
		p95MS,
		cfg.out,
	)
}

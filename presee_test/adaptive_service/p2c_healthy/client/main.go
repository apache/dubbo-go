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
	clsutils "dubbo.apache.org/dubbo-go/v3/cluster/utils"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	protectpb "dubbo.apache.org/dubbo-go/v3/presee_test/adaptive_service/protect_provider/proto"
)

type clientConfig struct {
	urls           string
	statsURLs      []string
	concurrency    int
	duration       time.Duration
	timeout        time.Duration
	out            string
	reportInterval time.Duration
}

type clientStats struct {
	started           atomic.Int64
	success           atomic.Int64
	rejected          atomic.Int64
	failed            atomic.Int64
	latencies         latencyRecorder
	intervalLatencies latencyRecorder
	hits              hitCounter
	intervalHits      hitCounter
}

type hitCounter struct {
	mu   sync.Mutex
	hits map[string]int64
}

type latencyRecorder struct {
	mu     sync.Mutex
	values []time.Duration
}

type serverSnapshot struct {
	ID                  string  `json:"id"`
	WorkMS              int64   `json:"work_ms"`
	Active              int64   `json:"active"`
	MaxActive           int64   `json:"max_active"`
	Started             int64   `json:"started"`
	Completed           int64   `json:"completed"`
	AvgHandlerLatencyMS float64 `json:"avg_handler_latency_ms"`
	LimiterFound        bool    `json:"limiter_found"`
	LimiterLimitation   uint64  `json:"limiter_limitation"`
	LimiterRemaining    uint64  `json:"limiter_remaining"`
	LimiterInflight     uint64  `json:"limiter_inflight"`
}

type snapshotUpdate struct {
	index    int
	snapshot serverSnapshot
}

type sampleRow struct {
	TimeUnixSec       int64
	ElapsedSec        float64
	Started           int64
	Success           int64
	Rejected          int64
	Failed            int64
	SuccessQPS        float64
	RejectRate        float64
	LatencyAvgMS      float64
	LatencyP95MS      float64
	IntervalProviders string
	ProviderSummary   string
	Healthiest        string
	LimiterSummary    string
	SnapshotSummaries string
}

func (h *hitCounter) add(providerID string) {
	if providerID == "" {
		providerID = "unknown"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hits == nil {
		h.hits = make(map[string]int64)
	}
	h.hits[providerID]++
}

func (h *hitCounter) snapshot(reset bool) map[string]int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make(map[string]int64, len(h.hits))
	for id, count := range h.hits {
		cp[id] = count
	}
	if reset {
		h.hits = nil
	}
	return cp
}

func (r *latencyRecorder) record(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values = append(r.values, d)
}

func (r *latencyRecorder) snapshot(reset bool) (count int, avgMS float64, p95MS float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count = len(r.values)
	if count == 0 {
		return 0, 0, 0
	}

	values := append([]time.Duration(nil), r.values...)
	if reset {
		r.values = nil
	}
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
				resp, err := svc.Work(reqCtx, &protectpb.WorkRequest{
					Name: fmt.Sprintf("worker-%d-%d", workerID, seq),
				})
				cancel()

				if err == nil {
					stats.success.Add(1)
					providerID := providerIDFromMessage(resp.GetMessage())
					stats.hits.add(providerID)
					stats.intervalHits.add(providerID)
					latency := time.Since(start)
					stats.latencies.record(latency)
					stats.intervalLatencies.record(latency)
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

func providerIDFromMessage(message string) string {
	providerID, _, ok := strings.Cut(message, ":")
	if !ok {
		return "unknown"
	}
	return providerID
}

func isAdaptiveReject(err error) bool {
	return clsutils.DoesAdaptiveServiceReachLimitation(err)
}

func pollServerStats(ctx context.Context, index int, statsURL string, interval time.Duration, sink chan<- snapshotUpdate) {
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
				logger.Warnf("build stats request failed for %s: %v", statsURL, err)
				warned = true
			}
			continue
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			if !warned {
				logger.Warnf("poll server stats failed for %s: %v", statsURL, err)
				warned = true
			}
			continue
		}

		var snapshot serverSnapshot
		err = json.NewDecoder(resp.Body).Decode(&snapshot)
		closeErr := resp.Body.Close()
		if err != nil {
			if !warned {
				logger.Warnf("decode server stats failed for %s: %v", statsURL, err)
				warned = true
			}
			continue
		}
		if closeErr != nil {
			if !warned {
				logger.Warnf("close stats response failed for %s: %v", statsURL, closeErr)
			}
			warned = true
		} else {
			warned = false
		}

		select {
		case sink <- snapshotUpdate{index: index, snapshot: snapshot}:
		case <-ctx.Done():
			return
		default:
		}
	}
}

func reportLoop(ctx context.Context, cfg clientConfig, stats *clientStats, updates <-chan snapshotUpdate) error {
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
			"reject_rate", "latency_avg_ms", "latency_p95_ms", "interval_providers", "providers",
			"healthiest", "limiters", "snapshots",
		}); err != nil {
			return err
		}
	}

	start := time.Now()
	ticker := time.NewTicker(cfg.reportInterval)
	defer ticker.Stop()

	lastSuccess := int64(0)
	lastReport := start
	snapshots := make([]serverSnapshot, len(cfg.statsURLs))

	for {
		select {
		case update := <-updates:
			if update.index >= 0 && update.index < len(snapshots) {
				snapshots[update.index] = update.snapshot
			}
			continue
		default:
		}

		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if update.index >= 0 && update.index < len(snapshots) {
				snapshots[update.index] = update.snapshot
			}
		case now := <-ticker.C:
			row := buildSampleRow(now, start, lastReport, lastSuccess, stats, snapshots)
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

func buildSampleRow(now time.Time, start time.Time, lastReport time.Time, lastSuccess int64, stats *clientStats,
	snapshots []serverSnapshot) sampleRow {
	started := stats.started.Load()
	success := stats.success.Load()
	rejected := stats.rejected.Load()
	failed := stats.failed.Load()
	_, avgMS, p95MS := stats.intervalLatencies.snapshot(true)

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

	hits := stats.hits.snapshot(false)
	intervalHits := stats.intervalHits.snapshot(true)
	return sampleRow{
		TimeUnixSec:       now.Unix(),
		ElapsedSec:        elapsed,
		Started:           started,
		Success:           success,
		Rejected:          rejected,
		Failed:            failed,
		SuccessQPS:        successQPS,
		RejectRate:        rejectRate,
		LatencyAvgMS:      avgMS,
		LatencyP95MS:      p95MS,
		IntervalProviders: formatProviderSummary(intervalHits, success-lastSuccess),
		ProviderSummary:   formatProviderSummary(hits, success),
		Healthiest:        formatHealthiest(snapshots),
		LimiterSummary:    formatLimiterSummary(snapshots),
		SnapshotSummaries: formatSnapshotSummaries(snapshots),
	}
}

func formatProviderSummary(hits map[string]int64, success int64) string {
	if len(hits) == 0 {
		return "none"
	}
	ids := make([]string, 0, len(hits))
	for id := range hits {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		ratio := float64(0)
		if success > 0 {
			ratio = float64(hits[id]) / float64(success) * 100
		}
		parts = append(parts, fmt.Sprintf("%s=%d/%.1f%%", id, hits[id], ratio))
	}
	return strings.Join(parts, ",")
}

func formatLimiterSummary(snapshots []serverSnapshot) string {
	parts := make([]string, 0, len(snapshots))
	for index, snapshot := range snapshots {
		id := snapshot.ID
		if id == "" {
			id = fmt.Sprintf("provider-%d", index+1)
		}
		parts = append(parts, fmt.Sprintf("%s found=%t remaining=%d limitation=%d inflight=%d",
			id, snapshot.LimiterFound, snapshot.LimiterRemaining, snapshot.LimiterLimitation, snapshot.LimiterInflight))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ";")
}

func formatHealthiest(snapshots []serverSnapshot) string {
	var (
		bestID        string
		bestRemaining uint64
		found         bool
	)
	for index, snapshot := range snapshots {
		if !snapshot.LimiterFound {
			continue
		}
		if !found || snapshot.LimiterRemaining > bestRemaining {
			found = true
			bestRemaining = snapshot.LimiterRemaining
			bestID = snapshot.ID
			if bestID == "" {
				bestID = fmt.Sprintf("provider-%d", index+1)
			}
		}
	}
	if !found {
		return "none"
	}
	return fmt.Sprintf("%s remaining=%d", bestID, bestRemaining)
}

func formatSnapshotSummaries(snapshots []serverSnapshot) string {
	parts := make([]string, 0, len(snapshots))
	for index, snapshot := range snapshots {
		id := snapshot.ID
		if id == "" {
			id = fmt.Sprintf("provider-%d", index+1)
		}
		parts = append(parts, fmt.Sprintf("%s work_ms=%d active=%d max_active=%d started=%d completed=%d avg_handler_latency_ms=%.1f",
			id, snapshot.WorkMS, snapshot.Active, snapshot.MaxActive, snapshot.Started, snapshot.Completed, snapshot.AvgHandlerLatencyMS))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ";")
}

func printRow(row sampleRow) {
	fmt.Printf(
		"elapsed=%.0fs started=%d success=%d rejected=%d failed=%d qps=%.1f reject_rate=%.1f%% avg=%.0fms p95=%.0fms interval_providers=[%s] providers=[%s] healthiest=[%s] limiters=[%s] snapshots=[%s]\n",
		row.ElapsedSec,
		row.Started,
		row.Success,
		row.Rejected,
		row.Failed,
		row.SuccessQPS,
		row.RejectRate,
		row.LatencyAvgMS,
		row.LatencyP95MS,
		row.IntervalProviders,
		row.ProviderSummary,
		row.Healthiest,
		row.LimiterSummary,
		row.SnapshotSummaries,
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
		row.IntervalProviders,
		row.ProviderSummary,
		row.Healthiest,
		row.LimiterSummary,
		row.SnapshotSummaries,
	})
}

func parseConfig() clientConfig {
	var cfg clientConfig
	var statsURLs string
	flag.StringVar(&cfg.urls, "urls", "tri://127.0.0.1:20101;tri://127.0.0.1:20102;tri://127.0.0.1:20103", "semicolon-separated Triple provider URLs")
	flag.StringVar(&statsURLs, "stats-urls", "http://127.0.0.1:21101/stats;http://127.0.0.1:21102/stats;http://127.0.0.1:21103/stats", "semicolon-separated server stats URLs")
	flag.IntVar(&cfg.concurrency, "concurrency", 200, "worker concurrency")
	flag.DurationVar(&cfg.duration, "duration", 60*time.Second, "load duration")
	flag.DurationVar(&cfg.timeout, "timeout", 3*time.Second, "per-request timeout")
	flag.StringVar(&cfg.out, "out", "p2c_healthy_result.csv", "CSV output path, empty disables CSV")
	flag.DurationVar(&cfg.reportInterval, "report-interval", time.Second, "report interval")
	flag.Parse()

	cfg.urls = normalizeDirectURLs(cfg.urls)
	cfg.statsURLs = splitSemicolonList(statsURLs)
	return cfg
}

func normalizeDirectURLs(urls string) string {
	parts := splitSemicolonList(urls)
	for i, part := range parts {
		if !strings.Contains(part, "://") {
			parts[i] = "tri://" + part
		}
	}
	return strings.Join(parts, ";")
}

func splitSemicolonList(value string) []string {
	rawParts := strings.Split(value, ";")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func main() {
	cfg := parseConfig()
	cli, err := client.NewClient(
		client.WithClientURL(cfg.urls),
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
	updates := make(chan snapshotUpdate, len(cfg.statsURLs))
	reportErr := make(chan error, 1)
	for index, statsURL := range cfg.statsURLs {
		go pollServerStats(ctx, index, statsURL, cfg.reportInterval, updates)
	}
	go func() {
		reportErr <- reportLoop(ctx, cfg, stats, updates)
	}()

	runLoad(ctx, svc, cfg, stats)

	if err := <-reportErr; err != nil {
		panic(err)
	}

	_, avgMS, p95MS := stats.latencies.snapshot(false)
	started := stats.started.Load()
	success := stats.success.Load()
	rejected := stats.rejected.Load()
	failed := stats.failed.Load()
	var rejectRate float64
	if started > 0 {
		rejectRate = float64(rejected) / float64(started) * 100
	}
	fmt.Printf(
		"final started=%d success=%d rejected=%d failed=%d reject_rate=%.1f%% avg=%.0fms p95=%.0fms providers=[%s] out=%s\n",
		started,
		success,
		rejected,
		failed,
		rejectRate,
		avgMS,
		p95MS,
		formatProviderSummary(stats.hits.snapshot(false), success),
		cfg.out,
	)
}

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
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	protectpb "dubbo.apache.org/dubbo-go/v3/samples/adaptive_service/protect_provider/proto"
	"dubbo.apache.org/dubbo-go/v3/server"
)

type ShrinkServer struct {
	start  time.Time
	stages []workStage
	stats  *serverStats
}

type workStage struct {
	name     string
	delay    time.Duration
	duration time.Duration
}

type serverStats struct {
	active         atomic.Int64
	maxActive      atomic.Int64
	started        atomic.Int64
	completed      atomic.Int64
	totalLatencyNS atomic.Int64
}

type statsResponse struct {
	Phase               string  `json:"phase"`
	WorkMS              int64   `json:"work_ms"`
	Active              int64   `json:"active"`
	MaxActive           int64   `json:"max_active"`
	Started             int64   `json:"started"`
	Completed           int64   `json:"completed"`
	AvgHandlerLatencyMS float64 `json:"avg_handler_latency_ms"`
	LimiterFound        bool    `json:"limiter_found"`
	LimiterInflight     uint64  `json:"limiter_inflight"`
	LimiterRemaining    uint64  `json:"limiter_remaining"`
	LimiterLimitation   uint64  `json:"limiter_limitation"`
}

func (s *ShrinkServer) Work(_ context.Context, req *protectpb.WorkRequest) (*protectpb.WorkResponse, error) {
	start := time.Now()
	seq := s.stats.started.Add(1)
	s.stats.begin()
	defer s.stats.end(start)

	time.Sleep(s.currentStage(start).delay)

	return &protectpb.WorkResponse{
		Message:  "ok:" + req.GetName(),
		Sequence: seq,
	}, nil
}

func (s *ShrinkServer) currentDelay(now time.Time) time.Duration {
	return s.currentStage(now).delay
}

func (s *ShrinkServer) phase(now time.Time) string {
	return s.currentStage(now).name
}

func (s *ShrinkServer) currentStage(now time.Time) workStage {
	elapsed := now.Sub(s.start)
	var passed time.Duration
	for _, stage := range s.stages {
		passed += stage.duration
		if elapsed < passed {
			return stage
		}
	}
	return s.stages[len(s.stages)-1]
}

func (s *serverStats) begin() int64 {
	current := s.active.Add(1)
	for {
		oldMax := s.maxActive.Load()
		if current <= oldMax {
			return current
		}
		if s.maxActive.CompareAndSwap(oldMax, current) {
			return current
		}
	}
}

func (s *serverStats) end(start time.Time) {
	s.active.Add(-1)
	s.completed.Add(1)
	s.totalLatencyNS.Add(time.Since(start).Nanoseconds())
}

func serveStats(addr string, provider *ShrinkServer) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now()
		completed := provider.stats.completed.Load()
		var avgHandlerLatencyMS float64
		if completed > 0 {
			avgHandlerLatencyMS = float64(provider.stats.totalLatencyNS.Load()) / float64(completed) / float64(time.Millisecond)
		}

		limiterSnapshot, limiterFound := adaptivesvc.GetMethodLimiterSnapshot(protectpb.ProtectServiceName, "Work")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(statsResponse{
			Phase:               provider.phase(now),
			WorkMS:              provider.currentDelay(now).Milliseconds(),
			Active:              provider.stats.active.Load(),
			MaxActive:           provider.stats.maxActive.Load(),
			Started:             provider.stats.started.Load(),
			Completed:           completed,
			AvgHandlerLatencyMS: avgHandlerLatencyMS,
			LimiterFound:        limiterFound,
			LimiterInflight:     limiterSnapshot.Inflight,
			LimiterRemaining:    limiterSnapshot.Remaining,
			LimiterLimitation:   limiterSnapshot.Limitation,
		}); err != nil {
			logger.Warnf("write stats response failed: %v", err)
		}
	})

	return http.ListenAndServe(addr, mux)
}

func main() {
	port := flag.Int("port", 20002, "Triple service port")
	statsPort := flag.Int("stats-port", 21002, "HTTP stats port")
	workMS := flag.Int("work-ms", 50, "fast phase handler work duration in milliseconds")
	slowAfter := flag.Duration("slow-after", 20*time.Second, "duration before switching to slow phase")
	slowWorkMS := flag.Int("slow-work-ms", 300, "slow phase handler work duration in milliseconds")
	stagesFlag := flag.String("stages", "", "comma-separated phases as name:work_ms:duration, for example fast:20:30s,medium:100:20s,slow:500:30s")
	flag.Parse()

	stages, err := parseStages(*stagesFlag, *workMS, *slowAfter, *slowWorkMS)
	if err != nil {
		panic(err)
	}

	provider := &ShrinkServer{
		start:  time.Now(),
		stages: stages,
		stats:  &serverStats{},
	}

	statsAddr := fmt.Sprintf("127.0.0.1:%d", *statsPort)
	go func() {
		logger.Infof("stats listening at http://%s/stats", statsAddr)
		if err := serveStats(statsAddr, provider); err != nil {
			logger.Errorf("stats server stopped: %v", err)
		}
	}()

	srv, err := server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithPort(*port),
		),
		server.WithServerAdaptiveService(),
	)
	if err != nil {
		panic(err)
	}

	if err := protectpb.RegisterProtectServiceHandler(srv, provider); err != nil {
		panic(err)
	}

	logger.Infof("rtt shrink provider listening at tri://127.0.0.1:%d", *port)
	if err := srv.Serve(); err != nil {
		panic(err)
	}
}

func parseStages(stagesFlag string, workMS int, slowAfter time.Duration, slowWorkMS int) ([]workStage, error) {
	if strings.TrimSpace(stagesFlag) == "" {
		return []workStage{
			{
				name:     "fast",
				delay:    time.Duration(workMS) * time.Millisecond,
				duration: slowAfter,
			},
			{
				name:     "slow",
				delay:    time.Duration(slowWorkMS) * time.Millisecond,
				duration: time.Duration(1<<63 - 1),
			},
		}, nil
	}

	parts := strings.Split(stagesFlag, ",")
	stages := make([]workStage, 0, len(parts))
	for _, part := range parts {
		fields := strings.Split(strings.TrimSpace(part), ":")
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid stage %q, want name:work_ms:duration", part)
		}
		workMS, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("invalid work_ms in stage %q: %w", part, err)
		}
		duration, err := time.ParseDuration(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid duration in stage %q: %w", part, err)
		}
		if duration <= 0 {
			return nil, fmt.Errorf("stage %q duration must be positive", part)
		}
		stages = append(stages, workStage{
			name:     fields[0],
			delay:    time.Duration(workMS) * time.Millisecond,
			duration: duration,
		})
	}
	if len(stages) == 0 {
		return nil, fmt.Errorf("at least one stage is required")
	}
	return stages, nil
}

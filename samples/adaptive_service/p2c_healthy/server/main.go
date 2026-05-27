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

type HealthyServer struct {
	id        string
	workDelay time.Duration
	stats     *serverStats
}

type serverStats struct {
	active         atomic.Int64
	maxActive      atomic.Int64
	started        atomic.Int64
	completed      atomic.Int64
	totalLatencyNS atomic.Int64
}

type statsResponse struct {
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

func (s *HealthyServer) Work(_ context.Context, req *protectpb.WorkRequest) (*protectpb.WorkResponse, error) {
	start := time.Now()
	seq := s.stats.started.Add(1)
	s.stats.begin()
	defer s.stats.end(start)

	time.Sleep(s.workDelay)

	return &protectpb.WorkResponse{
		Message:  fmt.Sprintf("%s:ok:%s", s.id, req.GetName()),
		Sequence: seq,
	}, nil
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

func serveStats(addr string, provider *HealthyServer) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", func(w http.ResponseWriter, _ *http.Request) {
		completed := provider.stats.completed.Load()
		var avgHandlerLatencyMS float64
		if completed > 0 {
			avgHandlerLatencyMS = float64(provider.stats.totalLatencyNS.Load()) / float64(completed) / float64(time.Millisecond)
		}

		limiterSnapshot, limiterFound := adaptivesvc.GetMethodLimiterSnapshot(protectpb.ProtectServiceName, "Work")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(statsResponse{
			ID:                  provider.id,
			WorkMS:              provider.workDelay.Milliseconds(),
			Active:              provider.stats.active.Load(),
			MaxActive:           provider.stats.maxActive.Load(),
			Started:             provider.stats.started.Load(),
			Completed:           completed,
			AvgHandlerLatencyMS: avgHandlerLatencyMS,
			LimiterFound:        limiterFound,
			LimiterLimitation:   limiterSnapshot.Limitation,
			LimiterRemaining:    limiterSnapshot.Remaining,
			LimiterInflight:     limiterSnapshot.Inflight,
		}); err != nil {
			logger.Warnf("write stats response failed: %v", err)
		}
	})

	return http.ListenAndServe(addr, mux)
}

func main() {
	id := flag.String("id", "provider-1", "provider ID returned in WorkResponse.Message")
	port := flag.Int("port", 20101, "Triple service port")
	statsPort := flag.Int("stats-port", 21101, "HTTP stats port")
	workMS := flag.Int("work-ms", 20, "handler work duration in milliseconds")
	flag.Parse()

	provider := &HealthyServer{
		id:        *id,
		workDelay: time.Duration(*workMS) * time.Millisecond,
		stats:     &serverStats{},
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

	logger.Infof("p2c healthy provider %s listening at tri://127.0.0.1:%d", *id, *port)
	if err := srv.Serve(); err != nil {
		panic(err)
	}
}

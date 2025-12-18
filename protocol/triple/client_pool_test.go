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

package triple

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// TriClientPool Unit Tests
// Test if TriClientPool can warm up correctly
func TestNewTriClientPoolWarmUp(t *testing.T) {
	var created int32
	factory := func() *tri.Client {
		atomic.AddInt32(&created, 1)
		return &tri.Client{}
	}

	const warmUpSize = 5
	const maxSize = 10

	pool, _ := NewTriClientPool(warmUpSize, maxSize, factory)
	defer pool.Close()

	if got := pool.MaxSize(); got != maxSize {
		t.Fatalf("MaxSize() = %d, want %d", got, maxSize)
	}
	if got := len(pool.clients); got != warmUpSize {
		t.Fatalf("len(pool.clients) = %d, want %d", got, warmUpSize)
	}
	if got := atomic.LoadInt32(&created); got != warmUpSize {
		t.Fatalf("created clients = %d, want %d", got, warmUpSize)
	}
}

// Test if TriClientPool can get and put clients correctly
func TestTriClientPoolGetAndPutBasic(t *testing.T) {
	var created int32
	factory := func() *tri.Client {
		atomic.AddInt32(&created, 1)
		return &tri.Client{}
	}

	pool, _ := NewTriClientPool(1, 1, factory)
	defer pool.Close()

	cli, err := pool.Get(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if cli == nil {
		t.Fatalf("Get() returned nil client")
	}
	if got := atomic.LoadInt32(&created); got != 1 {
		t.Fatalf("created clients = %d, want 1", got)
	}

	pool.Put(cli)
	if got := len(pool.clients); got != 1 {
		t.Fatalf("len(pool.clients) after Put = %d, want 1", got)
	}
}

// Test if TriClientPool can get timeout fallback correctly
func TestTriClientPoolGetTimeoutFallback(t *testing.T) {
	var created int32
	factory := func() *tri.Client {
		atomic.AddInt32(&created, 1)
		return &tri.Client{}
	}

	pool, _ := NewTriClientPool(0, 1, factory)
	defer pool.Close()

	cli, err := pool.Get(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("first Get() error = %v, want nil", err)
	}
	if cli == nil {
		t.Fatalf("first Get() returned nil client")
	}

	fallback, _ := pool.Get(5 * time.Millisecond)

	if fallback == nil {
		t.Fatalf("second Get() returned nil fallback client")
	}
	if pool.fallback != fallback {
		t.Fatalf("fallback client mismatch: Get() returned %p, pool.fallback = %p", fallback, pool.fallback)
	}
	if got := atomic.LoadInt32(&created); got != 2 {
		t.Fatalf("created clients = %d, want 2 (one normal + one fallback)", got)
	}
}

// Test if TriClientPool can close correctly
func TestTriClientPoolClose(t *testing.T) {
	pool, _ := NewTriClientPool(1, 1, func() *tri.Client {
		return &tri.Client{}
	})

	if pool.Closed() {
		t.Fatalf("pool should not be closed immediately after creation")
	}

	pool.Close()

	if !pool.Closed() {
		t.Fatalf("pool should be closed after Close()")
	}

	cli, err := pool.Get(1 * time.Millisecond)
	if cli != nil {
		t.Fatalf("Get() after Close() should return nil client, got %v", cli)
	}
	if err != nil && err != ErrTriClientPoolClosed {
		t.Fatalf("Get() after Close() error = %v, want nil or ErrTriClientPoolClosed", err)
	}
}

// Test if TriClientPool can check expand and shrink correctly
func TestCheckExpandAndShrink(t *testing.T) {
	if got := checkExpand(10, 0, 1); got != 2 {
		t.Fatalf("checkExpand(curSize=10,idle=0,timeouts=1) = %d, want 2", got)
	}
	if got := checkExpand(10, 0, 10); got != constant.MaxExpandPerCycle {
		t.Fatalf("checkExpand(..., timeouts=10) = %d, want maxExpandPerCycle(%d)", got, constant.MaxExpandPerCycle)
	}

	if got := checkExpand(10, 1, 0); got != 1 {
		t.Fatalf("checkExpand(curSize=10,idle=1,timeouts=0) = %d, want 1", got)
	}

	var streak int
	for i := 0; i < constant.HighIdleStreakLimit-1; i++ {
		if got := checkShrink(16, 16, &streak); got != 0 {
			t.Fatalf("checkShrink iteration %d returned %d, want 0 before reaching limit", i, got)
		}
	}
	if got := checkShrink(16, 16, &streak); got <= 0 {
		t.Fatalf("checkShrink at highIdleStreakLimit returned %d, want > 0", got)
	}
}

// TriClientPool Benchmark
// Benchmark if TriClientPool with different delays

var (
	testDelays = []time.Duration{
		0,
		100 * time.Microsecond,
		1 * time.Millisecond,
		100 * time.Millisecond,
	}
)

// setupTripleServer creates an HTTP/2 test server with configurable delay for benchmarks
func setupTripleServer(delay time.Duration) (*httptest.Server, string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/test/Echo", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"echo":"ok"}`))
	})

	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	return server, server.URL + "/test/Echo"
}

func benchmarkSingleClient(b *testing.B, baseTransport *http.Transport, testURL string) {
	b.SetParallelism(10)
	transport := &http.Transport{
		TLSClientConfig:     baseTransport.TLSClientConfig,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     50,
	}
	client := tri.NewClient(
		&http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
		testURL,
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			req := tri.NewRequest(map[string]string{"test": "data"})
			resp := tri.NewResponse(map[string]string{})
			_ = client.CallUnary(ctx, req, resp, "method")
			cancel()
		}
	})
}

func benchmarkTriClientPool(b *testing.B, baseTransport *http.Transport, testURL string) {
	b.SetParallelism(10)
	poolSize := 128
	pool, _ := NewTriClientPool(16, poolSize, func() *tri.Client {
		transport := &http.Transport{
			TLSClientConfig:     baseTransport.TLSClientConfig,
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 50,
			MaxConnsPerHost:     50,
		}
		sharedHTTPClient := &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		}
		return tri.NewClient(sharedHTTPClient, testURL)
	})
	defer pool.Close()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			triClient, err := pool.Get(50 * time.Millisecond)
			if err != nil {
				cancel()
				continue
			}
			req := tri.NewRequest(map[string]string{"test": "data"})
			resp := tri.NewResponse(map[string]string{})
			_ = triClient.CallUnary(ctx, req, resp, "method")
			pool.Put(triClient)
			cancel()
		}
	})
}

func BenchmarkThroughputComparisonDelay(b *testing.B) {
	originalLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalLogOutput)

	for _, delay := range testDelays {
		b.Run(fmt.Sprintf("delay_%v", delay), func(b *testing.B) {
			server, testURL := setupTripleServer(delay)
			defer server.Close()
			baseTransport := server.Client().Transport.(*http.Transport)
			b.Run("SingleClient", func(b *testing.B) {
				benchmarkSingleClient(b, baseTransport, testURL)
			})
			b.Run("TriClientPool", func(b *testing.B) {
				benchmarkTriClientPool(b, baseTransport, testURL)
			})
		})
	}
}

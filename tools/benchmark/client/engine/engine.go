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

package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

type BenchmarkFunc func(ctx context.Context) (duration time.Duration, err error)

type Engine struct {
	concurrency      int
	warmupDuration   time.Duration
	testDuration     time.Duration
	requestTimeout   time.Duration
	metricsCollector *MetricsCollector
	stats            *Statistics
	isWarmup         atomic.Bool
	wg               sync.WaitGroup
	stopChan         chan struct{}
	ctx              context.Context
	cancel           context.CancelFunc
	stopOnce         sync.Once
}

func NewEngine(concurrency int, warmupDuration, testDuration, requestTimeout time.Duration) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		concurrency:      concurrency,
		warmupDuration:   warmupDuration,
		testDuration:     testDuration,
		requestTimeout:   requestTimeout,
		metricsCollector: NewMetricsCollector(),
		stats:            NewStatistics(),
		stopChan:         make(chan struct{}),
		ctx:              ctx,
		cancel:           cancel,
	}
	e.isWarmup.Store(true)
	return e
}

func (e *Engine) Run(benchmarkFunc BenchmarkFunc) *Statistics {
	logger.Info("[INFO] Starting warmup...")

	e.startWorkers(benchmarkFunc)

	time.Sleep(e.warmupDuration)

	logger.Info("[INFO] Warmup completed, starting benchmark...")
	e.metricsCollector.Reset()
	e.isWarmup.Store(false)

	timer := time.NewTimer(e.testDuration)
	defer timer.Stop()

	<-timer.C

	e.Stop()

	return e.stats.Compute(e.metricsCollector)
}

func (e *Engine) startWorkers(benchmarkFunc BenchmarkFunc) {
	for i := 0; i < e.concurrency; i++ {
		e.wg.Add(1)
		go e.worker(benchmarkFunc)
	}
}

func (e *Engine) worker(benchmarkFunc BenchmarkFunc) {
	defer e.wg.Done()

	for {
		select {
		case <-e.stopChan:
			return
		default:
			ctx, cancel := context.WithTimeout(e.ctx, e.requestTimeout)
			start := time.Now()
			_, err := benchmarkFunc(ctx)
			duration := time.Since(start)
			cancel()

			if !e.isWarmup.Load() {
				e.metricsCollector.Record(duration, err)
			}
		}
	}
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopChan)
		e.wg.Wait()
		e.cancel()
		logger.Info("[INFO] Benchmark completed")
	})
}

func (e *Engine) GetMetricsCollector() *MetricsCollector {
	return e.metricsCollector
}

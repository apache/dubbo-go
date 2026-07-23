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
	"sync"
	"time"
)

type MetricsCollector struct {
	mu           sync.Mutex
	latencies    []time.Duration
	successCount int64
	failureCount int64
	startTime    time.Time
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies: make([]time.Duration, 0, 100000),
		startTime: time.Now(),
	}
}

func (m *MetricsCollector) Record(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencies = append(m.latencies, latency)
	if err == nil {
		m.successCount++
	} else {
		m.failureCount++
	}
}

func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencies = make([]time.Duration, 0, 100000)
	m.successCount = 0
	m.failureCount = 0
	m.startTime = time.Now()
}

func (m *MetricsCollector) GetLatencies() []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]time.Duration, len(m.latencies))
	copy(result, m.latencies)
	return result
}

func (m *MetricsCollector) GetSuccessCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.successCount
}

func (m *MetricsCollector) GetFailureCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failureCount
}

func (m *MetricsCollector) GetTotalCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.successCount + m.failureCount
}

func (m *MetricsCollector) GetStartTime() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startTime
}

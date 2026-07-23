/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package engine

import (
	"fmt"
	"sort"
	"time"
)

const separator = "========================================"

type Statistics struct {
	QPS         float64
	P50         time.Duration
	P90         time.Duration
	P95         time.Duration
	P99         time.Duration
	Min         time.Duration
	Max         time.Duration
	Avg         time.Duration
	Success     int64
	Failure     int64
	Total       int64
	SuccessRate float64
}

func NewStatistics() *Statistics {
	return &Statistics{}
}

func (s *Statistics) Compute(m *MetricsCollector) *Statistics {
	latencies := m.GetLatencies()
	if len(latencies) == 0 {
		return s
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	s.Total = m.GetTotalCount()
	s.Success = m.GetSuccessCount()
	s.Failure = m.GetFailureCount()
	s.SuccessRate = float64(s.Success) / float64(s.Total) * 100

	duration := time.Since(m.GetStartTime())
	s.QPS = float64(s.Success) / duration.Seconds()

	s.Min = latencies[0]
	s.Max = latencies[len(latencies)-1]

	sum := time.Duration(0)
	for _, l := range latencies {
		sum += l
	}
	s.Avg = sum / time.Duration(len(latencies))

	s.P50 = s.percentile(latencies, 0.50)
	s.P90 = s.percentile(latencies, 0.90)
	s.P95 = s.percentile(latencies, 0.95)
	s.P99 = s.percentile(latencies, 0.99)

	return s
}

func (s *Statistics) percentile(latencies []time.Duration, p float64) time.Duration {
	index := int(float64(len(latencies)) * p)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}
	return latencies[index]
}

func (s *Statistics) String() string {
	return fmt.Sprintf("\n%s\n          性能测试报告\n%s\nQPS:             %.2f\n成功率:          %.2f%%\n总请求数:        %d\n成功数:          %d\n失败数:          %d\n----------------------------------------\n延迟统计(ms):\n  Min:           %.2f\n  Avg:           %.2f\n  P50:           %.2f\n  P90:           %.2f\n  P95:           %.2f\n  P99:           %.2f\n  Max:           %.2f\n%s",
		separator,
		separator,
		s.QPS,
		s.SuccessRate,
		s.Total,
		s.Success,
		s.Failure,
		float64(s.Min)/float64(time.Millisecond),
		float64(s.Avg)/float64(time.Millisecond),
		float64(s.P50)/float64(time.Millisecond),
		float64(s.P90)/float64(time.Millisecond),
		float64(s.P95)/float64(time.Millisecond),
		float64(s.P99)/float64(time.Millisecond),
		float64(s.Max)/float64(time.Millisecond),
	)
}

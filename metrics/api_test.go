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

package metrics

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// mockMetricRegistry is a simple mock implementation of MetricRegistry for testing
type mockMetricRegistry struct {
	counters   map[string]*mockCounterMetric
	gauges     map[string]*mockGaugeMetric
	histograms map[string]*mockObservableMetric
	summaries  map[string]*mockObservableMetric
	rts        map[string]*mockObservableMetric
}

func newMockMetricRegistry() *mockMetricRegistry {
	return &mockMetricRegistry{
		counters:   make(map[string]*mockCounterMetric),
		gauges:     make(map[string]*mockGaugeMetric),
		histograms: make(map[string]*mockObservableMetric),
		summaries:  make(map[string]*mockObservableMetric),
		rts:        make(map[string]*mockObservableMetric),
	}
}

func (m *mockMetricRegistry) Counter(id *MetricId) CounterMetric {
	if c, ok := m.counters[id.Name]; ok {
		return c
	}
	c := &mockCounterMetric{}
	m.counters[id.Name] = c
	return c
}

func (m *mockMetricRegistry) Gauge(id *MetricId) GaugeMetric {
	if g, ok := m.gauges[id.Name]; ok {
		return g
	}
	g := &mockGaugeMetric{}
	m.gauges[id.Name] = g
	return g
}

func (m *mockMetricRegistry) Histogram(id *MetricId) ObservableMetric {
	if h, ok := m.histograms[id.Name]; ok {
		return h
	}
	h := &mockObservableMetric{}
	m.histograms[id.Name] = h
	return h
}

func (m *mockMetricRegistry) Summary(id *MetricId) ObservableMetric {
	if s, ok := m.summaries[id.Name]; ok {
		return s
	}
	s := &mockObservableMetric{}
	m.summaries[id.Name] = s
	return s
}

func (m *mockMetricRegistry) Rt(id *MetricId, opts *RtOpts) ObservableMetric {
	if rt, ok := m.rts[id.Name]; ok {
		return rt
	}
	rt := &mockObservableMetric{}
	m.rts[id.Name] = rt
	return rt
}

func (m *mockMetricRegistry) Export() {}

type mockCounterMetric struct {
	value float64
}

func (m *mockCounterMetric) Inc()          { m.value++ }
func (m *mockCounterMetric) Add(v float64) { m.value += v }

type mockGaugeMetric struct {
	value float64
}

func (m *mockGaugeMetric) Set(v float64) { m.value = v }
func (m *mockGaugeMetric) Inc()          { m.value++ }
func (m *mockGaugeMetric) Dec()          { m.value-- }
func (m *mockGaugeMetric) Add(v float64) { m.value += v }
func (m *mockGaugeMetric) Sub(v float64) { m.value -= v }

type mockObservableMetric struct {
	value float64
}

func (m *mockObservableMetric) Observe(v float64) { m.value = v }

func TestMetricIdTagKeys(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		want int
	}{
		{
			name: "empty tags",
			tags: map[string]string{},
			want: 0,
		},
		{
			name: "single tag",
			tags: map[string]string{"app": "dubbo"},
			want: 1,
		},
		{
			name: "multiple tags",
			tags: map[string]string{
				"app":     "dubbo",
				"version": "1.0.0",
				"ip":      "127.0.0.1",
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricId{Name: "test_metric", Tags: tt.tags}
			got := m.TagKeys()
			assert.Equal(t, tt.want, len(got))
			for k := range tt.tags {
				assert.Contains(t, got, k)
			}
		})
	}
}

func TestBaseCollectorStateCount(t *testing.T) {
	registry := newMockMetricRegistry()
	collector := &BaseCollector{R: registry}
	level := GetApplicationLevel()

	totalKey := NewMetricKey("total", "Total requests")
	succKey := NewMetricKey("succ", "Success requests")
	failKey := NewMetricKey("fail", "Failed requests")

	t.Run("success", func(t *testing.T) {
		collector.StateCount(totalKey, succKey, failKey, level, true)

		total := registry.counters["total"]
		succ := registry.counters["succ"]
		fail := registry.counters["fail"]

		assert.Equal(t, float64(1), total.value)
		assert.Equal(t, float64(1), succ.value)
		assert.Equal(t, float64(0), fail.value)
	})

	t.Run("failure", func(t *testing.T) {
		collector.StateCount(totalKey, succKey, failKey, level, false)

		total := registry.counters["total"]
		fail := registry.counters["fail"]

		assert.Equal(t, float64(2), total.value)
		assert.Equal(t, float64(1), fail.value)
	})
}

func TestDefaultCounterVec(t *testing.T) {
	registry := newMockMetricRegistry()
	key := NewMetricKey("test_counter", "Test counter")
	counterVec := NewCounterVec(registry, key)
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	t.Run("Inc", func(t *testing.T) {
		counterVec.Inc(labels)
		metricId := NewMetricIdByLabels(key, labels)
		counter := registry.Counter(metricId).(*mockCounterMetric)
		assert.Equal(t, float64(1), counter.value)
	})

	t.Run("Add", func(t *testing.T) {
		counterVec.Add(labels, 5.0)
		metricId := NewMetricIdByLabels(key, labels)
		counter := registry.Counter(metricId).(*mockCounterMetric)
		assert.Equal(t, float64(6), counter.value)
	})
}

func TestDefaultGaugeVec(t *testing.T) {
	registry := newMockMetricRegistry()
	key := NewMetricKey("test_gauge", "Test gauge")
	gaugeVec := NewGaugeVec(registry, key)
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	t.Run("Set", func(t *testing.T) {
		gaugeVec.Set(labels, 100.0)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.Equal(t, float64(100), gauge.value)
	})

	t.Run("Inc", func(t *testing.T) {
		gaugeVec.Inc(labels)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.Equal(t, float64(101), gauge.value)
	})

	t.Run("Dec", func(t *testing.T) {
		gaugeVec.Dec(labels)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.Equal(t, float64(100), gauge.value)
	})

	t.Run("Add", func(t *testing.T) {
		gaugeVec.Add(labels, 50.0)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.Equal(t, float64(150), gauge.value)
	})

	t.Run("Sub", func(t *testing.T) {
		gaugeVec.Sub(labels, 30.0)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.Equal(t, float64(120), gauge.value)
	})
}

func TestDefaultRtVec(t *testing.T) {
	registry := newMockMetricRegistry()
	key := NewMetricKey("test_rt", "Test response time")
	rtOpts := &RtOpts{Aggregate: false}
	rtVec := NewRtVec(registry, key, rtOpts)
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	rtVec.Record(labels, 100.0)
	rt := registry.Rt(NewMetricIdByLabels(key, labels), rtOpts)
	assert.NotNil(t, rt)
}

func TestDefaultQpsMetricVec(t *testing.T) {
	registry := newMockMetricRegistry()
	key := NewMetricKey("test_qps", "Test QPS")
	qpsVec := NewQpsMetricVec(registry, key)
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	for i := 0; i < 5; i++ {
		qpsVec.Record(labels)
	}

	gauge := registry.Gauge(NewMetricIdByLabels(key, labels))
	assert.NotNil(t, gauge)
}

func TestDefaultAggregateCounterVec(t *testing.T) {
	registry := newMockMetricRegistry()
	key := NewMetricKey("test_agg_counter", "Test aggregate counter")
	aggCounterVec := NewAggregateCounterVec(registry, key)
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	for i := 0; i < 3; i++ {
		aggCounterVec.Inc(labels)
	}

	gauge := registry.Gauge(NewMetricIdByLabels(key, labels))
	assert.NotNil(t, gauge)
}

func TestDefaultQuantileMetricVec(t *testing.T) {
	registry := newMockMetricRegistry()
	keys := []*MetricKey{
		NewMetricKey("test_quantile_p50", "P50 quantile"),
		NewMetricKey("test_quantile_p90", "P90 quantile"),
	}
	quantileVec := NewQuantileMetricVec(registry, keys, []float64{0.5, 0.9})
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	for i := 0; i < 10; i++ {
		quantileVec.Record(labels, float64(i*10))
	}

	for _, key := range keys {
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels))
		assert.NotNil(t, gauge)
	}
}

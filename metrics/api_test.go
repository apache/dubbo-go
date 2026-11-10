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
	"math"
	"testing"
	"testing/synctest"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGaugeFuncRegistry struct {
	*mockMetricRegistry
	callbacks     map[string]func() float64
	registrations map[string]int
}

func (r *mockGaugeFuncRegistry) GaugeFunc(id *MetricId, value func() float64) {
	r.callbacks[id.Name] = value
	r.registrations[id.Name]++
}

func TestWindowMetricGaugeFuncRegistry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := &mockGaugeFuncRegistry{
			mockMetricRegistry: newMockMetricRegistry(),
			callbacks:          make(map[string]func() float64),
			registrations:      make(map[string]int),
		}
		labels := map[string]string{"app": "dubbo"}
		qps := NewQpsMetricVec(registry, NewMetricKey("qps", "QPS"))
		count := NewAggregateCounterVec(registry, NewMetricKey("count", "Window count"))
		quantiles := NewQuantileMetricVec(registry,
			[]*MetricKey{NewMetricKey("p50", "P50"), NewMetricKey("p99", "P99")},
			[]float64{0.5, 0.99})
		record := func(duration float64) {
			qps.Record(labels)
			count.Inc(labels)
			quantiles.Record(labels, duration)
		}
		require.Empty(t, registry.callbacks, "No callbacks before the first record")
		for i := 1; i <= 100; i++ {
			record(float64(i))
		}
		for _, name := range []string{"qps", "count", "p50", "p99"} {
			require.NotNil(t, registry.callbacks[name])
			require.Equal(t, 1, registry.registrations[name], "Register each callback only once")
		}
		require.Empty(t, registry.gauges, "Dynamic registries must not receive static Gauge snapshots")
		for range 2 {
			require.InDelta(t, 100.0/12, registry.callbacks["qps"](), 0)
			require.InDelta(t, 100.0, registry.callbacks["count"](), 0)
			require.InDelta(t, 50.5, registry.callbacks["p50"](), 0.01)
			require.InDelta(t, 99.5, registry.callbacks["p99"](), 0.01)
		}
		time.Sleep(121 * time.Second)
		require.InDelta(t, 0.0, registry.callbacks["qps"](), 0)
		require.InDelta(t, 0.0, registry.callbacks["count"](), 0)
		require.True(t, math.IsNaN(registry.callbacks["p50"]()))
		require.True(t, math.IsNaN(registry.callbacks["p99"]()))
		record(1000)
		require.InDelta(t, 1.0/12, registry.callbacks["qps"](), 0)
		require.InDelta(t, 1.0, registry.callbacks["count"](), 0)
		require.InDelta(t, 1000.0, registry.callbacks["p50"](), 0)
		require.InDelta(t, 1000.0, registry.callbacks["p99"](), 0)
		for _, name := range []string{"qps", "count", "p50", "p99"} {
			require.Equal(t, 1, registry.registrations[name], "Reuse callbacks when recording resumes")
		}
		require.Empty(t, registry.gauges)
	})
}

func TestLegacyWindowMetricRegistry(t *testing.T) {
	// Third-party registries implementing only MetricRegistry remain usable.
	// Without GaugeFuncRegistry they retain their previous record-time behavior.
	synctest.Test(t, func(t *testing.T) {
		registry := newMockMetricRegistry()
		labels := map[string]string{"app": "dubbo"}
		qps := NewQpsMetricVec(registry, NewMetricKey("qps", "QPS"))
		count := NewAggregateCounterVec(registry, NewMetricKey("count", "Window count"))
		quantiles := NewQuantileMetricVec(registry, []*MetricKey{NewMetricKey("p99", "P99")}, []float64{0.99})
		for range 6 {
			qps.Record(labels)
			count.Inc(labels)
			quantiles.Record(labels, 100)
		}
		assert.InDelta(t, 0.5, registry.gauges["qps"].value, 0)
		assert.InDelta(t, 6.0, registry.gauges["count"].value, 0)
		assert.InDelta(t, 100.0, registry.gauges["p99"].value, 0)
		time.Sleep(121 * time.Second)
		qps.Record(labels)
		count.Inc(labels)
		quantiles.Record(labels, 1000)
		assert.InDelta(t, 1.0/12, registry.gauges["qps"].value, 0)
		assert.InDelta(t, 1.0, registry.gauges["count"].value, 0)
		assert.InDelta(t, 1000.0, registry.gauges["p99"].value, 0)
	})
}

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
			assert.Len(t, got, tt.want)
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

		assert.NotNil(t, total)
		assert.NotNil(t, succ)
		assert.InDelta(t, 1.0, total.value, 0.01)
		assert.InDelta(t, 1.0, succ.value, 0.01)
		if fail != nil {
			assert.InDelta(t, 0.0, fail.value, 0.01)
		}
	})

	t.Run("failure", func(t *testing.T) {
		collector.StateCount(totalKey, succKey, failKey, level, false)

		total := registry.counters["total"]
		fail := registry.counters["fail"]

		assert.NotNil(t, total)
		assert.NotNil(t, fail)
		assert.InDelta(t, 2.0, total.value, 0.01)
		assert.InDelta(t, 1.0, fail.value, 0.01)
	})
}

func TestDefaultCounterVec(t *testing.T) {
	key := NewMetricKey("test_counter", "Test counter")
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	t.Run("Inc", func(t *testing.T) {
		registry := newMockMetricRegistry()
		counterVec := NewCounterVec(registry, key)
		counterVec.Inc(labels)
		metricId := NewMetricIdByLabels(key, labels)
		counter := registry.Counter(metricId).(*mockCounterMetric)
		assert.InDelta(t, 1.0, counter.value, 0.01)
	})

	t.Run("Add", func(t *testing.T) {
		registry := newMockMetricRegistry()
		counterVec := NewCounterVec(registry, key)
		counterVec.Add(labels, 5.0)
		metricId := NewMetricIdByLabels(key, labels)
		counter := registry.Counter(metricId).(*mockCounterMetric)
		assert.InDelta(t, 5.0, counter.value, 0.01)
	})
}

func TestDefaultGaugeVec(t *testing.T) {
	key := NewMetricKey("test_gauge", "Test gauge")
	labels := map[string]string{"app": "dubbo", "version": "1.0.0"}

	t.Run("Set", func(t *testing.T) {
		registry := newMockMetricRegistry()
		gaugeVec := NewGaugeVec(registry, key)
		gaugeVec.Set(labels, 100.0)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.InDelta(t, 100.0, gauge.value, 0.01)
	})

	t.Run("Inc", func(t *testing.T) {
		registry := newMockMetricRegistry()
		gaugeVec := NewGaugeVec(registry, key)
		gaugeVec.Set(labels, 100.0)
		gaugeVec.Inc(labels)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.InDelta(t, 101.0, gauge.value, 0.01)
	})

	t.Run("Dec", func(t *testing.T) {
		registry := newMockMetricRegistry()
		gaugeVec := NewGaugeVec(registry, key)
		gaugeVec.Set(labels, 100.0)
		gaugeVec.Dec(labels)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.InDelta(t, 99.0, gauge.value, 0.01)
	})

	t.Run("Add", func(t *testing.T) {
		registry := newMockMetricRegistry()
		gaugeVec := NewGaugeVec(registry, key)
		gaugeVec.Set(labels, 100.0)
		gaugeVec.Add(labels, 50.0)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.InDelta(t, 150.0, gauge.value, 0.01)
	})

	t.Run("Sub", func(t *testing.T) {
		registry := newMockMetricRegistry()
		gaugeVec := NewGaugeVec(registry, key)
		gaugeVec.Set(labels, 100.0)
		gaugeVec.Sub(labels, 30.0)
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels)).(*mockGaugeMetric)
		assert.InDelta(t, 70.0, gauge.value, 0.01)
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

	for range 5 {
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

	for range 3 {
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

	for i := range 10 {
		quantileVec.Record(labels, float64(i*10))
	}

	for _, key := range keys {
		gauge := registry.Gauge(NewMetricIdByLabels(key, labels))
		assert.NotNil(t, gauge)
	}
}

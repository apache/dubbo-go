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

package metadata

import (
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func TestMetadataMetricEventType(t *testing.T) {
	event := &MetadataMetricEvent{
		Name: MetadataPush,
		Succ: true,
	}

	assert.Equal(t, constant.MetricsMetadata, event.Type())
}

func TestMetadataMetricEventCostMs(t *testing.T) {
	start := time.Now()
	end := start.Add(10 * time.Millisecond)

	event := &MetadataMetricEvent{
		Name:  MetadataPush,
		Start: start,
		End:   end,
	}

	cost := event.CostMs()
	assert.InDelta(t, 10.0, cost, 0.01)
}

func TestNewMetadataMetricTimeEvent(t *testing.T) {
	event := NewMetadataMetricTimeEvent(MetadataPush)

	assert.NotNil(t, event)
	assert.Equal(t, MetadataPush, event.Name)
	assert.NotNil(t, event.Start)
	assert.NotNil(t, event.Attachment)
	assert.Empty(t, event.Attachment)
}

func TestMetadataMetricCollectorHandleMapping(t *testing.T) {
	tests := []struct {
		name      string
		eventName MetricName
		handler   func(*MetadataMetricCollector, *MetadataMetricEvent)
		prefix    string
	}{
		{
			name:      "register",
			eventName: MetadataMappingRegister,
			handler:   (*MetadataMetricCollector).handleMetadataMappingRegister,
			prefix:    "dubbo_metadata_mapping_register",
		},
		{
			name:      "get",
			eventName: MetadataMappingGet,
			handler:   (*MetadataMetricCollector).handleMetadataMappingGet,
			prefix:    "dubbo_metadata_mapping_get",
		},
		{
			name:      "listen",
			eventName: MetadataMappingListen,
			handler:   (*MetadataMetricCollector).handleMetadataMappingListen,
			prefix:    "dubbo_metadata_mapping_listen",
		},
		{
			name:      "remove",
			eventName: MetadataMappingRemove,
			handler:   (*MetadataMetricCollector).handleMetadataMappingRemove,
			prefix:    "dubbo_metadata_mapping_remove",
		},
	}

	for _, tt := range tests {
		for _, succ := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/succ=%v", tt.name, succ), func(t *testing.T) {
				registry := newMockMetricRegistry()
				collector := &MetadataMetricCollector{BaseCollector: metrics.BaseCollector{R: registry}}
				event := NewMetadataMetricTimeEvent(tt.eventName)
				event.End = event.Start.Add(10 * time.Millisecond)
				event.Succ = succ
				event.Attachment[constant.InterfaceKey] = "interfaceName"
				event.Attachment[constant.GroupKey] = "group"
				event.Attachment[constant.ApplicationKey] = "application"

				tt.handler(collector, event)

				assert.InDelta(t, 1.0, registry.counters[tt.prefix+"_num_total"], 0.000001)
				if succ {
					assert.InDelta(t, 1.0, registry.counters[tt.prefix+"_num_succeed_total"], 0.000001)
					assert.NotContains(t, registry.counters, tt.prefix+"_num_failed_total")
				} else {
					assert.InDelta(t, 1.0, registry.counters[tt.prefix+"_num_failed_total"], 0.000001)
					assert.NotContains(t, registry.counters, tt.prefix+"_num_succeed_total")
				}
				assert.Equal(t, []float64{10.0}, registry.rts[tt.prefix+"_rt_milliseconds"])

				id := registry.ids[tt.prefix+"_num_total"]
				assert.Equal(t, "interfaceName", id.Tags[constant.TagInterface])
				assert.Equal(t, "group", id.Tags[constant.TagGroup])
				assert.Equal(t, "application", id.Tags[constant.TagApplicationName])
			})
		}
	}
}

type mockMetricRegistry struct {
	counters map[string]float64
	rts      map[string][]float64
	ids      map[string]*metrics.MetricId
}

func newMockMetricRegistry() *mockMetricRegistry {
	return &mockMetricRegistry{
		counters: make(map[string]float64),
		rts:      make(map[string][]float64),
		ids:      make(map[string]*metrics.MetricId),
	}
}

func (m *mockMetricRegistry) Counter(id *metrics.MetricId) metrics.CounterMetric {
	m.ids[id.Name] = id
	return &mockCounterMetric{m: m, name: id.Name}
}

func (m *mockMetricRegistry) Rt(id *metrics.MetricId, _ *metrics.RtOpts) metrics.ObservableMetric {
	m.ids[id.Name] = id
	return &mockRtMetric{m: m, name: id.Name}
}

func (m *mockMetricRegistry) Gauge(id *metrics.MetricId) metrics.GaugeMetric {
	return nil
}

func (m *mockMetricRegistry) Histogram(id *metrics.MetricId) metrics.ObservableMetric {
	return nil
}

func (m *mockMetricRegistry) Summary(id *metrics.MetricId) metrics.ObservableMetric {
	return nil
}

func (m *mockMetricRegistry) Export() {}

type mockCounterMetric struct {
	m    *mockMetricRegistry
	name string
}

func (c *mockCounterMetric) Inc()          { c.m.counters[c.name]++ }
func (c *mockCounterMetric) Add(v float64) { c.m.counters[c.name] += v }

type mockRtMetric struct {
	m    *mockMetricRegistry
	name string
}

func (r *mockRtMetric) Observe(v float64) { r.m.rts[r.name] = append(r.m.rts[r.name], v) }

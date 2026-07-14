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

func TestMetadataMetricCollectorHandleMappingRegister(t *testing.T) {
	registry := newMetadataTestMetricRegistry()
	collector := &MetadataMetricCollector{BaseCollector: metrics.BaseCollector{R: registry}}
	event := NewMetadataMetricTimeEvent(MetadataMappingRegister)
	event.End = event.Start.Add(10 * time.Millisecond)
	event.Succ = true
	event.Attachment[MetadataMappingOperationKey] = "register"
	event.Attachment[constant.InterfaceKey] = "interfaceName"
	event.Attachment[constant.GroupKey] = "group"
	event.Attachment[constant.ApplicationKey] = "application"

	collector.handleMetadataMappingRegister(event)

	assert.Equal(t, 1.0, registry.counterValue("dubbo_metadata_mapping_register_num_total"))
	assert.Equal(t, 1.0, registry.counterValue("dubbo_metadata_mapping_register_num_succeed_total"))
	assert.Equal(t, 0.0, registry.counterValue("dubbo_metadata_mapping_register_num_failed_total"))
	assert.Equal(t, []float64{10.0}, registry.rtValues("dubbo_metadata_mapping_register_rt_milliseconds"))
	id := registry.metricId("dubbo_metadata_mapping_register_num_total")
	assert.Equal(t, "register", id.Tags[MetadataMappingOperationKey])
	assert.Equal(t, "interfaceName", id.Tags[constant.TagInterface])
	assert.Equal(t, "group", id.Tags[constant.GroupKey])
	assert.Equal(t, "application", id.Tags[constant.ApplicationKey])
}

func TestMetadataMetricCollectorHandleMappingGet(t *testing.T) {
	registry := newMetadataTestMetricRegistry()
	collector := &MetadataMetricCollector{BaseCollector: metrics.BaseCollector{R: registry}}
	event := NewMetadataMetricTimeEvent(MetadataMappingGet)
	event.End = event.Start.Add(10 * time.Millisecond)
	event.Succ = true
	event.Attachment[MetadataMappingOperationKey] = "get"
	event.Attachment[constant.InterfaceKey] = "interfaceName"
	event.Attachment[constant.GroupKey] = "group"
	event.Attachment[MetadataMappingListenerRequestedKey] = "true"

	collector.handleMetadataMappingGet(event)

	assert.Equal(t, 1.0, registry.counterValue("dubbo_metadata_mapping_get_num_total"))
	assert.Equal(t, 1.0, registry.counterValue("dubbo_metadata_mapping_get_num_succeed_total"))
	assert.Equal(t, []float64{10.0}, registry.rtValues("dubbo_metadata_mapping_get_rt_milliseconds"))
	id := registry.metricId("dubbo_metadata_mapping_get_num_total")
	assert.Equal(t, "get", id.Tags[MetadataMappingOperationKey])
	assert.Equal(t, "interfaceName", id.Tags[constant.TagInterface])
	assert.Equal(t, "group", id.Tags[constant.GroupKey])
	assert.Equal(t, "true", id.Tags[MetadataMappingListenerRequestedKey])
}

func TestMetadataMetricCollectorHandleMappingRemove(t *testing.T) {
	registry := newMetadataTestMetricRegistry()
	collector := &MetadataMetricCollector{BaseCollector: metrics.BaseCollector{R: registry}}
	event := NewMetadataMetricTimeEvent(MetadataMappingRemove)
	event.End = event.Start.Add(10 * time.Millisecond)
	event.Succ = false
	event.Attachment[MetadataMappingOperationKey] = "remove"
	event.Attachment[constant.InterfaceKey] = "interfaceName"
	event.Attachment[constant.GroupKey] = "group"

	collector.handleMetadataMappingRemove(event)

	assert.Equal(t, 1.0, registry.counterValue("dubbo_metadata_mapping_remove_num_total"))
	assert.Equal(t, 1.0, registry.counterValue("dubbo_metadata_mapping_remove_num_failed_total"))
	assert.Equal(t, []float64{10.0}, registry.rtValues("dubbo_metadata_mapping_remove_rt_milliseconds"))
	id := registry.metricId("dubbo_metadata_mapping_remove_num_total")
	assert.Equal(t, "remove", id.Tags[MetadataMappingOperationKey])
	assert.Equal(t, "interfaceName", id.Tags[constant.TagInterface])
	assert.Equal(t, "group", id.Tags[constant.GroupKey])
}

type metadataTestMetricRegistry struct {
	counters map[string]*metadataTestCounterMetric
	rts      map[string]*metadataTestObservableMetric
	ids      map[string]*metrics.MetricId
}

func newMetadataTestMetricRegistry() *metadataTestMetricRegistry {
	return &metadataTestMetricRegistry{
		counters: make(map[string]*metadataTestCounterMetric),
		rts:      make(map[string]*metadataTestObservableMetric),
		ids:      make(map[string]*metrics.MetricId),
	}
}

func (m *metadataTestMetricRegistry) Counter(id *metrics.MetricId) metrics.CounterMetric {
	m.ids[id.Name] = id
	if c, ok := m.counters[id.Name]; ok {
		return c
	}
	c := &metadataTestCounterMetric{}
	m.counters[id.Name] = c
	return c
}

func (m *metadataTestMetricRegistry) Gauge(*metrics.MetricId) metrics.GaugeMetric {
	return &metadataTestGaugeMetric{}
}

func (m *metadataTestMetricRegistry) Histogram(*metrics.MetricId) metrics.ObservableMetric {
	return &metadataTestObservableMetric{}
}

func (m *metadataTestMetricRegistry) Summary(*metrics.MetricId) metrics.ObservableMetric {
	return &metadataTestObservableMetric{}
}

func (m *metadataTestMetricRegistry) Rt(id *metrics.MetricId, _ *metrics.RtOpts) metrics.ObservableMetric {
	m.ids[id.Name] = id
	if rt, ok := m.rts[id.Name]; ok {
		return rt
	}
	rt := &metadataTestObservableMetric{}
	m.rts[id.Name] = rt
	return rt
}

func (m *metadataTestMetricRegistry) Export() {}

func (m *metadataTestMetricRegistry) counterValue(name string) float64 {
	if counter, ok := m.counters[name]; ok {
		return counter.value
	}
	return 0
}

func (m *metadataTestMetricRegistry) rtValues(name string) []float64 {
	if rt, ok := m.rts[name]; ok {
		return rt.values
	}
	return nil
}

func (m *metadataTestMetricRegistry) metricId(name string) *metrics.MetricId {
	return m.ids[name]
}

type metadataTestCounterMetric struct {
	value float64
}

func (m *metadataTestCounterMetric) Inc() {
	m.value++
}

func (m *metadataTestCounterMetric) Add(v float64) {
	m.value += v
}

type metadataTestObservableMetric struct {
	values []float64
}

func (m *metadataTestObservableMetric) Observe(v float64) {
	m.values = append(m.values, v)
}

type metadataTestGaugeMetric struct{}

func (*metadataTestGaugeMetric) Set(float64) {}
func (*metadataTestGaugeMetric) Inc()        {}
func (*metadataTestGaugeMetric) Dec()        {}
func (*metadataTestGaugeMetric) Add(float64) {}
func (*metadataTestGaugeMetric) Sub(float64) {}

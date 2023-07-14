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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

const eventType = constant.MetricsMetadata

var ch = make(chan metrics.MetricsEvent, 10)

func init() {
	metrics.AddCollector("metadata", func(mr metrics.MetricRegistry, rc *metrics.ReporterConfig) {
		l := &MetadataMetricCollector{r: mr}
		l.start()
	})
}

type MetadataMetricCollector struct {
	r metrics.MetricRegistry
}

func (c *MetadataMetricCollector) start() {
	metrics.Subscribe(eventType, ch)
	go func() {
		for e := range ch {
			if event, ok := e.(*MetadataMetricEvent); ok {
				switch event.Name {
				case StoreProvider:
					c.handleStoreProvider(event)
				case MetadataPush:
					c.handleMetadataPush(event)
				case MetadataSub:
					c.handleMetadataSub(event)
				default:
				}
			}
		}
	}()
}

func (c *MetadataMetricCollector) handleMetadataPush(event *MetadataMetricEvent) {
	m := newStatesMetricFunc(metadataPushNum, metadataPushNumSucceed, metadataPushNumFailed, metrics.GetApplicationLevel(), c.r)
	m.Inc(event.Succ)
	// TODO add RT metric dubbo_push_rt_milliseconds
}

func (c *MetadataMetricCollector) handleMetadataSub(event *MetadataMetricEvent) {
	m := newStatesMetricFunc(metadataSubNum, metadataSubNumSucceed, metadataSubNumFailed, metrics.GetApplicationLevel(), c.r)
	m.Inc(event.Succ)
	// TODO add RT metric dubbo_subscribe_rt_milliseconds
}

func (c *MetadataMetricCollector) handleStoreProvider(event *MetadataMetricEvent) {
	level := metrics.NewServiceMetric(event.Attachment[constant.InterfaceKey])
	m := newStatesMetricFunc(metadataStoreProvider, metadataStoreProviderSucceed, metadataStoreProviderFailed, level, c.r)
	m.Inc(event.Succ)
	// TODO add RT metric dubbo_store_provider_interface_rt_milliseconds
}

func newStatesMetricFunc(total *metrics.MetricKey, succ *metrics.MetricKey, fail *metrics.MetricKey, level metrics.MetricLevel, reg metrics.MetricRegistry) metrics.StatesMetrics {
	return metrics.NewStatesMetrics(
		func() *metrics.MetricId { return metrics.NewMetricId(total, level) },
		func() *metrics.MetricId { return metrics.NewMetricId(succ, level) },
		func() *metrics.MetricId { return metrics.NewMetricId(fail, level) },
		reg,
	)
}

type MetadataMetricEvent struct {
	Name       MetricName
	Succ       bool
	Start      time.Time
	End        time.Time
	Attachment map[string]string
}

func (*MetadataMetricEvent) Type() string {
	return eventType
}

func (e *MetadataMetricEvent) CostMs() float64 {
	return float64(e.End.Sub(e.Start)) / float64(time.Millisecond)
}

func NewMetadataMetricTimeEvent(n MetricName) *MetadataMetricEvent {
	return &MetadataMetricEvent{Name: n, Start: time.Now()}
}

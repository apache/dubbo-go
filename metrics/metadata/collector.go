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
				case SubscribeServiceRt:
					c.handleSubscribeService(event)
				default:
				}
			}
		}
	}()
}

func (c *MetadataMetricCollector) handleMetadataPush(event *MetadataMetricEvent) {
	m := metrics.ComputeIfAbsentCache(dubboMetadataPush, func() interface{} {
		return newStatesMetricFunc(metadataPushNum, metadataPushNumSucceed, metadataPushNumFailed, metrics.GetApplicationLevel(), c.r)
	}).(metrics.StatesMetrics)
	m.Inc(event.Succ)
	metric := metrics.ComputeIfAbsentCache(dubboPushRt, func() interface{} {
		return newTimeMetrics(pushRtMin, pushRtMax, pushRtAvg, pushRtSum, pushRtLast, metrics.GetApplicationLevel(), c.r)
	}).(metrics.TimeMetric)
	metric.Record(event.CostMs())
}

func (c *MetadataMetricCollector) handleMetadataSub(event *MetadataMetricEvent) {
	m := metrics.ComputeIfAbsentCache(dubboMetadataSubscribe, func() interface{} {
		return newStatesMetricFunc(metadataSubNum, metadataSubNumSucceed, metadataSubNumFailed, metrics.GetApplicationLevel(), c.r)
	}).(metrics.StatesMetrics)
	m.Inc(event.Succ)
	metric := metrics.ComputeIfAbsentCache(dubboSubscribeRt, func() interface{} {
		return newTimeMetrics(subscribeRtMin, subscribeRtMax, subscribeRtAvg, subscribeRtSum, subscribeRtLast, metrics.GetApplicationLevel(), c.r)
	}).(metrics.TimeMetric)
	metric.Record(event.CostMs())
}

func (c *MetadataMetricCollector) handleStoreProvider(event *MetadataMetricEvent) {
	interfaceName := event.Attachment[constant.InterfaceKey]
	m := metrics.ComputeIfAbsentCache(dubboMetadataStoreProvider+":"+interfaceName, func() interface{} {
		return newStatesMetricFunc(metadataStoreProvider, metadataStoreProviderSucceed, metadataStoreProviderFailed,
			metrics.NewServiceMetric(interfaceName), c.r)
	}).(metrics.StatesMetrics)
	m.Inc(event.Succ)
	metric := metrics.ComputeIfAbsentCache(dubboStoreProviderInterfaceRt+":"+interfaceName, func() interface{} {
		return newTimeMetrics(storeProviderInterfaceRtMin, storeProviderInterfaceRtMax, storeProviderInterfaceRtAvg,
			storeProviderInterfaceRtSum, storeProviderInterfaceRtLast, metrics.NewServiceMetric(interfaceName), c.r)
	}).(metrics.TimeMetric)
	metric.Record(event.CostMs())
}

func (c *MetadataMetricCollector) handleSubscribeService(event *MetadataMetricEvent) {
	interfaceName := event.Attachment[constant.InterfaceKey]
	metric := metrics.ComputeIfAbsentCache(dubboSubscribeServiceRt+":"+interfaceName, func() interface{} {
		return newTimeMetrics(subscribeServiceRtMin, subscribeServiceRtMax, subscribeServiceRtAvg, subscribeServiceRtSum,
			subscribeServiceRtLast, metrics.NewServiceMetric(interfaceName), c.r)
	}).(metrics.TimeMetric)
	metric.Record(event.CostMs())
}

func newStatesMetricFunc(total *metrics.MetricKey, succ *metrics.MetricKey, fail *metrics.MetricKey,
	level metrics.MetricLevel, reg metrics.MetricRegistry) metrics.StatesMetrics {
	return metrics.NewStatesMetrics(metrics.NewMetricId(total, level), metrics.NewMetricId(succ, level),
		metrics.NewMetricId(fail, level), reg)
}

func newTimeMetrics(min, max, avg, sum, last *metrics.MetricKey, level metrics.MetricLevel, mr metrics.MetricRegistry) metrics.TimeMetric {
	return metrics.NewTimeMetric(metrics.NewMetricId(min, level), metrics.NewMetricId(max, level), metrics.NewMetricId(avg, level),
		metrics.NewMetricId(sum, level), metrics.NewMetricId(last, level), mr)
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
	return &MetadataMetricEvent{Name: n, Start: time.Now(), Attachment: make(map[string]string)}
}

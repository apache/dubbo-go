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

package registry

import (
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	rc           registryCollector
	registryChan = make(chan metrics.MetricsEvent, 128)
)

func Collector(m metrics.MetricRegistry, c *metrics.ReporterConfig) {
	rc.regRegistry = m
}

func init() {
	metrics.AddCollector("registry_info", Collector)
	metrics.Subscribe(constant.MetricsRegistry, registryChan)
	go rc.start()
}

type registryCollector struct {
	regRegistry metrics.MetricRegistry
}

func (rc *registryCollector) start() {
	for event := range registryChan {
		if registryEvent, ok := event.(*RegistryMetricsEvent); ok {
			switch registryEvent.Name {
			case Reg:
				rc.regHandler(registryEvent)
			case Sub:
				rc.subHandler(registryEvent)
			case Notify:
				rc.notifyHandler(registryEvent)
			case ServerReg:
				rc.serverRegHandler(registryEvent)
			case ServerSub:
				rc.serverSubHandler(registryEvent)
			default:
			}
		}
	}
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

func (rc *registryCollector) regHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()

	m := metrics.ComputeIfAbsentCache(dubboRegNum, func() interface{} {
		return newStatesMetricFunc(RegisterMetricRequests, RegisterMetricRequestsSucceed, RegisterMetricRequestsFailed, level, rc.regRegistry)
	}).(metrics.StatesMetrics)
	m.Inc(event.Succ)

	metric := metrics.ComputeIfAbsentCache(dubboRegRt, func() interface{} {
		return newTimeMetrics(RegisterRtMillisecondsMin, RegisterRtMillisecondsMax, RegisterRtMillisecondsAvg, RegisterRtMillisecondsSum, RegisterRtMillisecondsLast, metrics.GetApplicationLevel(), rc.regRegistry)
	}).(metrics.TimeMetric)
	metric.Record(event.CostMs())
}

func (rc *registryCollector) subHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	m := newStatesMetricFunc(SubscribeMetricNum, SubscribeMetricNumSucceed, SubscribeMetricNumFailed, level, rc.regRegistry)
	m.Inc(event.Succ)
}

func (rc *registryCollector) notifyHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	rc.regRegistry.Counter(metrics.NewMetricId(NotifyMetricRequests, level)).Inc()
	rc.regRegistry.Gauge(metrics.NewMetricId(NotifyMetricNumLast, level)).Set(float64(event.End.UnixNano()) / float64(time.Second))

	metric := metrics.ComputeIfAbsentCache(dubboNotifyRt, func() interface{} {
		return newTimeMetrics(NotifyRtMillisecondsMin, NotifyRtMillisecondsMax, NotifyRtMillisecondsAvg, NotifyRtMillisecondsAvg, NotifyRtMillisecondsLast, metrics.GetApplicationLevel(), rc.regRegistry)
	}).(metrics.TimeMetric)
	metric.Record(event.CostMs())
}

func (rc *registryCollector) directoryHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	typ := event.Attachment["DirTyp"]
	switch typ {
	case NumAllInc:
		rc.regRegistry.Counter(metrics.NewMetricId(DirectoryMetricNumAll, level)).Inc()
	case NumAllDec:
		rc.regRegistry.Counter(metrics.NewMetricId(DirectoryMetricNumAll, level)).Add(-1)
	case NumDisableTotal:
		rc.regRegistry.Counter(metrics.NewMetricId(DirectoryMetricNumDisable, level)).Inc()
	case NumToReconnectTotal:
		rc.regRegistry.Counter(metrics.NewMetricId(DirectoryMetricNumToReconnect, level)).Inc()
	case NumValidTotal:
		rc.regRegistry.Counter(metrics.NewMetricId(DirectoryMetricNumValid, level)).Inc()
	default:
	}

}

func (rc *registryCollector) serverRegHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	m := metrics.ComputeIfAbsentCache(dubboRegServerNum, func() interface{} {
		return newStatesMetricFunc(ServiceRegisterMetricRequests, ServiceRegisterMetricRequestsSucceed, ServiceRegisterMetricRequestsFailed, level, rc.regRegistry)
	}).(metrics.StatesMetrics)
	m.Inc(event.Succ)

	metric := metrics.ComputeIfAbsentCache(dubboRegServerRt, func() interface{} {
		return newTimeMetrics(RegisterServiceRtMillisecondsMin, RegisterServiceRtMillisecondsMax, RegisterServiceRtMillisecondsAvg, RegisterServiceRtMillisecondsSum, RegisterServiceRtMillisecondsLast, metrics.GetApplicationLevel(), rc.regRegistry)
	}).(metrics.TimeMetric)
	metric.Record(event.CostMs())
}

func (rc *registryCollector) serverSubHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	m := newStatesMetricFunc(ServiceSubscribeMetricNum, ServiceSubscribeMetricNumSucceed, ServiceSubscribeMetricNumFailed, level, rc.regRegistry)
	m.Inc(event.Succ)
}

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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	registryChan = make(chan metrics.MetricsEvent, 128)
)

func init() {
	metrics.AddCollector("registry", func(m metrics.MetricRegistry, url *common.URL) {
		if url.GetParamBool(constant.RegistryEnabledKey, true) {
			rc := &registryCollector{metrics.BaseCollector{R: m}}
			go rc.start()
		}
	})
}

// registryCollector is the registry's metrics collector
type registryCollector struct {
	metrics.BaseCollector
}

func (rc *registryCollector) start() {
	metrics.Subscribe(constant.MetricsRegistry, registryChan)
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

// regHandler handles register metrics
func (rc *registryCollector) regHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	rc.StateCount(RegisterMetricRequests, RegisterMetricRequestsSucceed, RegisterMetricRequestsFailed, level, event.Succ)
	rc.R.Rt(metrics.NewMetricId(RegisterRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

// subHandler handles subscribe metrics
func (rc *registryCollector) subHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	rc.StateCount(SubscribeMetricNum, SubscribeMetricNumSucceed, SubscribeMetricNumFailed, level, event.Succ)
}

// notifyHandler handles notify metrics
func (rc *registryCollector) notifyHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	rc.R.Counter(metrics.NewMetricId(NotifyMetricRequests, level)).Inc()
	rc.R.Gauge(metrics.NewMetricId(NotifyMetricNumLast, level)).Set(event.CostMs())
	rc.R.Rt(metrics.NewMetricId(NotifyRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

// directoryHandler handles directory metrics
func (rc *registryCollector) directoryHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	typ := event.Attachment["DirTyp"]
	switch typ {
	case NumAllInc:
		rc.R.Counter(metrics.NewMetricId(DirectoryMetricNumAll, level)).Inc()
	case NumAllDec:
		rc.R.Counter(metrics.NewMetricId(DirectoryMetricNumAll, level)).Add(-1)
	case NumDisableTotal:
		rc.R.Counter(metrics.NewMetricId(DirectoryMetricNumDisable, level)).Inc()
	case NumToReconnectTotal:
		rc.R.Counter(metrics.NewMetricId(DirectoryMetricNumToReconnect, level)).Inc()
	case NumValidTotal:
		rc.R.Counter(metrics.NewMetricId(DirectoryMetricNumValid, level)).Inc()
	default:
	}

}

// serverRegHandler handles server register metrics
func (rc *registryCollector) serverRegHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	rc.StateCount(ServiceRegisterMetricRequests, ServiceRegisterMetricRequestsSucceed, ServiceRegisterMetricRequestsFailed, level, event.Succ)
	rc.R.Rt(metrics.NewMetricId(RegisterServiceRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

// serverSubHandler handles server subscribe metrics
func (rc *registryCollector) serverSubHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics
	// Save metrics to the MetricRegistry
	level := metrics.GetApplicationLevel()
	rc.StateCount(ServiceSubscribeMetricNum, ServiceSubscribeMetricNumSucceed, ServiceSubscribeMetricNumFailed, level, event.Succ)
}

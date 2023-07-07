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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	//regRegistry  metrics.MetricRegistry
	registryChan = make(chan metrics.MetricsEvent, 128)
	handlers     []func(event *RegistryMetricsEvent)
)

//func Collector(m metrics.MetricRegistry, r *metrics.ReporterConfig) {
//	regRegistry = m
//
//	// init related metrics
//}

func init() {
	AddHandler(regHandler, subHandler, notifyHandler, directoryHandler, serverSubHandler, serverRegHandler)
	//metrics.AddCollector(Collector)
	metrics.Subscribe(constant.MetricsRegistry, registryChan)
	go receiveEvent()
}

func receiveEvent() {
	for event := range registryChan {
		registryEvent, ok := event.(*RegistryMetricsEvent)
		if !ok {
			continue
		}
		for _, handler := range handlers {
			go func(handler func(event *RegistryMetricsEvent)) {
				handler(registryEvent)
			}(handler)
		}
	}
}

func AddHandler(handler ...func(event *RegistryMetricsEvent)) {
	for _, h := range handler {
		handlers = append(handlers, h)
	}
}

func regHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics

	// Save metrics to the MetricRegistry
}

func subHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics

	// Save metrics to the MetricRegistry

}

func notifyHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics

	// Save metrics to the MetricRegistry
}

func directoryHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics

	// Save metrics to the MetricRegistry
}

func serverRegHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics

	// Save metrics to the MetricRegistry
}

func serverSubHandler(event *RegistryMetricsEvent) {
	// Event is converted to metrics

	// Save metrics to the MetricRegistry
}

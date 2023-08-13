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

package rpc

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	rpcMetricsChan = make(chan metrics.MetricsEvent, 1024)
)

func init() {
	var collectorFunc metrics.CollectorFunc
	collectorFunc = func(registry metrics.MetricRegistry, c *metrics.ReporterConfig) {
		rc := &rpcCollector{
			registry:  registry,
			metricSet: buildMetricSet(registry),
		}
		go rc.start()
	}

	metrics.AddCollector("rpc", collectorFunc)
}

type rpcCollector struct {
	registry  metrics.MetricRegistry
	metricSet *metricSet
}

func (c *rpcCollector) start() {
	metrics.Subscribe(constant.MetricsRpc, rpcMetricsChan)
	for event := range rpcMetricsChan {
		if rpcEvent, ok := event.(*MetricsEvent); ok {
			switch rpcEvent.name {
			case BeforeInvoke:
				c.beforeInvokeHandler(rpcEvent)
			case AfterInvoke:
				c.afterInvokeHandler(rpcEvent)
			default:
			}
		}
	}
}

func (c *rpcCollector) beforeInvokeHandler(event *MetricsEvent) {
	url := event.invoker.GetURL()
	role := getRole(url)

	if role == "" {
		return
	}
	labels := buildLabels(url)
	c.recordQps(role, labels)
	c.incRequestsProcessingTotal(role, labels)
}

func (c *rpcCollector) afterInvokeHandler(event *MetricsEvent) {
	url := event.invoker.GetURL()
	role := getRole(url)

	if role == "" {
		return
	}
	labels := buildLabels(url)
	c.incRequestsTotal(role, labels)
	c.decRequestsProcessingTotal(role, labels)
	if event.result != nil {
		if event.result.Error() == nil {
			c.incRequestsSucceedTotal(role, labels)
		}
	}
}

func newMetricId(key *metrics.MetricKey, labels map[string]string) *metrics.MetricId {
	return &metrics.MetricId{Name: key.Name, Desc: key.Desc, Tags: labels}
}

func (c *rpcCollector) recordQps(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.metricSet.provider.qpsTotal.Record(labels)
	case consumerField:
		c.metricSet.consumer.qpsTotal.Record(labels)
	}
}

func (c *rpcCollector) incRequestsTotal(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.registry.Counter(newMetricId(c.metricSet.provider.requestsTotal, labels)).Inc()
	case consumerField:
		c.registry.Counter(newMetricId(c.metricSet.consumer.requestsTotal, labels)).Inc()
	}
}

func (c *rpcCollector) incRequestsProcessingTotal(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.registry.Gauge(newMetricId(c.metricSet.provider.requestsProcessingTotal, labels)).Inc()
	case consumerField:
		c.registry.Gauge(newMetricId(c.metricSet.consumer.requestsProcessingTotal, labels)).Inc()
	}
}

func (c *rpcCollector) decRequestsProcessingTotal(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.registry.Gauge(newMetricId(c.metricSet.provider.requestsProcessingTotal, labels)).Dec()
	case consumerField:
		c.registry.Gauge(newMetricId(c.metricSet.consumer.requestsProcessingTotal, labels)).Dec()
	}
}

func (c *rpcCollector) incRequestsSucceedTotal(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.registry.Counter(newMetricId(c.metricSet.provider.requestsSucceedTotal, labels)).Inc()
	case consumerField:
		c.registry.Counter(newMetricId(c.metricSet.consumer.requestsSucceedTotal, labels)).Inc()
	}
}

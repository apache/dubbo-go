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
	"github.com/dubbogo/gost/log/logger"
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
		if rpcEvent, ok := event.(*metricsEvent); ok {
			switch rpcEvent.name {
			case BeforeInvoke:
				c.beforeInvokeHandler(rpcEvent)
			case AfterInvoke:
				c.afterInvokeHandler(rpcEvent)
			default:
			}
		} else {
			logger.Error("Bad metrics event found in RPC collector")
		}
	}
}

func (c *rpcCollector) beforeInvokeHandler(event *metricsEvent) {
	url := event.invoker.GetURL()
	role := getRole(url)

	if role == "" {
		return
	}
	labels := buildLabels(url, event.invocation)
	c.recordQps(role, labels)
	c.incRequestsProcessingTotal(role, labels)
}

func (c *rpcCollector) afterInvokeHandler(event *metricsEvent) {
	url := event.invoker.GetURL()
	role := getRole(url)

	if role == "" {
		return
	}
	labels := buildLabels(url, event.invocation)
	c.incRequestsTotal(role, labels)
	c.decRequestsProcessingTotal(role, labels)
	if event.result != nil {
		if event.result.Error() == nil {
			c.incRequestsSucceedTotal(role, labels)
		}
	}
	c.reportRTMilliseconds(role, labels, event.costTime.Milliseconds())
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
		c.metricSet.provider.requestsTotal.Inc(labels)
		c.metricSet.provider.requestsTotalAggregate.Inc(labels)
	case consumerField:
		c.metricSet.consumer.requestsTotal.Inc(labels)
		c.metricSet.consumer.requestsTotalAggregate.Inc(labels)
	}
}

func (c *rpcCollector) incRequestsProcessingTotal(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.metricSet.provider.requestsProcessingTotal.Inc(labels)
	case consumerField:
		c.metricSet.consumer.requestsProcessingTotal.Inc(labels)
	}
}

func (c *rpcCollector) decRequestsProcessingTotal(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.metricSet.provider.requestsProcessingTotal.Dec(labels)
	case consumerField:
		c.metricSet.consumer.requestsProcessingTotal.Dec(labels)
	}
}

func (c *rpcCollector) incRequestsSucceedTotal(role string, labels map[string]string) {
	switch role {
	case providerField:
		c.metricSet.provider.requestsSucceedTotal.Inc(labels)
		c.metricSet.provider.requestsSucceedTotalAggregate.Inc(labels)
	case consumerField:
		c.metricSet.consumer.requestsSucceedTotal.Inc(labels)
		c.metricSet.consumer.requestsSucceedTotalAggregate.Inc(labels)
	}
}

func (c *rpcCollector) reportRTMilliseconds(role string, labels map[string]string, cost int64) {
	switch role {
	case providerField:
		c.metricSet.provider.rtMilliseconds.Record(labels, float64(cost))
		c.metricSet.provider.rtMillisecondsAggregate.Record(labels, float64(cost))
		c.metricSet.provider.rtMillisecondsQuantiles.Record(labels, float64(cost))
	case consumerField:
		c.metricSet.consumer.rtMilliseconds.Record(labels, float64(cost))
		c.metricSet.consumer.rtMillisecondsAggregate.Record(labels, float64(cost))
		c.metricSet.consumer.rtMillisecondsQuantiles.Record(labels, float64(cost))
	}
}

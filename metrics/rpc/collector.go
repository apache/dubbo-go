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
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	rpcMetricsChan = make(chan metrics.MetricsEvent, 1024)
)

// init will add the rpc collectorFunc to metrics.collectors slice, and lazy start the rpc collector goroutine
func init() {
	collectorFunc := func(registry metrics.MetricRegistry, url *common.URL) {
		if url.GetParamBool(constant.RpcEnabledKey, true) {
			rc := &rpcCollector{
				registry:  registry,
				metricSet: buildMetricSet(registry),
			}
			go rc.start()
		}
	}

	metrics.AddCollector("rpc", collectorFunc)
}

// rpcCollector is a collector which will collect the rpc metrics
type rpcCollector struct {
	registry  metrics.MetricRegistry
	metricSet *metricSet // metricSet is a struct which contains all metrics about rpc
}

// start will subscribe the rpc.metricsEvent from channel rpcMetricsChan, and handle the event from the channel
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
		} else {
			// TODO: Breaking down RPC exceptions further
			c.incRequestsFailedTotal(role, labels)
		}
	}
	c.reportRTMilliseconds(role, labels, event.costTime.Milliseconds())
}

func (c *rpcCollector) recordQps(role string, labels map[string]string) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.qpsTotal.Record(labels)
	case constant.SideConsumer:
		c.metricSet.consumer.qpsTotal.Record(labels)
	}
}

func (c *rpcCollector) incRequestsTotal(role string, labels map[string]string) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.requestsTotal.Inc(labels)
		c.metricSet.provider.requestsTotalAggregate.Inc(labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsTotal.Inc(labels)
		c.metricSet.consumer.requestsTotalAggregate.Inc(labels)
	}
}

func (c *rpcCollector) incRequestsProcessingTotal(role string, labels map[string]string) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.requestsProcessingTotal.Inc(labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsProcessingTotal.Inc(labels)
	}
}

func (c *rpcCollector) decRequestsProcessingTotal(role string, labels map[string]string) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.requestsProcessingTotal.Dec(labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsProcessingTotal.Dec(labels)
	}
}

func (c *rpcCollector) incRequestsSucceedTotal(role string, labels map[string]string) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.requestsSucceedTotal.Inc(labels)
		c.metricSet.provider.requestsSucceedTotalAggregate.Inc(labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsSucceedTotal.Inc(labels)
		c.metricSet.consumer.requestsSucceedTotalAggregate.Inc(labels)
	}
}

func (c *rpcCollector) incRequestsFailedTotal(role string, labels map[string]string) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.requestsFailedTotal.Inc(labels)
		c.metricSet.provider.requestsFailedTotalAggregate.Inc(labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsFailedTotal.Inc(labels)
		c.metricSet.consumer.requestsFailedTotalAggregate.Inc(labels)
	}
}

func (c *rpcCollector) reportRTMilliseconds(role string, labels map[string]string, cost int64) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.rtMilliseconds.Record(labels, float64(cost))
		c.metricSet.provider.rtMillisecondsAggregate.Record(labels, float64(cost))
		c.metricSet.provider.rtMillisecondsQuantiles.Record(labels, float64(cost))
	case constant.SideConsumer:
		c.metricSet.consumer.rtMilliseconds.Record(labels, float64(cost))
		c.metricSet.consumer.rtMillisecondsAggregate.Record(labels, float64(cost))
		c.metricSet.consumer.rtMillisecondsQuantiles.Record(labels, float64(cost))
	}
}

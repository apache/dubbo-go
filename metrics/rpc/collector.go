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
	"sync"
	"sync/atomic"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/internal/rpcexemplar"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	activeCollector atomic.Pointer[rpcCollector]
)

// init registers RPC recording synchronously before Metrics.Init returns.
func init() {
	collectorFunc := func(registry metrics.MetricRegistry, url *common.URL) {
		if url.GetParamBool(constant.RpcEnabledKey, true) {
			rc := newRPCCollector(registry, url)
			metrics.SubscribeRecorder(constant.MetricsRpc, rc.record)
			activeCollector.Store(rc)
			if rc.histogramRegistry != nil && url.GetParamBool(constant.HistogramExemplarsKey, false) {
				rpcexemplar.Enable()
			}
		}
	}

	metrics.AddCollector("rpc", collectorFunc)
}

// rpcCollector is a collector which will collect the rpc metrics
type rpcCollector struct {
	registry          metrics.MetricRegistry
	metricSet         *metricSet // metricSet is a struct which contains all metrics about rpc
	series            sync.Map   // seriesKey -> *rpcSeries; owned by this registry
	bindMu            sync.Mutex
	fallbackMu        sync.Mutex // preserve serialized calls for legacy registries
	reportFailure     sync.Once
	histogramRegistry metrics.HistogramProvider
	histogramOpts     *metrics.HistogramOpts
	aggregation       aggregationSettings
}

// aggregationSettings controls the sliding-window metrics: QPS, the *_aggregate
// counters, the aggregated RT and the RT quantiles. The zero value keeps them on
// with the default window.
type aggregationSettings struct {
	disabled bool
	window   metrics.Window
}

func aggregationSettingsOf(url *common.URL) aggregationSettings {
	return aggregationSettings{
		disabled: !url.GetParamBool(constant.AggregationEnabledKey, true),
		window: metrics.Window{
			BucketNum:         url.GetParamByIntValue(constant.AggregationBucketNumKey, metrics.DefaultBucketNum),
			TimeWindowSeconds: url.GetParamByIntValue(constant.AggregationTimeWindowSecondsKey, metrics.DefaultTimeWindowSeconds),
		}.OrDefault(),
	}
}

func newRPCCollector(registry metrics.MetricRegistry, url *common.URL) *rpcCollector {
	aggregation := aggregationSettingsOf(url)
	c := &rpcCollector{registry: registry, metricSet: buildMetricSet(registry, aggregation.window), aggregation: aggregation}
	if url.GetParamBool(constant.HistogramEnabledKey, false) {
		if configurable, ok := registry.(metrics.HistogramProvider); ok {
			c.histogramRegistry = configurable
			c.histogramOpts = rpcHistogramOpts(url.GetParam(constant.HistogramBucketsKey, ""))
		} else {
			logger.Warnf("[Metrics][RPC] registry %T does not support configurable histograms; RPC histograms disabled, existing metrics unchanged", registry)
		}
	}
	return c
}

// record is called by Publish, before best-effort observer delivery. It never
// lets a broken custom registry replace a business result or suppress callbacks.
func (c *rpcCollector) record(event metrics.MetricsEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			c.reportFailure.Do(func() { logger.Errorf("[Metrics][RPC] synchronous recording failed: %v", recovered) })
		}
	}()
	e, ok := event.(*metricsEvent)
	if !ok || e.collector != c || (e.role != constant.SideConsumer && e.role != constant.SideProvider) {
		return
	}
	if _, ok := c.registry.(metrics.GaugeFuncRegistry); !ok {
		// Retain record-time window gauges and serial registry access for custom
		// registries that do not implement the optional collection-time extension.
		c.fallbackMu.Lock()
		defer c.fallbackMu.Unlock()
		switch e.name {
		case BeforeInvoke:
			c.beforeInvokeHandler(e)
		case AfterInvoke:
			c.afterInvokeHandler(e)
		case ObservationExpired:
			c.observationExpiredHandler(e)
		}
		return
	}
	c.getSeries(e).record(e)
}

// Expiry closes monitoring only; it does not prove an RPC result or latency.
func (c *rpcCollector) observationExpiredHandler(event *metricsEvent) {
	if event.role != "" {
		c.decRequestsProcessingTotal(event.role, event.labels)
	}
}

func (c *rpcCollector) beforeInvokeHandler(event *metricsEvent) {
	role := event.role

	if role == "" {
		return
	}
	labels := event.labels
	c.recordQps(role, labels)
	c.incRequestsProcessingTotal(role, labels)
}

func (c *rpcCollector) afterInvokeHandler(event *metricsEvent) {
	role := event.role

	if role == "" {
		return
	}
	labels := event.labels
	c.incRequestsTotal(role, labels)
	if role == constant.SideConsumer {
		if event.retry {
			c.metricSet.consumer.retriesTotal.Inc(labels)
		} else {
			// Export the counter with requests_total, before the first retry, so
			// that it can be subtracted from requests_total.
			c.metricSet.consumer.retriesTotal.Add(labels, 0)
		}
	}
	c.decRequestsProcessingTotal(role, labels)
	if event.hasResult {
		if event.err == nil {
			c.incRequestsSucceedTotal(role, labels)
		} else {
			// Increment total failed count
			c.incRequestsFailedTotal(role, labels)
			// Classify and increment granular error metrics
			errType := classifyEvent(event)
			c.incRequestsFailedByType(role, labels, errType)
		}
	}
	c.reportRTMilliseconds(role, labels, event.costTime.Milliseconds())
	c.reportRTHistogram(event)
}

func (c *rpcCollector) recordQps(role string, labels map[string]string) {
	if c.aggregation.disabled {
		return
	}
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
		c.incAggregate(c.metricSet.provider.requestsTotalAggregate, labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsTotal.Inc(labels)
		c.incAggregate(c.metricSet.consumer.requestsTotalAggregate, labels)
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
		c.incAggregate(c.metricSet.provider.requestsSucceedTotalAggregate, labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsSucceedTotal.Inc(labels)
		c.incAggregate(c.metricSet.consumer.requestsSucceedTotalAggregate, labels)
	}
}

func (c *rpcCollector) incRequestsFailedTotal(role string, labels map[string]string) {
	switch role {
	case constant.SideProvider:
		c.metricSet.provider.requestsFailedTotal.Inc(labels)
		c.incAggregate(c.metricSet.provider.requestsFailedTotalAggregate, labels)
	case constant.SideConsumer:
		c.metricSet.consumer.requestsFailedTotal.Inc(labels)
		c.incAggregate(c.metricSet.consumer.requestsFailedTotalAggregate, labels)
	}
}

func (c *rpcCollector) incRequestsFailedByType(role string, labels map[string]string, errType ErrorType) {
	var ms *rpcCommonMetrics

	switch role {
	case constant.SideProvider:
		ms = &c.metricSet.provider.rpcCommonMetrics
	case constant.SideConsumer:
		ms = &c.metricSet.consumer.rpcCommonMetrics
	default:
		return
	}

	switch errType {
	case ErrorTypeNetworkFailure:
		ms.requestsNetworkFailedTotal.Inc(labels)
		c.incAggregate(ms.requestsNetworkFailedTotalAggregate, labels)
	case ErrorTypeCodec:
		ms.requestsCodecFailedTotal.Inc(labels)
		c.incAggregate(ms.requestsCodecFailedTotalAggregate, labels)
	case ErrorTypeTimeout:
		ms.requestsTimeoutTotal.Inc(labels)
		c.incAggregate(ms.requestsTimeoutTotalAggregate, labels)
	case ErrorTypeLimit:
		ms.requestsLimitTotal.Inc(labels)
		c.incAggregate(ms.requestsLimitTotalAggregate, labels)
	case ErrorTypeServiceUnavailable:
		ms.requestsServiceUnavailableTotal.Inc(labels)
		c.incAggregate(ms.requestsServiceUnavailableTotalAggregate, labels)
	case ErrorTypeBusinessFailed:
		ms.requestsBusinessFailedTotal.Inc(labels)
		c.incAggregate(ms.requestsBusinessFailedTotalAggregate, labels)
	default:
		ms.requestsUnknownFailedTotal.Inc(labels)
		c.incAggregate(ms.requestsUnknownFailedTotalAggregate, labels)
	}
}

func (c *rpcCollector) reportRTMilliseconds(role string, labels map[string]string, cost int64) {
	var common *rpcCommonMetrics
	switch role {
	case constant.SideProvider:
		common = &c.metricSet.provider.rpcCommonMetrics
	case constant.SideConsumer:
		common = &c.metricSet.consumer.rpcCommonMetrics
	default:
		return
	}
	common.rtMilliseconds.Record(labels, float64(cost))
	if !c.aggregation.disabled {
		common.rtMillisecondsAggregate.Record(labels, float64(cost))
		common.rtMillisecondsQuantiles.Record(labels, float64(cost))
	}
}

func (c *rpcCollector) incAggregate(vec metrics.AggregateCounterVec, labels map[string]string) {
	if !c.aggregation.disabled {
		vec.Inc(labels)
	}
}

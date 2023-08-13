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
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

type metricSet struct {
	provider *providerMetrics
	consumer *consumerMetrics
}

type providerMetrics struct {
	rpcCommonMetrics
}

type consumerMetrics struct {
	rpcCommonMetrics
}

type rpcCommonMetrics struct {
	qpsTotal                      metrics.QpsMetric
	requestsTotal                 metrics.CounterVec
	requestsTotalAggregate        metrics.AggregateCounterMetric
	requestsProcessingTotal       metrics.GaugeVec
	requestsSucceedTotal          metrics.CounterVec
	requestsSucceedTotalAggregate metrics.AggregateCounterMetric
	//rtMillisecondsMin             *GaugeVecWithSyncMap
	//rtMillisecondsMax             *GaugeVecWithSyncMap
	//rtMillisecondsSum             *prometheus.CounterVec
	//rtMillisecondsAvg             *GaugeVecWithSyncMap
	//rtMillisecondsLast            *prometheus.GaugeVec
	//rtMillisecondsQuantiles       *quantileGaugeVec
	//rtMillisecondsAggregate       *aggregateFunctionsGaugeVec
}

func buildMetricSet(registry metrics.MetricRegistry) *metricSet {
	ms := &metricSet{
		provider: &providerMetrics{},
		consumer: &consumerMetrics{},
	}
	ms.provider.init(registry)
	ms.consumer.init(registry)
	return ms
}

func (pm *providerMetrics) init(registry metrics.MetricRegistry) {
	pm.qpsTotal = metrics.NewQpsMetric(metrics.NewMetricKey("dubbo_provider_qps_total", "The number of requests received by the provider per second"), registry)
	pm.requestsTotal = metrics.NewCounterVec(metrics.NewMetricKey("dubbo_provider_requests_total", "The total number of received requests by the provider"), registry)
	pm.requestsTotalAggregate = metrics.NewAggregateCounterMetric(metrics.NewMetricKey("dubbo_provider_requests_total_aggregate", "The total number of received requests by the provider under the sliding window"), registry)
	pm.requestsProcessingTotal = metrics.NewGaugeVec(metrics.NewMetricKey("dubbo_provider_requests_processing_total", "The number of received requests being processed by the provider"), registry)
	pm.requestsSucceedTotal = metrics.NewCounterVec(metrics.NewMetricKey("dubbo_provider_requests_succeed_total", "The number of requests successfully received by the provider"), registry)
	pm.requestsSucceedTotalAggregate = metrics.NewAggregateCounterMetric(metrics.NewMetricKey("dubbo_provider_requests_succeed_total_aggregate", "The number of successful requests received by the provider under the sliding window"), registry)
}

func (cm *consumerMetrics) init(registry metrics.MetricRegistry) {
	cm.qpsTotal = metrics.NewQpsMetric(metrics.NewMetricKey("dubbo_consumer_qps_total", "The number of requests sent by consumers per second"), registry)
	cm.requestsTotal = metrics.NewCounterVec(metrics.NewMetricKey("dubbo_consumer_requests_total", "The total number of requests sent by consumers"), registry)
	cm.requestsTotalAggregate = metrics.NewAggregateCounterMetric(metrics.NewMetricKey("dubbo_consumer_requests_total_aggregate", "The total number of requests sent by consumers under the sliding window"), registry)
	cm.requestsProcessingTotal = metrics.NewGaugeVec(metrics.NewMetricKey("dubbo_consumer_requests_processing_total", "The number of received requests being processed by the consumer"), registry)
	cm.requestsSucceedTotal = metrics.NewCounterVec(metrics.NewMetricKey("dubbo_consumer_requests_succeed_total", "The number of successful requests sent by consumers"), registry)
	cm.requestsSucceedTotalAggregate = metrics.NewAggregateCounterMetric(metrics.NewMetricKey("dubbo_consumer_requests_succeed_total_aggregate", "The number of successful requests sent by consumers under the sliding window"), registry)
}

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

// metricSet is the metric set for rpc
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

// rpcCommonMetrics is the common metrics for both provider and consumer
type rpcCommonMetrics struct {
	qpsTotal                      metrics.QpsMetricVec
	requestsTotal                 metrics.CounterVec
	requestsTotalAggregate        metrics.AggregateCounterVec
	requestsProcessingTotal       metrics.GaugeVec
	requestsSucceedTotal          metrics.CounterVec
	requestsSucceedTotalAggregate metrics.AggregateCounterVec
	requestsFailedTotal           metrics.CounterVec
	requestsFailedTotalAggregate  metrics.AggregateCounterVec
	rtMilliseconds                metrics.RtVec
	rtMillisecondsQuantiles       metrics.QuantileMetricVec
	rtMillisecondsAggregate       metrics.RtVec
}

// buildMetricSet will call init functions to initialize the metricSet
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
	pm.qpsTotal = metrics.NewQpsMetricVec(registry, metrics.NewMetricKey("dubbo_provider_qps_total", "The number of requests received by the provider per second"))
	pm.requestsTotal = metrics.NewCounterVec(registry, metrics.NewMetricKey("dubbo_provider_requests_total", "The total number of received requests by the provider"))
	pm.requestsTotalAggregate = metrics.NewAggregateCounterVec(registry, metrics.NewMetricKey("dubbo_provider_requests_total_aggregate", "The total number of received requests by the provider under the sliding window"))
	pm.requestsProcessingTotal = metrics.NewGaugeVec(registry, metrics.NewMetricKey("dubbo_provider_requests_processing_total", "The number of received requests being processed by the provider"))
	pm.requestsSucceedTotal = metrics.NewCounterVec(registry, metrics.NewMetricKey("dubbo_provider_requests_succeed_total", "The number of requests successfully received by the provider"))
	pm.requestsSucceedTotalAggregate = metrics.NewAggregateCounterVec(registry, metrics.NewMetricKey("dubbo_provider_requests_succeed_total_aggregate", "The number of successful requests received by the provider under the sliding window"))
	pm.requestsFailedTotal = metrics.NewCounterVec(registry, metrics.NewMetricKey("dubbo_provider_requests_failed_total", "Total Failed Requests"))
	pm.requestsFailedTotalAggregate = metrics.NewAggregateCounterVec(registry, metrics.NewMetricKey("dubbo_provider_requests_failed_total_aggregate", "Total Failed Aggregate Requests"))
	pm.rtMilliseconds = metrics.NewRtVec(registry,
		metrics.NewMetricKey("dubbo_provider_rt_milliseconds", "response time among all requests processed by the provider"),
		&metrics.RtOpts{Aggregate: false},
	)
	pm.rtMillisecondsAggregate = metrics.NewRtVec(registry,
		metrics.NewMetricKey("dubbo_provider_rt", "response time of the provider under the sliding window"),
		&metrics.RtOpts{Aggregate: true, BucketNum: metrics.DefaultBucketNum, TimeWindowSeconds: metrics.DefaultTimeWindowSeconds},
	)
	pm.rtMillisecondsQuantiles = metrics.NewQuantileMetricVec(registry, []*metrics.MetricKey{
		metrics.NewMetricKey("dubbo_provider_rt_milliseconds_p50", "The total response time spent by providers processing 50% of requests"),
		metrics.NewMetricKey("dubbo_provider_rt_milliseconds_p90", "The total response time spent by providers processing 90% of requests"),
		metrics.NewMetricKey("dubbo_provider_rt_milliseconds_p95", "The total response time spent by providers processing 95% of requests"),
		metrics.NewMetricKey("dubbo_provider_rt_milliseconds_p99", "The total response time spent by providers processing 99% of requests"),
	}, []float64{0.5, 0.9, 0.95, 0.99})
}

func (cm *consumerMetrics) init(registry metrics.MetricRegistry) {
	cm.qpsTotal = metrics.NewQpsMetricVec(registry, metrics.NewMetricKey("dubbo_consumer_qps_total", "The number of requests sent by consumers per second"))
	cm.requestsTotal = metrics.NewCounterVec(registry, metrics.NewMetricKey("dubbo_consumer_requests_total", "The total number of requests sent by consumers"))
	cm.requestsTotalAggregate = metrics.NewAggregateCounterVec(registry, metrics.NewMetricKey("dubbo_consumer_requests_total_aggregate", "The total number of requests sent by consumers under the sliding window"))
	cm.requestsProcessingTotal = metrics.NewGaugeVec(registry, metrics.NewMetricKey("dubbo_consumer_requests_processing_total", "The number of received requests being processed by the consumer"))
	cm.requestsSucceedTotal = metrics.NewCounterVec(registry, metrics.NewMetricKey("dubbo_consumer_requests_succeed_total", "The number of successful requests sent by consumers"))
	cm.requestsSucceedTotalAggregate = metrics.NewAggregateCounterVec(registry, metrics.NewMetricKey("dubbo_consumer_requests_succeed_total_aggregate", "The number of successful requests sent by consumers under the sliding window"))
	cm.requestsFailedTotal = metrics.NewCounterVec(registry, metrics.NewMetricKey("dubbo_consumer_requests_failed_total", "Total Failed Requests"))
	cm.requestsFailedTotalAggregate = metrics.NewAggregateCounterVec(registry, metrics.NewMetricKey("dubbo_consumer_requests_failed_total_aggregate", "Total Failed Aggregate Requests"))
	cm.rtMilliseconds = metrics.NewRtVec(registry,
		metrics.NewMetricKey("dubbo_consumer_rt_milliseconds", "response time among all requests from consumers"),
		&metrics.RtOpts{Aggregate: false},
	)
	cm.rtMillisecondsAggregate = metrics.NewRtVec(registry,
		metrics.NewMetricKey("dubbo_consumer_rt", "response time of the consumer under the sliding window"),
		&metrics.RtOpts{Aggregate: true, BucketNum: metrics.DefaultBucketNum, TimeWindowSeconds: metrics.DefaultTimeWindowSeconds},
	)
	cm.rtMillisecondsQuantiles = metrics.NewQuantileMetricVec(registry, []*metrics.MetricKey{
		metrics.NewMetricKey("dubbo_consumer_rt_milliseconds_p50", "The total response time spent by consumers processing 50% of requests"),
		metrics.NewMetricKey("dubbo_consumer_rt_milliseconds_p90", "The total response time spent by consumers processing 90% of requests"),
		metrics.NewMetricKey("dubbo_consumer_rt_milliseconds_p95", "The total response time spent by consumers processing 95% of requests"),
		metrics.NewMetricKey("dubbo_consumer_rt_milliseconds_p99", "The total response time spent by consumers processing 99% of requests"),
	}, []float64{0.5, 0.9, 0.95, 0.99})
}

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
	//qpsTotal                      *qpsGaugeVec
	//requestsTotal                 *prometheus.CounterVec
	//requestsTotalAggregate        *aggregateCounterGaugeVec
	requestsProcessingTotal *metrics.MetricKey
	//requestsSucceedTotal          *prometheus.CounterVec
	//requestsSucceedTotalAggregate *aggregateCounterGaugeVec
	//rtMillisecondsMin             *GaugeVecWithSyncMap
	//rtMillisecondsMax             *GaugeVecWithSyncMap
	//rtMillisecondsSum             *prometheus.CounterVec
	//rtMillisecondsAvg             *GaugeVecWithSyncMap
	//rtMillisecondsLast            *prometheus.GaugeVec
	//rtMillisecondsQuantiles       *quantileGaugeVec
	//rtMillisecondsAggregate       *aggregateFunctionsGaugeVec
}

func buildMetricSet() *metricSet {
	ms := &metricSet{
		provider: &providerMetrics{},
		consumer: &consumerMetrics{},
	}
	ms.provider.init()
	ms.consumer.init()
	return ms
}

func (pm *providerMetrics) init() {
	pm.requestsProcessingTotal = metrics.NewMetricKey("dubbo_provider_requests_processing_total", "The number of received requests being processed by the provider")
}

func (cm *consumerMetrics) init() {
	cm.requestsProcessingTotal = metrics.NewMetricKey("dubbo_consumer_requests_processing_total", "The number of received requests being processed by the consumer")
}

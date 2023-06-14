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

package prometheus

import (
	"strings"
)

import (
	"github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

// metricSet is a set of metrics that are reported to prometheus in dubbo go
type metricSet struct {
	// report the consumer-side's rt gauge data
	consumerRTSummaryVec *prometheus.SummaryVec
	// report the provider-side's rt gauge data
	providerRTSummaryVec *prometheus.SummaryVec

	// report the provider-side's request total counter data
	providerRequestsTotalCounterVec *prometheus.CounterVec
	// report the provider-side's processing request counter data
	providerRequestsProcessingTotalGaugeVec *prometheus.GaugeVec
	// The number of requests successfully received by the provider
	providerRequestsSucceedTotalCounterVec *prometheus.CounterVec

	// report the consumer-side's request total counter data
	consumerRequestsTotalCounterVec *prometheus.CounterVec
	// report the consumer-side's processing request counter data
	consumerRequestsProcessingTotalGaugeVec *prometheus.GaugeVec
	// The number of successful requests sent by consumers
	consumerRequestsSucceedTotalCounterVec *prometheus.CounterVec
}

var labelNames = []string{applicationNameKey, groupKey, hostnameKey, interfaceKey, ipKey, methodKey, versionKey}

// init metric set and register to prometheus
func (ms *metricSet) initAndRegister(reporterConfig *metrics.ReporterConfig) {
	ms.consumerRTSummaryVec = newAutoSummaryVec(buildMetricsName(consumerField, rtField, milliSecondsField, summaryField), reporterConfig.Namespace, labelNames, reporterConfig.SummaryMaxAge)
	ms.providerRTSummaryVec = newAutoSummaryVec(buildMetricsName(providerField, rtField, milliSecondsField, summaryField), reporterConfig.Namespace, labelNames, reporterConfig.SummaryMaxAge)
	ms.consumerRequestsTotalCounterVec = newAutoCounterVec(buildMetricsName(consumerField, requestsField, totalField), reporterConfig.Namespace, labelNames)
	ms.providerRequestsTotalCounterVec = newAutoCounterVec(buildMetricsName(providerField, requestsField, totalField), reporterConfig.Namespace, labelNames)
	ms.consumerRequestsProcessingTotalGaugeVec = newAutoGaugeVec(buildMetricsName(consumerField, requestsField, processingField, totalField), reporterConfig.Namespace, labelNames)
	ms.providerRequestsProcessingTotalGaugeVec = newAutoGaugeVec(buildMetricsName(providerField, requestsField, processingField, totalField), reporterConfig.Namespace, labelNames)
	ms.consumerRequestsSucceedTotalCounterVec = newAutoCounterVec(buildMetricsName(consumerField, requestsField, succeedField, totalField), reporterConfig.Namespace, labelNames)
	ms.providerRequestsSucceedTotalCounterVec = newAutoCounterVec(buildMetricsName(providerField, requestsField, succeedField, totalField), reporterConfig.Namespace, labelNames)
}

func buildMetricsName(args ...string) string {
	sb := strings.Builder{}
	for _, arg := range args {
		sb.WriteString("_")
		sb.WriteString(arg)
	}
	res := strings.TrimPrefix(sb.String(), "_")
	return res
}

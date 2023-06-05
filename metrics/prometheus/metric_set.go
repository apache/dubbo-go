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
	providerRequestsProcessingGaugeVec *prometheus.GaugeVec

	// report the consumer-side's request total counter data
	consumerRequestsTotalCounterVec *prometheus.CounterVec
	// report the consumer-side's processing request counter data
	consumerRequestsProcessingGaugeVec *prometheus.GaugeVec
}

var labelNames = []string{applicationNameKey, groupKey, hostnameKey, interfaceKey, ipKey, methodKey, versionKey}

// init metric set and register to prometheus
func (ms *metricSet) initAndRegister(reporterConfig *metrics.ReporterConfig) {
	ms.consumerRTSummaryVec = newSummaryVec(buildMetricsName(consumerField, rtField, milliSecondsField, summaryField), reporterConfig.Namespace, labelNames, reporterConfig.SummaryMaxAge)
	ms.providerRTSummaryVec = newSummaryVec(buildMetricsName(providerField, rtField, milliSecondsField, summaryField), reporterConfig.Namespace, labelNames, reporterConfig.SummaryMaxAge)
	ms.consumerRequestsTotalCounterVec = newCounterVec(buildMetricsName(consumerField, requestsField, totalField), reporterConfig.Namespace, labelNames)
	ms.providerRequestsTotalCounterVec = newCounterVec(buildMetricsName(providerField, requestsField, totalField), reporterConfig.Namespace, labelNames)
	ms.consumerRequestsProcessingGaugeVec = newGaugeVec(buildMetricsName(consumerField, requestsField, processingField), reporterConfig.Namespace, labelNames)
	ms.providerRequestsProcessingGaugeVec = newGaugeVec(buildMetricsName(providerField, requestsField, processingField), reporterConfig.Namespace, labelNames)

	prometheus.DefaultRegisterer.MustRegister(
		ms.consumerRTSummaryVec,
		ms.providerRTSummaryVec,
		ms.consumerRequestsTotalCounterVec,
		ms.providerRequestsTotalCounterVec,
		ms.consumerRequestsProcessingGaugeVec,
		ms.providerRequestsProcessingGaugeVec,
	)
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

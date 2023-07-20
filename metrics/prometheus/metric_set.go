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
	"fmt"
	"strconv"
	"strings"
)

import (
	"github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

// metricSet is a set of metrics that are reported to prometheus in dubbo-go
type metricSet struct {
	provider providerMetrics
	consumer consumerMetrics
}

func (ms *metricSet) init(reporterConfig *metrics.ReporterConfig) {
	ms.provider.init(reporterConfig)
	ms.consumer.init(reporterConfig)
}

type rpcCommonMetrics struct {
	requestsTotal           *prometheus.CounterVec
	requestsProcessingTotal *prometheus.GaugeVec
	requestsSucceedTotal    *prometheus.CounterVec
	rtMillisecondsMin       *GaugeVecWithSyncMap
	rtMillisecondsMax       *GaugeVecWithSyncMap
	rtMillisecondsSum       *prometheus.CounterVec
	rtMillisecondsAvg       *GaugeVecWithSyncMap
	rtMillisecondsLast      *prometheus.GaugeVec
	rtMillisecondsQuantiles *quantileGaugeVec
}

type providerMetrics struct {
	rpcCommonMetrics
}

func (pm *providerMetrics) init(reporterConfig *metrics.ReporterConfig) {
	pm.requestsTotal = newAutoCounterVec(buildMetricsName(providerField, requestsField, totalField), reporterConfig.Namespace, labelNames)
	pm.requestsProcessingTotal = newAutoGaugeVec(buildMetricsName(providerField, requestsField, processingField, totalField), reporterConfig.Namespace, labelNames)
	pm.requestsSucceedTotal = newAutoCounterVec(buildMetricsName(providerField, requestsField, succeedField, totalField), reporterConfig.Namespace, labelNames)
	pm.rtMillisecondsMin = newAutoGaugeVecWithSyncMap(buildMetricsName(providerField, rtField, milliSecondsField, minField), reporterConfig.Namespace, labelNames)
	pm.rtMillisecondsMax = newAutoGaugeVecWithSyncMap(buildMetricsName(providerField, rtField, milliSecondsField, maxField), reporterConfig.Namespace, labelNames)
	pm.rtMillisecondsSum = newAutoCounterVec(buildMetricsName(providerField, rtField, milliSecondsField, sumField), reporterConfig.Namespace, labelNames)
	pm.rtMillisecondsAvg = newAutoGaugeVecWithSyncMap(buildMetricsName(providerField, rtField, milliSecondsField, avgField), reporterConfig.Namespace, labelNames)
	pm.rtMillisecondsLast = newAutoGaugeVec(buildMetricsName(providerField, rtField, milliSecondsField, lastField), reporterConfig.Namespace, labelNames)
	pm.rtMillisecondsQuantiles = newQuantileGaugeVec(buildRTQuantilesMetricsNames(providerField, quantiles), reporterConfig.Namespace, labelNames, quantiles)
}

type consumerMetrics struct {
	rpcCommonMetrics
}

func (cm *consumerMetrics) init(reporterConfig *metrics.ReporterConfig) {
	cm.requestsTotal = newAutoCounterVec(buildMetricsName(consumerField, requestsField, totalField), reporterConfig.Namespace, labelNames)
	cm.requestsProcessingTotal = newAutoGaugeVec(buildMetricsName(consumerField, requestsField, processingField, totalField), reporterConfig.Namespace, labelNames)
	cm.requestsSucceedTotal = newAutoCounterVec(buildMetricsName(consumerField, requestsField, succeedField, totalField), reporterConfig.Namespace, labelNames)
	cm.rtMillisecondsMin = newAutoGaugeVecWithSyncMap(buildMetricsName(consumerField, rtField, milliSecondsField, minField), reporterConfig.Namespace, labelNames)
	cm.rtMillisecondsMax = newAutoGaugeVecWithSyncMap(buildMetricsName(consumerField, rtField, milliSecondsField, maxField), reporterConfig.Namespace, labelNames)
	cm.rtMillisecondsSum = newAutoCounterVec(buildMetricsName(consumerField, rtField, milliSecondsField, sumField), reporterConfig.Namespace, labelNames)
	cm.rtMillisecondsAvg = newAutoGaugeVecWithSyncMap(buildMetricsName(consumerField, rtField, milliSecondsField, avgField), reporterConfig.Namespace, labelNames)
	cm.rtMillisecondsLast = newAutoGaugeVec(buildMetricsName(consumerField, rtField, milliSecondsField, lastField), reporterConfig.Namespace, labelNames)
	cm.rtMillisecondsQuantiles = newQuantileGaugeVec(buildRTQuantilesMetricsNames(consumerField, quantiles), reporterConfig.Namespace, labelNames, quantiles)
}

// buildMetricsName builds metrics name split by "_".
func buildMetricsName(args ...string) string {
	sb := strings.Builder{}
	for _, arg := range args {
		sb.WriteString("_")
		sb.WriteString(arg)
	}
	res := strings.TrimPrefix(sb.String(), "_")
	return res
}

// buildRTQuantilesMetricsNames is only used for building rt quantiles metric names.
func buildRTQuantilesMetricsNames(role string, quantiles []float64) []string {
	res := make([]string, 0, len(quantiles))
	for _, q := range quantiles {
		quantileField := fmt.Sprintf("p%v", strconv.FormatFloat(q*100, 'f', -1, 64))
		name := buildMetricsName(role, rtField, milliSecondsField, quantileField)
		res = append(res, name)
	}
	return res
}

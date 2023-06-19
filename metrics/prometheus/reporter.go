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
	"context"
	"net/http"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	reporterInstance *PrometheusReporter
	reporterInitOnce sync.Once
)

// should initialize after loading configuration
func init() {
	// newPrometheusReporter()
	extension.SetMetricReporter(reporterName, newPrometheusReporter)
}

// PrometheusReporter will collect the data for Prometheus
// if you want to use this feature, you need to initialize your prometheus.
// https://prometheus.io/docs/guides/go-application/
type PrometheusReporter struct {
	reporterServer *http.Server
	reporterConfig *metrics.ReporterConfig
	metricSet
	syncMaps
	namespace string
}

// newPrometheusReporter create new prometheusReporter
// it will register the metrics into prometheus
func newPrometheusReporter(reporterConfig *metrics.ReporterConfig) metrics.Reporter {
	if reporterInstance == nil {
		reporterInitOnce.Do(func() {
			ms := &metricSet{}
			ms.initAndRegister(reporterConfig)
			reporterInstance = &PrometheusReporter{
				reporterConfig: reporterConfig,
				namespace:      reporterConfig.Namespace,
				metricSet:      *ms,
			}
		})
	}

	if reporterConfig.Enable {
		if reporterConfig.Mode == metrics.ReportModePull {
			go reporterInstance.startupServer(reporterConfig)
		}
		// todo pushgateway support
	} else {
		reporterInstance.shutdownServer()
	}

	return reporterInstance
}

func (reporter *PrometheusReporter) startupServer(reporterConfig *metrics.ReporterConfig) {
	// start server
	mux := http.NewServeMux()
	mux.Handle(reporterConfig.Path, promhttp.Handler())
	reporterInstance.reporterServer = &http.Server{Addr: ":" + reporterConfig.Port, Handler: mux}
	if err := reporterInstance.reporterServer.ListenAndServe(); err != nil {
		logger.Warnf("new prometheus reporter with error = %s", err)
	}
}

func (reporter *PrometheusReporter) shutdownServer() {
	if reporterInstance.reporterServer != nil {
		err := reporterInstance.reporterServer.Shutdown(context.Background())
		if err != nil {
			logger.Errorf("shutdown prometheus reporter with error = %s, prometheus reporter close now", err)
			reporterInstance.reporterServer.Close()
		}
	}
}

func (reporter *PrometheusReporter) reportRTSummaryVec(role string, labels *prometheus.Labels, costMs int64) {
	switch role {
	case providerField:
		reporter.providerRTSummaryVec.With(*labels).Observe(float64(costMs))
	case consumerField:
		reporter.consumerRTSummaryVec.With(*labels).Observe(float64(costMs))
	}
}

func (reporter *PrometheusReporter) reportRequestsTotalCounterVec(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.providerRequestsTotalCounterVec.With(*labels).Inc()
	case consumerField:
		reporter.consumerRequestsTotalCounterVec.With(*labels).Inc()
	}
}

func (reporter *PrometheusReporter) incRequestsProcessingTotalGaugeVec(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.providerRequestsProcessingTotalGaugeVec.With(*labels).Inc()
	case consumerField:
		reporter.consumerRequestsProcessingTotalGaugeVec.With(*labels).Inc()
	}
}

func (reporter *PrometheusReporter) decRequestsProcessingTotalGaugeVec(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.providerRequestsProcessingTotalGaugeVec.With(*labels).Dec()
	case consumerField:
		reporter.consumerRequestsProcessingTotalGaugeVec.With(*labels).Dec()
	}
}

func (reporter *PrometheusReporter) incRequestsSucceedTotalCounterVec(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.providerRequestsSucceedTotalCounterVec.With(*labels).Inc()
	case consumerField:
		reporter.consumerRequestsSucceedTotalCounterVec.With(*labels).Inc()
	}
}

func (reporter *PrometheusReporter) updateRTMillisecondsMinGaugeVec(role string, labels *prometheus.Labels, costMs int64) {
	switch role {
	case providerField:
		go reporter.providerRTMillisecondsMinGaugeVecWithSyncMap.updateMin(labels, costMs)
	case consumerField:
		go reporter.consumerRTMillisecondsMinGaugeVecWithSyncMap.updateMin(labels, costMs)
	}
}

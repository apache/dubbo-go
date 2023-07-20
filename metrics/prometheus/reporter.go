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
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol"
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
			ms.init(reporterConfig)
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

func (reporter *PrometheusReporter) ReportBeforeInvocation(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) {
	if !reporter.reporterConfig.Enable {
		return
	}
	url := invoker.GetURL()

	role := getRole(url)
	if role == "" {
		return
	}
	labels := buildLabels(url)

	reporter.incRequestsProcessingTotal(role, &labels)
}

func (reporter *PrometheusReporter) ReportAfterInvocation(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, cost time.Duration, res protocol.Result) {
	if !reporter.reporterConfig.Enable {
		return
	}
	url := invoker.GetURL()

	role := getRole(url)
	if role == "" {
		return
	}
	labels := buildLabels(url)

	reporter.incRequestsTotal(role, &labels)
	reporter.decRequestsProcessingTotal(role, &labels)
	reporter.reportRTMilliseconds(role, &labels, cost.Milliseconds())

	if res != nil && res.Error() == nil {
		// succeed
		reporter.incRequestsSucceedTotal(role, &labels)
	}
}

func (reporter *PrometheusReporter) incRequestsTotal(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.provider.requestsTotal.With(*labels).Inc()
	case consumerField:
		reporter.consumer.requestsTotal.With(*labels).Inc()
	}
}

func (reporter *PrometheusReporter) incRequestsProcessingTotal(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.provider.requestsProcessingTotal.With(*labels).Inc()
	case consumerField:
		reporter.consumer.requestsProcessingTotal.With(*labels).Inc()
	}
}

func (reporter *PrometheusReporter) decRequestsProcessingTotal(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.provider.requestsProcessingTotal.With(*labels).Dec()
	case consumerField:
		reporter.consumer.requestsProcessingTotal.With(*labels).Dec()
	}
}

func (reporter *PrometheusReporter) incRequestsSucceedTotal(role string, labels *prometheus.Labels) {
	switch role {
	case providerField:
		reporter.provider.requestsSucceedTotal.With(*labels).Inc()
	case consumerField:
		reporter.consumer.requestsSucceedTotal.With(*labels).Inc()
	}
}

func (reporter *PrometheusReporter) reportRTMilliseconds(role string, labels *prometheus.Labels, costMs int64) {
	switch role {
	case providerField:
		go reporter.provider.rtMillisecondsLast.With(*labels).Set(float64(costMs))
		go reporter.provider.rtMillisecondsSum.With(*labels).Add(float64(costMs))
		go reporter.provider.rtMillisecondsMin.updateMin(labels, costMs)
		go reporter.provider.rtMillisecondsMax.updateMax(labels, costMs)
		go reporter.provider.rtMillisecondsAvg.updateAvg(labels, costMs)
		go reporter.provider.rtMillisecondsQuantiles.updateQuantile(labels, costMs)
	case consumerField:
		go reporter.consumer.rtMillisecondsLast.With(*labels).Set(float64(costMs))
		go reporter.consumer.rtMillisecondsSum.With(*labels).Add(float64(costMs))
		go reporter.consumer.rtMillisecondsMin.updateMin(labels, costMs)
		go reporter.consumer.rtMillisecondsMax.updateMax(labels, costMs)
		go reporter.consumer.rtMillisecondsAvg.updateAvg(labels, costMs)
		go reporter.consumer.rtMillisecondsQuantiles.updateQuantile(labels, costMs)
	}
}

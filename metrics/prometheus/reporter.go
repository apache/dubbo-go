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

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/dubbogo/gost/log/logger"
	"github.com/prometheus/client_golang/prometheus"

	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	reporterInstance       *PrometheusReporter
	reporterInitOnce       sync.Once
	defaultHistogramBucket = []float64{10, 50, 100, 200, 500, 1000, 10000}
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
	metricsExporter, err := ocprom.NewExporter(ocprom.Options{
		Registry: prometheus.DefaultRegisterer.(*prometheus.Registry),
	})
	if err != nil {
		logger.Errorf("new prometheus reporter with error = %s", err)
		return
	}

	// start server
	mux := http.NewServeMux()
	mux.Handle(reporterConfig.Path, metricsExporter)
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

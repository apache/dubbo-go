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

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	reporterInstance *reporter
	reporterInitOnce sync.Once
)

const (
	reporterName = "prometheus"
)

// should initialize after loading configuration
func init() {
	// newPrometheusReporter()
	extension.SetMetricReporter(reporterName, newPrometheusReporter)
}

// reporter will export the metrics to Prometheus
// if you want to use this feature, you need to initialize your prometheus.
// https://prometheus.io/docs/guides/go-application/
type reporter struct {
	reporterServer *http.Server
	reporterConfig *metrics.ReporterConfig
	namespace      string
}

// newPrometheusReporter create a new prometheus server or push gateway reporter
func newPrometheusReporter(reporterConfig *metrics.ReporterConfig) metrics.Reporter {
	if reporterInstance == nil {
		reporterInitOnce.Do(func() {
			reporterInstance = &reporter{
				reporterConfig: reporterConfig,
				namespace:      reporterConfig.Namespace,
			}
		})
	}

	if reporterConfig.Enable {
		if reporterConfig.Mode == metrics.ReportModePull {
			go reporterInstance.StartServer(reporterConfig)
		}
		// todo pushgateway support
	} else {
		reporterInstance.ShutdownServer()
	}

	return reporterInstance
}

func (r *reporter) StartServer(reporterConfig *metrics.ReporterConfig) {
	// start server
	mux := http.NewServeMux()
	mux.Handle(reporterConfig.Path, promhttp.Handler())
	reporterInstance.reporterServer = &http.Server{Addr: ":" + reporterConfig.Port, Handler: mux}
	logger.Infof("new prometheus reporter with port = %s, path = %s", reporterConfig.Port, reporterConfig.Path)
	if err := reporterInstance.reporterServer.ListenAndServe(); err != nil {
		logger.Warnf("new prometheus reporter with error = %s", err)
	}
}

func (r *reporter) ShutdownServer() {
	if reporterInstance.reporterServer != nil {
		err := reporterInstance.reporterServer.Shutdown(context.Background())
		if err != nil {
			logger.Errorf("shutdown prometheus reporter with error = %s, prometheus reporter close now", err)
			reporterInstance.reporterServer.Close()
		}
	}
}

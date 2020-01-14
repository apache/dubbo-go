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

package monitor

import (
	"flag"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/monitor/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const (
	namespace = "dubbogo" // For Prometheus metrics.
)

var (
	listenAddr       = flag.String("web.listen-port", "9001", "An port to listen on for web interface and telemetry.")
	metricsPath      = flag.String("web.telemetry-path", "/metrics", "A path under which to expose metrics.")
	metricsNamespace = flag.String("metric.namespace", namespace, "Prometheus dubbogo metrics namespace, as the prefix of metrics name")
)

func main() {
	exporter, err := collector.NewExporter("dubbogo")
	if err != nil {
		logger.Error("create new Prometheus Dubbogo Exporter failed")
		return
	}
	registry := prometheus.NewRegistry()
	registry.MustRegister(exporter)


	http.Handle(*metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>A Prometheus Dubbogo Exporter</title></head>
			<body>
			<h1>A Prometheus Dubbogo Exporter</h1>
			<p><a href='/metrics'>Metrics</a></p>
			</body>
			</html>`))
	})

	logger.Info("Starting Server at http://localhost:%s%s", *listenAddr, *metricsPath)
	if err := http.ListenAndServe(":"+*listenAddr, nil); err != nil {
		logger.Errorf("Error occur when start Prometheus server %v", err)
	}
}

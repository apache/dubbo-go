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
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/prometheus/common/expfmt"

	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func TestGaugeFuncExportPaths(t *testing.T) {
	r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
	var value atomic.Int64
	value.Store(7)
	r.GaugeFunc(&metrics.MetricId{Name: "window_value", Desc: "Window value", Tags: map[string]string{"method": "A"}}, func() float64 {
		return float64(value.Load())
	})
	text, err := r.Scrape()
	require.NoError(t, err)
	require.Contains(t, text, "window_value{method=\"A\"} 7\n")
	value.Store(0)
	response := httptest.NewRecorder()
	handler := promhttp.InstrumentMetricHandler(r.r, promhttp.HandlerFor(r.gather, promhttp.HandlerOpts{}))
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "window_value{method=\"A\"} 0\n")

	body := make(chan string, 1)
	errors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		data, readErr := io.ReadAll(req.Body)
		errors <- readErr
		body <- string(data)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	value.Store(11)
	err = push.New(server.URL, "window-proof").Gatherer(r.gather).Format(expfmt.NewFormat(expfmt.TypeTextPlain)).Add()
	require.NoError(t, err)
	require.NoError(t, <-errors)
	require.Contains(t, <-body, "window_value{method=\"A\"} 11\n")
}

func TestGaugeFuncLabelIsolation(t *testing.T) {
	r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
	// These values collide under naive underscore concatenation.
	labels := map[string]string{"application_name": "a_b", "method": "c"}
	r.GaugeFunc(&metrics.MetricId{Name: "window_value", Desc: "Window value", Tags: labels}, func() float64 { return 1 })
	labels["application_name"] = "a"
	labels["method"] = "b_c"
	r.GaugeFunc(&metrics.MetricId{Name: "window_value", Desc: "Window value", Tags: labels}, func() float64 { return 2 })
	labels["method"] = "mutated"
	text, err := r.Scrape()
	require.NoError(t, err)
	require.Contains(t, text, "window_value{application_name=\"a_b\",method=\"c\"} 1\n")
	require.Contains(t, text, "window_value{application_name=\"a\",method=\"b_c\"} 2\n")
	require.NotContains(t, text, "mutated")
}

func TestGaugeFuncDoesNotChangeOrdinaryMetrics(t *testing.T) {
	r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
	gauge := r.Gauge(&metrics.MetricId{Name: "ordinary_gauge", Desc: "Ordinary gauge"})
	gauge.Set(3)
	gauge.Inc()
	counter := r.Counter(&metrics.MetricId{Name: "ordinary_counter_total", Desc: "Ordinary counter"})
	counter.Add(6)
	var value atomic.Int64
	value.Store(10)
	r.GaugeFunc(&metrics.MetricId{Name: "window_value", Desc: "Window value"}, func() float64 { return float64(value.Load()) })
	value.Store(0)
	for range 2 {
		text, err := r.Scrape()
		require.NoError(t, err)
		require.Contains(t, text, "ordinary_gauge 4\n")
		require.Contains(t, text, "ordinary_counter_total 6\n")
		require.Contains(t, text, "window_value 0\n")
	}
}

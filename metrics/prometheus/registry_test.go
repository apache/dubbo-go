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
	"sync"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	tags     = map[string]string{"app": "dubbo", "version": "1.0.0"}
	metricId = &metrics.MetricId{Name: "dubbo_request", Desc: "request", Tags: tags}
	url      = common.NewURLWithOptions(
		common.WithProtocol(constant.ProtocolPrometheus),
		common.WithParamsValue(constant.PrometheusExporterEnabledKey, "true"),
		common.WithParamsValue(constant.PrometheusExporterMetricsPortKey, constant.PrometheusDefaultMetricsPort),
		common.WithParamsValue(constant.PrometheusExporterMetricsPathKey, constant.PrometheusDefaultMetricsPath),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
		common.WithParamsValue(constant.AppVersionKey, "1.0.0"),
		common.WithParamsValue(constant.PrometheusPushgatewayEnabledKey, "true"),
		common.WithParamsValue(constant.PrometheusPushgatewayBaseUrlKey, "localhost:9091"),
		common.WithParamsValue(constant.PrometheusPushgatewayUsernameKey, ""),
		common.WithParamsValue(constant.PrometheusPushgatewayPasswordKey, ""),
		common.WithParamsValue(constant.PrometheusPushgatewayPushIntervalKey, "2"),
		common.WithParamsValue(constant.PrometheusPushgatewayJobKey, "dubbo-push"),
	)
)

func TestPromMetricRegistryCounter(t *testing.T) {
	p := NewPromMetricRegistry(prom.NewRegistry(), url)
	p.Counter(metricId).Inc()
	text, err := p.Scrape()
	assert.Nil(t, err)
	assert.Contains(t, text, "# HELP dubbo_request request\n# TYPE dubbo_request counter")
	assert.Contains(t, text, `dubbo_request{app="dubbo",version="1.0.0"} 1`)
}

func TestPromMetricRegistryGauge(t *testing.T) {
	p := NewPromMetricRegistry(prom.NewRegistry(), url)
	p.Gauge(metricId).Set(100)
	text, err := p.Scrape()
	assert.Nil(t, err)
	assert.Contains(t, text, "# HELP dubbo_request request\n# TYPE dubbo_request gauge")
	assert.Contains(t, text, `dubbo_request{app="dubbo",version="1.0.0"} 100`)

}

func TestPromMetricRegistryHistogram(t *testing.T) {
	p := NewPromMetricRegistry(prom.NewRegistry(), url)
	p.Histogram(metricId).Observe(100)
	text, err := p.Scrape()
	assert.Nil(t, err)
	assert.Contains(t, text, "# HELP dubbo_request request\n# TYPE dubbo_request histogram")
	assert.Contains(t, text, `dubbo_request_bucket{app="dubbo",version="1.0.0",le="+Inf"} 1`)
	assert.Contains(t, text, `dubbo_request_sum{app="dubbo",version="1.0.0"} 100`)
	assert.Contains(t, text, `dubbo_request_count{app="dubbo",version="1.0.0"} 1`)
}

func TestPromMetricRegistrySummary(t *testing.T) {
	p := NewPromMetricRegistry(prom.NewRegistry(), url)
	p.Summary(metricId).Observe(100)
	text, err := p.Scrape()
	assert.Nil(t, err)
	assert.Contains(t, text, "# HELP dubbo_request request\n# TYPE dubbo_request summary")
	assert.Contains(t, text, "dubbo_request_sum{app=\"dubbo\",version=\"1.0.0\"} 100")
	assert.Contains(t, text, "dubbo_request_count{app=\"dubbo\",version=\"1.0.0\"} 1")
}

func TestPromMetricRegistryRt(t *testing.T) {
	p := NewPromMetricRegistry(prom.NewRegistry(), url)
	for i := 0; i < 10; i++ {
		p.Rt(metricId, &metrics.RtOpts{}).Observe(10 * float64(i))
	}
	text, err := p.Scrape()
	assert.Nil(t, err)
	assert.Contains(t, text, "# HELP dubbo_request_avg Average request\n# TYPE dubbo_request_avg gauge\ndubbo_request_avg{app=\"dubbo\",version=\"1.0.0\"} 45")
	assert.Contains(t, text, "# HELP dubbo_request_last Last request\n# TYPE dubbo_request_last gauge\ndubbo_request_last{app=\"dubbo\",version=\"1.0.0\"} 90")
	assert.Contains(t, text, "# HELP dubbo_request_max Max request\n# TYPE dubbo_request_max gauge\ndubbo_request_max{app=\"dubbo\",version=\"1.0.0\"} 90")
	assert.Contains(t, text, "# HELP dubbo_request_min Min request\n# TYPE dubbo_request_min gauge\ndubbo_request_min{app=\"dubbo\",version=\"1.0.0\"} 0")
	assert.Contains(t, text, "# HELP dubbo_request_sum Sum request\n# TYPE dubbo_request_sum gauge\ndubbo_request_sum{app=\"dubbo\",version=\"1.0.0\"} 450")

	p = NewPromMetricRegistry(prom.NewRegistry(), url)
	for i := 0; i < 10; i++ {
		p.Rt(metricId, &metrics.RtOpts{Aggregate: true, BucketNum: 10, TimeWindowSeconds: 60}).Observe(10 * float64(i))
	}
	text, err = p.Scrape()
	assert.Nil(t, err)
	assert.Contains(t, text, "# HELP dubbo_request_avg_milliseconds_aggregate The average request\n# TYPE dubbo_request_avg_milliseconds_aggregate gauge\ndubbo_request_avg_milliseconds_aggregate{app=\"dubbo\",version=\"1.0.0\"} 45")
	assert.Contains(t, text, "# HELP dubbo_request_max_milliseconds_aggregate The maximum request\n# TYPE dubbo_request_max_milliseconds_aggregate gauge\ndubbo_request_max_milliseconds_aggregate{app=\"dubbo\",version=\"1.0.0\"} 90")
	assert.Contains(t, text, "# HELP dubbo_request_min_milliseconds_aggregate The minimum request\n# TYPE dubbo_request_min_milliseconds_aggregate gauge\ndubbo_request_min_milliseconds_aggregate{app=\"dubbo\",version=\"1.0.0\"} 0")
}

func TestPromMetricRegistryCounterConcurrent(t *testing.T) {
	p := NewPromMetricRegistry(prom.NewRegistry(), url)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			p.Counter(metricId).Inc()
			wg.Done()
		}()
	}
	wg.Wait()
	text, err := p.Scrape()
	assert.Nil(t, err)
	assert.Contains(t, text, "# HELP dubbo_request request\n# TYPE dubbo_request counter")
	assert.Contains(t, text, `dubbo_request{app="dubbo",version="1.0.0"} 10`)
}

func TestPromMetricRegistryExport(t *testing.T) {
	p := NewPromMetricRegistry(prom.NewRegistry(), url)
	go func() {
		for {
			p.Rt(metricId, &metrics.RtOpts{}).Observe(10 * float64(1))
			time.Sleep(1 * time.Second)
		}
	}()
	p.Export()
	// test push
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := http.ListenAndServe(url.GetParam(constant.PrometheusPushgatewayBaseUrlKey, ""),
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bodyBytes, err := io.ReadAll(r.Body)
				assert.Nil(t, err)
				text := string(bodyBytes)
				assert.Contains(t, text, "dubbo_request_avg")
				wg.Done()
			}))
		assert.Nil(t, err)
	}()
	timeout := url.GetParamByIntValue(constant.PrometheusPushgatewayPushIntervalKey, constant.PrometheusDefaultPushInterval)
	if waitTimeout(&wg, time.Duration(timeout+1)*time.Second) {
		assert.Fail(t, "wait pushgateway data timeout")
	}
	// test pull
	resp, err := http.Get("http://localhost:" +
		url.GetParam(constant.PrometheusExporterMetricsPortKey, constant.PrometheusDefaultMetricsPort) +
		url.GetParam(constant.PrometheusExporterMetricsPathKey, constant.PrometheusDefaultMetricsPath),
	)
	assert.Nil(t, err)
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	text := string(bodyBytes)
	assert.Contains(t, text, "dubbo_request_avg")
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

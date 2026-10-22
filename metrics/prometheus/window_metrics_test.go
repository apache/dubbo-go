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
	"math"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

type windowFixture struct {
	prefix    string
	qps       metrics.QpsMetricVec
	count     metrics.AggregateCounterVec
	total     metrics.CounterVec
	quantiles metrics.QuantileMetricVec
}

func newWindowFixture(r metrics.MetricRegistry, side string) *windowFixture {
	prefix := "dubbo_" + side
	keys := make([]*metrics.MetricKey, 4)
	for i, suffix := range []string{"p50", "p90", "p95", "p99"} {
		keys[i] = metrics.NewMetricKey(prefix+"_rt_milliseconds_"+suffix, "Window quantile")
	}
	return &windowFixture{
		prefix:    prefix,
		qps:       metrics.NewQpsMetricVec(r, metrics.NewMetricKey(prefix+"_qps_total", "Window QPS")),
		count:     metrics.NewAggregateCounterVec(r, metrics.NewMetricKey(prefix+"_requests_total_aggregate", "Window requests")),
		total:     metrics.NewCounterVec(r, metrics.NewMetricKey(prefix+"_requests_total", "Cumulative requests")),
		quantiles: metrics.NewQuantileMetricVec(r, keys, []float64{0.5, 0.9, 0.95, 0.99}),
	}
}

func (f *windowFixture) record(labels map[string]string, duration float64) {
	f.qps.Record(labels)
	f.count.Inc(labels)
	f.total.Inc(labels)
	f.quantiles.Record(labels, duration)
}

func windowValues(t *testing.T, r *promMetricRegistry) map[string]float64 {
	t.Helper()
	families, err := r.gather.Gather()
	require.NoError(t, err)
	values := make(map[string]float64)
	for _, family := range families {
		for _, metric := range family.Metric {
			method := ""
			for _, label := range metric.Label {
				if label.GetName() == "method" {
					method = label.GetValue()
				}
			}
			if metric.Gauge != nil {
				values[family.GetName()+"/"+method] = metric.Gauge.GetValue()
			} else if metric.Counter != nil {
				values[family.GetName()+"/"+method] = metric.Counter.GetValue()
			}
		}
	}
	return values
}

func TestWindowMetricsExpireOnCollection(t *testing.T) {
	for _, side := range []string{"provider", "consumer"} {
		t.Run(side, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
				f := newWindowFixture(r, side)
				text, err := r.Scrape()
				require.NoError(t, err)
				require.Empty(t, text, "Do not create series before recording")
				a := map[string]string{"application_name": "shop-user", "interface": "DemoService", "method": "A"}
				b := map[string]string{"application_name": "shop-user", "interface": "DemoService", "method": "B"}
				for range 6 {
					f.record(a, 100)
				}
				values := windowValues(t, r)
				require.InDelta(t, 6.0, values[f.prefix+"_requests_total_aggregate/A"], 0)
				require.InDelta(t, 0.5, values[f.prefix+"_qps_total/A"], 0)
				require.InDelta(t, 100.0, values[f.prefix+"_rt_milliseconds_p99/A"], 0)

				time.Sleep(60 * time.Second)
				f.record(b, 200)
				time.Sleep(61 * time.Second)
				for range 2 {
					values = windowValues(t, r)
					require.InDelta(t, 0.0, values[f.prefix+"_requests_total_aggregate/A"], 0)
					require.InDelta(t, 0.0, values[f.prefix+"_qps_total/A"], 0)
					for _, suffix := range []string{"p50", "p90", "p95", "p99"} {
						require.True(t, math.IsNaN(values[f.prefix+"_rt_milliseconds_"+suffix+"/A"]), "Empty window has no quantile")
						require.InDelta(t, 200.0, values[f.prefix+"_rt_milliseconds_"+suffix+"/B"], 0)
					}
					require.InDelta(t, 6.0, values[f.prefix+"_requests_total/A"], 0)
					require.InDelta(t, 1.0, values[f.prefix+"_requests_total_aggregate/B"], 0)
					require.InDelta(t, 1.0/12, values[f.prefix+"_qps_total/B"], 0)
				}

				f.record(a, 1000)
				values = windowValues(t, r)
				require.InDelta(t, 1.0, values[f.prefix+"_requests_total_aggregate/A"], 0)
				require.InDelta(t, 1.0/12, values[f.prefix+"_qps_total/A"], 0)
				require.InDelta(t, 1000.0, values[f.prefix+"_rt_milliseconds_p99/A"], 0)
				require.InDelta(t, 7.0, values[f.prefix+"_requests_total/A"], 0)
			})
		})
	}
}

func TestWindowMetricsRecreatedVectorsReplaceCallbacks(t *testing.T) {
	for _, side := range []string{"provider", "consumer"} {
		t.Run(side, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
				labels := map[string]string{"application_name": "shop-user", "interface": "DemoService", "method": "A"}
				old := newWindowFixture(r, side)
				for range 6 {
					old.record(labels, 100)
				}
				time.Sleep(60 * time.Second)
				current := newWindowFixture(r, side)
				for range 3 {
					current.record(labels, 300)
				}
				check := func(count, qps float64) {
					t.Helper()
					values := windowValues(t, r)
					require.InDelta(t, count, values[current.prefix+"_requests_total_aggregate/A"], 0)
					require.InDelta(t, qps, values[current.prefix+"_qps_total/A"], 0)
					for _, suffix := range []string{"p50", "p90", "p95", "p99"} {
						require.InDelta(t, 300.0, values[current.prefix+"_rt_milliseconds_"+suffix+"/A"], 0)
					}
				}
				check(3, 3.0/12)
				time.Sleep(61 * time.Second)
				check(3, 3.0/12) // The old window is empty, but the replacement is still live.
				// Recording through the old vector must not reclaim the callbacks.
				old.record(labels, 100)
				for range 2 {
					current.record(labels, 300)
				}
				for range 2 {
					check(5, 5.0/24) // Two live buckets, each spanning 12 seconds.
				}
				values := windowValues(t, r)
				require.InDelta(t, 12.0, values[current.prefix+"_requests_total/A"], 0,
					"Cumulative counters remain shared and are not reset by vector recreation")
				time.Sleep(121 * time.Second)
				values = windowValues(t, r)
				require.InDelta(t, 0.0, values[current.prefix+"_requests_total_aggregate/A"], 0)
				require.InDelta(t, 0.0, values[current.prefix+"_qps_total/A"], 0)
				for _, suffix := range []string{"p50", "p90", "p95", "p99"} {
					require.True(t, math.IsNaN(values[current.prefix+"_rt_milliseconds_"+suffix+"/A"]))
				}
			})
		})
	}
}

func TestWindowMetricsConcurrentRecordAndCollect(t *testing.T) {
	r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
	f := newWindowFixture(r, "provider")
	const workers = 8
	const records = 100
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Go(func() {
			labels := map[string]string{"application_name": "shop-user", "interface": "DemoService", "method": "A"}
			for range records {
				f.record(labels, 100)
			}
		})
		wg.Go(func() {
			for range records {
				if _, err := r.gather.Gather(); err != nil {
					errs <- err
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	values := windowValues(t, r)
	require.InDelta(t, float64(workers*records), values[f.prefix+"_requests_total_aggregate/A"], 0)
	require.InDelta(t, float64(workers*records), values[f.prefix+"_requests_total/A"], 0)
	require.InDelta(t, 100.0, values[f.prefix+"_rt_milliseconds_p99/A"], 0)
	require.Greater(t, values[f.prefix+"_qps_total/A"], 0.0)
}

func TestWindowMetricsApplicationIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
		f := newWindowFixture(r, "provider")
		labels := map[string]string{"application_name": "app-a", "interface": "DemoService", "method": "Call"}
		f.record(labels, 100)
		time.Sleep(60 * time.Second)
		labels["application_name"] = "app-b"
		f.record(labels, 200)
		time.Sleep(61 * time.Second)
		text, err := r.Scrape()
		require.NoError(t, err)
		require.Contains(t, text, fmt.Sprintf("%s_requests_total_aggregate{application_name=\"app-a\",interface=\"DemoService\",method=\"Call\"} 0\n", f.prefix))
		require.Contains(t, text, fmt.Sprintf("%s_requests_total_aggregate{application_name=\"app-b\",interface=\"DemoService\",method=\"Call\"} 1\n", f.prefix))
		require.Contains(t, text, fmt.Sprintf("%s_rt_milliseconds_p99{application_name=\"app-a\",interface=\"DemoService\",method=\"Call\"} NaN\n", f.prefix))
		require.Contains(t, text, fmt.Sprintf("%s_rt_milliseconds_p99{application_name=\"app-b\",interface=\"DemoService\",method=\"Call\"} 200\n", f.prefix))
	})
}

func TestWindowQuantileValues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := NewPromMetricRegistry(prom.NewRegistry(), common.NewURLWithOptions())
		f := newWindowFixture(r, "provider")
		labels := map[string]string{"method": "Call"}
		for i := 1; i <= 100; i++ {
			f.record(labels, float64(i))
		}
		values := windowValues(t, r)
		for suffix, want := range map[string]float64{"p50": 50.5, "p90": 90.5, "p95": 95.5, "p99": 99.5} {
			require.InDelta(t, want, values[f.prefix+"_rt_milliseconds_"+suffix+"/Call"], 0.01)
		}
	})
}

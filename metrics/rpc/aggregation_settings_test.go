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

package rpc

import (
	"strings"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsProm "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
)

func newConfiguredCollector(t *testing.T, params string, legacy bool) (*rpcCollector, *prom.Registry) {
	t.Helper()
	reg := prom.NewRegistry()
	promURL, err := common.NewURL("prometheus://127.0.0.1")
	require.NoError(t, err)
	var r metrics.MetricRegistry = metricsProm.NewPromMetricRegistry(reg, promURL)
	if legacy {
		// Hiding GaugeFunc takes the record-time path kept for custom registries.
		r = legacyRegistry{r}
	}
	url, err := common.NewURL("metrics://127.0.0.1?" + params)
	require.NoError(t, err)
	return newRPCCollector(r, url), reg
}

func recordSuccessfulCall(t *testing.T, c *rpcCollector) {
	t.Helper()
	before := recordingEvent(t, c, common.PROVIDER)
	c.record(before)
	c.record(comparisonCompletion(before, 0, []error{nil}))
}

func isAggregationMetric(name string) bool {
	return strings.HasSuffix(name, "_aggregate") || strings.HasSuffix(name, "_qps_total") ||
		strings.Contains(name, "_rt_milliseconds_p")
}

func TestAggregationSettingsOf(t *testing.T) {
	cases := []struct {
		name   string
		params string
		want   aggregationSettings
	}{
		{"defaults", "", aggregationSettings{window: metrics.DefaultWindow()}},
		{"disabled", constant.AggregationEnabledKey + "=false", aggregationSettings{disabled: true, window: metrics.DefaultWindow()}},
		{"window", constant.AggregationBucketNumKey + "=6&" + constant.AggregationTimeWindowSecondsKey + "=30",
			aggregationSettings{window: metrics.Window{BucketNum: 6, TimeWindowSeconds: 30}}},
		{"non-positive window keeps the default", constant.AggregationBucketNumKey + "=0&" + constant.AggregationTimeWindowSecondsKey + "=-1",
			aggregationSettings{window: metrics.DefaultWindow()}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url, err := common.NewURL("metrics://127.0.0.1?" + c.params)
			require.NoError(t, err)
			assert.Equal(t, c.want, aggregationSettingsOf(url))
		})
	}
}

// TestDisabledAggregationKeepsCumulativeMetrics covers both recording paths.
func TestDisabledAggregationKeepsCumulativeMetrics(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "collection-time gauges"
		if legacy {
			name = "record-time gauges"
		}
		t.Run(name, func(t *testing.T) {
			for _, enabled := range []bool{true, false} {
				params := ""
				if !enabled {
					params = constant.AggregationEnabledKey + "=false"
				}
				c, reg := newConfiguredCollector(t, params, legacy)
				recordSuccessfulCall(t, c)

				snapshot := scalarSnapshot(t, reg)
				assert.InDelta(t, 1.0, snapshot["dubbo_provider_requests_total"], 1e-9, "cumulative metrics are always exported")
				assert.Contains(t, snapshot, "dubbo_provider_rt_milliseconds_max")
				var aggregationMetrics []string
				for name := range snapshot {
					if isAggregationMetric(name) {
						aggregationMetrics = append(aggregationMetrics, name)
					}
				}
				if enabled {
					assert.Contains(t, aggregationMetrics, "dubbo_provider_requests_total_aggregate")
					assert.Contains(t, aggregationMetrics, "dubbo_provider_qps_total")
					assert.Contains(t, aggregationMetrics, "dubbo_provider_rt_milliseconds_p99")
					assert.Contains(t, aggregationMetrics, "dubbo_provider_rt_avg_milliseconds_aggregate")
				} else {
					assert.Empty(t, aggregationMetrics)
				}
			}
		})
	}
}

// TestAggregationWindowIsConfigurable checks that the configured window, not
// the 120-second default, decides how long a call stays in *_aggregate.
func TestAggregationWindowIsConfigurable(t *testing.T) {
	short, shortReg := newConfiguredCollector(t,
		constant.AggregationBucketNumKey+"=1&"+constant.AggregationTimeWindowSecondsKey+"=1", false)
	def, defReg := newConfiguredCollector(t, "", false)
	recordSuccessfulCall(t, short)
	recordSuccessfulCall(t, def)
	require.InDelta(t, 1.0, scalarSnapshot(t, shortReg)["dubbo_provider_requests_total_aggregate"], 1e-9)

	time.Sleep(2500 * time.Millisecond)

	shortSnapshot := scalarSnapshot(t, shortReg)
	assert.InDelta(t, 0.0, shortSnapshot["dubbo_provider_requests_total_aggregate"], 1e-9, "the call left the 1-second window")
	assert.InDelta(t, 1.0, shortSnapshot["dubbo_provider_requests_total"], 1e-9)
	assert.InDelta(t, 1.0, scalarSnapshot(t, defReg)["dubbo_provider_requests_total_aggregate"], 1e-9, "the default window still holds it")
}

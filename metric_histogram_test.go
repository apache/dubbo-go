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

package dubbo

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func TestHistogramConfigDefaultsAndURL(t *testing.T) {
	for _, present := range []bool{false, true} {
		rc := defaultInstanceOptions()
		rc.Metrics = &global.MetricsConfig{}
		if present {
			rc.Metrics.Histogram = &global.HistogramConfig{}
		}
		require.NoError(t, rc.initGlobalMetrics(false))
		require.NotNil(t, rc.Metrics.Histogram)
		require.NotNil(t, rc.Metrics.Histogram.Enabled)
		assert.False(t, *rc.Metrics.Histogram.Enabled)
		assert.Equal(t, []float64{0.1, 0.3, 0.5, 1, 3, 5, 10}, rc.Metrics.Histogram.Buckets)
		u := metricsURL(rc)
		assert.False(t, u.GetParamBool(constant.HistogramEnabledKey, true))
		assert.Equal(t, "0.1,0.3,0.5,1,3,5,10", u.GetParam(constant.HistogramBucketsKey, ""))
	}
}

func TestHistogramYAMLConfigAndURL(t *testing.T) {
	conf := NewLoaderConf(WithGenre("yaml"), WithBytes([]byte(`
dubbo:
  application:
    name: histogram-yaml
  metrics:
    enable: true
    histogram:
      enabled: true
      buckets: [0.0001, 0.001, 0.01, 1]
`)))
	rc, err := loadInstanceOptions(conf)
	require.NoError(t, err)
	require.NoError(t, rc.initGlobalMetrics(false))
	require.True(t, *rc.Metrics.Enable)
	require.True(t, *rc.Metrics.Histogram.Enabled)
	assert.False(t, *rc.Metrics.Aggregation.Enabled)
	assert.Equal(t, []float64{0.0001, 0.001, 0.01, 1}, rc.Metrics.Histogram.Buckets)
	u := metricsURL(rc)
	assert.True(t, u.GetParamBool(constant.HistogramEnabledKey, false))
	assert.Equal(t, "0.0001,0.001,0.01,1", u.GetParam(constant.HistogramBucketsKey, ""))
}

func TestHistogramFunctionalConfigAndClone(t *testing.T) {
	rc := defaultInstanceOptions()
	WithMetrics(metrics.WithEnabled(), metrics.WithHistogramEnabled(), metrics.WithHistogramBuckets([]float64{0.0005, 0.005}))(rc)
	require.NoError(t, rc.initGlobalMetrics(false))
	u := metricsURL(rc)
	assert.True(t, u.GetParamBool(constant.HistogramEnabledKey, false))
	assert.Equal(t, "0.0005,0.005", u.GetParam(constant.HistogramBucketsKey, ""))
	clone := rc.CloneMetrics()
	clone.Histogram.Buckets[0] = 10
	*clone.Histogram.Enabled = false
	assert.Equal(t, []float64{0.0005, 0.005}, rc.Metrics.Histogram.Buckets)
	assert.True(t, *rc.Metrics.Histogram.Enabled)
}

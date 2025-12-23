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

package metrics

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Metrics)
}

func TestNewOptions(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		opts := NewOptions()
		assert.NotNil(t, opts)
		assert.NotNil(t, opts.Metrics)
	})

	t.Run("with single option", func(t *testing.T) {
		opts := NewOptions(WithPrometheus())
		assert.NotNil(t, opts)
		assert.Equal(t, "prometheus", opts.Metrics.Protocol)
	})

	t.Run("with multiple options", func(t *testing.T) {
		opts := NewOptions(
			WithPrometheus(),
			WithPort(9090),
			WithPath("/metrics"),
		)
		assert.NotNil(t, opts)
		assert.Equal(t, "prometheus", opts.Metrics.Protocol)
		assert.Equal(t, "9090", opts.Metrics.Port)
		assert.Equal(t, "/metrics", opts.Metrics.Path)
	})
}

func TestWithAggregationEnabled(t *testing.T) {
	opts := NewOptions(WithAggregationEnabled())
	assert.NotNil(t, opts.Metrics.Aggregation.Enabled)
	assert.True(t, *opts.Metrics.Aggregation.Enabled)
}

func TestWithAggregationBucketNum(t *testing.T) {
	opts := NewOptions(WithAggregationBucketNum(20))
	assert.Equal(t, 20, opts.Metrics.Aggregation.BucketNum)
}

func TestWithAggregationTimeWindowSeconds(t *testing.T) {
	opts := NewOptions(WithAggregationTimeWindowSeconds(60))
	assert.Equal(t, 60, opts.Metrics.Aggregation.TimeWindowSeconds)
}

func TestWithPrometheus(t *testing.T) {
	opts := NewOptions(WithPrometheus())
	assert.Equal(t, "prometheus", opts.Metrics.Protocol)
}

func TestWithPrometheusExporterEnabled(t *testing.T) {
	opts := NewOptions(WithPrometheusExporterEnabled())
	assert.NotNil(t, opts.Metrics.Prometheus.Exporter.Enabled)
	assert.True(t, *opts.Metrics.Prometheus.Exporter.Enabled)
}

func TestWithPrometheusPushgatewayEnabled(t *testing.T) {
	opts := NewOptions(WithPrometheusPushgatewayEnabled())
	assert.NotNil(t, opts.Metrics.Prometheus.Pushgateway.Enabled)
	assert.True(t, *opts.Metrics.Prometheus.Pushgateway.Enabled)
}

func TestWithPrometheusGatewayUrl(t *testing.T) {
	opts := NewOptions(WithPrometheusGatewayUrl("http://localhost:9091"))
	assert.Equal(t, "http://localhost:9091", opts.Metrics.Prometheus.Pushgateway.BaseUrl)
}

func TestWithPrometheusGatewayJob(t *testing.T) {
	opts := NewOptions(WithPrometheusGatewayJob("test-job"))
	assert.Equal(t, "test-job", opts.Metrics.Prometheus.Pushgateway.Job)
}

func TestWithPrometheusGatewayUsername(t *testing.T) {
	opts := NewOptions(WithPrometheusGatewayUsername("admin"))
	assert.Equal(t, "admin", opts.Metrics.Prometheus.Pushgateway.Username)
}

func TestWithPrometheusGatewayPassword(t *testing.T) {
	opts := NewOptions(WithPrometheusGatewayPassword("secret"))
	assert.Equal(t, "secret", opts.Metrics.Prometheus.Pushgateway.Password)
}

func TestWithPrometheusGatewayInterval(t *testing.T) {
	opts := NewOptions(WithPrometheusGatewayInterval(60 * time.Second))
	assert.Equal(t, 60, opts.Metrics.Prometheus.Pushgateway.PushInterval)
}

func TestWithConfigCenterEnabled(t *testing.T) {
	opts := NewOptions(WithConfigCenterEnabled())
	assert.NotNil(t, opts.Metrics.EnableConfigCenter)
	assert.True(t, *opts.Metrics.EnableConfigCenter)
}

func TestWithMetadataEnabled(t *testing.T) {
	opts := NewOptions(WithMetadataEnabled())
	assert.NotNil(t, opts.Metrics.EnableMetadata)
	assert.True(t, *opts.Metrics.EnableMetadata)
}

func TestWithRegistryEnabled(t *testing.T) {
	opts := NewOptions(WithRegistryEnabled())
	assert.NotNil(t, opts.Metrics.EnableRegistry)
	assert.True(t, *opts.Metrics.EnableRegistry)
}

func TestWithEnabled(t *testing.T) {
	opts := NewOptions(WithEnabled())
	assert.NotNil(t, opts.Metrics.Enable)
	assert.True(t, *opts.Metrics.Enable)
}

func TestWithPort(t *testing.T) {
	opts := NewOptions(WithPort(8080))
	assert.Equal(t, "8080", opts.Metrics.Port)
}

func TestWithPath(t *testing.T) {
	opts := NewOptions(WithPath("/custom/metrics"))
	assert.Equal(t, "/custom/metrics", opts.Metrics.Path)
}

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
	// the metrics module and all sub-modules are disabled by default
	assert.Nil(t, opts.Metrics.Enable)
	assert.Nil(t, opts.Metrics.Aggregation.Enabled)
	assert.Nil(t, opts.Metrics.Probe.Enabled)
	assert.Nil(t, opts.Metrics.Prometheus.Pushgateway.Enabled)
	assert.Equal(t, "", opts.Metrics.Port)
	assert.Equal(t, "", opts.Metrics.Path)
	assert.Equal(t, "", opts.Metrics.Protocol)
}

func TestNewOptions(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		check   func(*testing.T, *Options)
	}{
		{
			name: "no options",
			check: func(t *testing.T, opts *Options) {
				assert.NotNil(t, opts)
				assert.NotNil(t, opts.Metrics)
			},
		},
		{
			name:    "single option",
			options: []Option{WithPrometheus()},
			check: func(t *testing.T, opts *Options) {
				assert.Equal(t, "prometheus", opts.Metrics.Protocol)
			},
		},
		{
			name:    "multiple options",
			options: []Option{WithPrometheus(), WithPort(9090), WithPath("/metrics")},
			check: func(t *testing.T, opts *Options) {
				assert.Equal(t, "prometheus", opts.Metrics.Protocol)
				assert.Equal(t, "9090", opts.Metrics.Port)
				assert.Equal(t, "/metrics", opts.Metrics.Path)
			},
		},
		{
			name:    "later option wins",
			options: []Option{WithPort(8080), WithPort(9090)},
			check: func(t *testing.T, opts *Options) {
				assert.Equal(t, "9090", opts.Metrics.Port)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(tt.options...)
			tt.check(t, opts)
		})
	}
}

func TestBoolFlagOptions(t *testing.T) {
	tests := []struct {
		name   string
		option Option
		get    func(*Options) *bool
	}{
		{
			name:   "WithAggregationEnabled",
			option: WithAggregationEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.Aggregation.Enabled },
		},
		{
			name:   "WithPrometheusExporterEnabled",
			option: WithPrometheusExporterEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.Prometheus.Exporter.Enabled },
		},
		{
			name:   "WithPrometheusPushgatewayEnabled",
			option: WithPrometheusPushgatewayEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.Prometheus.Pushgateway.Enabled },
		},
		{
			name:   "WithConfigCenterEnabled",
			option: WithConfigCenterEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.EnableConfigCenter },
		},
		{
			name:   "WithMetadataEnabled",
			option: WithMetadataEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.EnableMetadata },
		},
		{
			name:   "WithRegistryEnabled",
			option: WithRegistryEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.EnableRegistry },
		},
		{
			name:   "WithEnabled",
			option: WithEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.Enable },
		},
		{
			name:   "WithProbeEnabled",
			option: WithProbeEnabled(),
			get:    func(o *Options) *bool { return o.Metrics.Probe.Enabled },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(tt.option)
			assert.NotNil(t, opts.Metrics)
			got := tt.get(opts)
			assert.NotNil(t, got)
			assert.True(t, *got)
		})
	}
}

func TestStringFieldOptions(t *testing.T) {
	tests := []struct {
		name   string
		option Option
		want   string
		get    func(*Options) string
	}{
		{
			name:   "WithPrometheus",
			option: WithPrometheus(),
			want:   "prometheus",
			get:    func(o *Options) string { return o.Metrics.Protocol },
		},
		{
			name:   "WithPrometheusGatewayUrl",
			option: WithPrometheusGatewayUrl("http://localhost:9091"),
			want:   "http://localhost:9091",
			get:    func(o *Options) string { return o.Metrics.Prometheus.Pushgateway.BaseUrl },
		},
		{
			name:   "WithPrometheusGatewayJob",
			option: WithPrometheusGatewayJob("test-job"),
			want:   "test-job",
			get:    func(o *Options) string { return o.Metrics.Prometheus.Pushgateway.Job },
		},
		{
			name:   "WithPrometheusGatewayUsername",
			option: WithPrometheusGatewayUsername("admin"),
			want:   "admin",
			get:    func(o *Options) string { return o.Metrics.Prometheus.Pushgateway.Username },
		},
		{
			name:   "WithPrometheusGatewayPassword",
			option: WithPrometheusGatewayPassword("secret"),
			want:   "secret",
			get:    func(o *Options) string { return o.Metrics.Prometheus.Pushgateway.Password },
		},
		{
			name:   "WithPort",
			option: WithPort(8080),
			want:   "8080",
			get:    func(o *Options) string { return o.Metrics.Port },
		},
		{
			name:   "WithPath",
			option: WithPath("/custom/metrics"),
			want:   "/custom/metrics",
			get:    func(o *Options) string { return o.Metrics.Path },
		},
		{
			name:   "WithProbePort",
			option: WithProbePort(12345),
			want:   "12345",
			get:    func(o *Options) string { return o.Metrics.Probe.Port },
		},
		{
			name:   "WithProbeLivenessPath",
			option: WithProbeLivenessPath("/custom/live"),
			want:   "/custom/live",
			get:    func(o *Options) string { return o.Metrics.Probe.LivenessPath },
		},
		{
			name:   "WithProbeReadinessPath",
			option: WithProbeReadinessPath("/custom/ready"),
			want:   "/custom/ready",
			get:    func(o *Options) string { return o.Metrics.Probe.ReadinessPath },
		},
		{
			name:   "WithProbeStartupPath",
			option: WithProbeStartupPath("/custom/startup"),
			want:   "/custom/startup",
			get:    func(o *Options) string { return o.Metrics.Probe.StartupPath },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(tt.option)
			assert.Equal(t, tt.want, tt.get(opts))
		})
	}
}

func TestIntFieldOptions(t *testing.T) {
	tests := []struct {
		name   string
		option Option
		want   int
		get    func(*Options) int
	}{
		{
			name:   "WithAggregationBucketNum",
			option: WithAggregationBucketNum(20),
			want:   20,
			get:    func(o *Options) int { return o.Metrics.Aggregation.BucketNum },
		},
		{
			name:   "WithAggregationTimeWindowSeconds",
			option: WithAggregationTimeWindowSeconds(60),
			want:   60,
			get:    func(o *Options) int { return o.Metrics.Aggregation.TimeWindowSeconds },
		},
		{
			name:   "WithPrometheusGatewayInterval",
			option: WithPrometheusGatewayInterval(60 * time.Second),
			want:   60,
			get:    func(o *Options) int { return o.Metrics.Prometheus.Pushgateway.PushInterval },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(tt.option)
			assert.Equal(t, tt.want, tt.get(opts))
		})
	}
}

func TestWithProbeUseInternalState(t *testing.T) {
	tests := []struct {
		name string
		use  bool
		want bool
	}{
		{
			name: "use internal state",
			use:  true,
			want: true,
		},
		{
			name: "not use internal state",
			use:  false,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(WithProbeUseInternalState(tt.use))
			assert.NotNil(t, opts.Metrics.Probe.UseInternalState)
			assert.Equal(t, tt.want, *opts.Metrics.Probe.UseInternalState)
		})
	}
}

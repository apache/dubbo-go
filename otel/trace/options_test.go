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

package trace

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Otel)
	assert.NotNil(t, opts.Otel.TracingConfig)
}

func TestNewOptions(t *testing.T) {
	// Test with no options
	opts := NewOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Otel)
	assert.NotNil(t, opts.Otel.TracingConfig)

	// Test with multiple options
	opts = NewOptions(
		WithEnabled(),
		WithStdoutExporter(),
		WithEndpoint("localhost:4318"),
	)
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Otel.TracingConfig.Enable)
	assert.True(t, *opts.Otel.TracingConfig.Enable)
	assert.Equal(t, "stdout", opts.Otel.TracingConfig.Exporter)
	assert.Equal(t, "localhost:4318", opts.Otel.TracingConfig.Endpoint)
}

func TestWithEnabled(t *testing.T) {
	opts := NewOptions(WithEnabled())
	assert.NotNil(t, opts.Otel.TracingConfig.Enable)
	assert.True(t, *opts.Otel.TracingConfig.Enable)
}

func TestWithStdoutExporter(t *testing.T) {
	opts := NewOptions(WithStdoutExporter())
	assert.Equal(t, "stdout", opts.Otel.TracingConfig.Exporter)
}

func TestWithJaegerExporter(t *testing.T) {
	opts := NewOptions(WithJaegerExporter())
	assert.Equal(t, "jaeger", opts.Otel.TracingConfig.Exporter)
}

func TestWithZipkinExporter(t *testing.T) {
	opts := NewOptions(WithZipkinExporter())
	assert.Equal(t, "zipkin", opts.Otel.TracingConfig.Exporter)
}

func TestWithOtlpHttpExporter(t *testing.T) {
	opts := NewOptions(WithOtlpHttpExporter())
	assert.Equal(t, "otlp-http", opts.Otel.TracingConfig.Exporter)
}

func TestWithOtlpGrpcExporter(t *testing.T) {
	opts := NewOptions(WithOtlpGrpcExporter())
	assert.Equal(t, "otlp-grpc", opts.Otel.TracingConfig.Exporter)
}

func TestWithExporter(t *testing.T) {
	cases := []struct {
		name     string
		exporter string
	}{
		{
			name:     "custom exporter",
			exporter: "custom-exporter",
		},
		{
			name:     "empty exporter",
			exporter: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := NewOptions(WithExporter(c.exporter))
			assert.Equal(t, c.exporter, opts.Otel.TracingConfig.Exporter)
		})
	}
}

func TestWithW3cPropagator(t *testing.T) {
	opts := NewOptions(WithW3cPropagator())
	assert.Equal(t, "w3c", opts.Otel.TracingConfig.Propagator)
}

func TestWithB3Propagator(t *testing.T) {
	opts := NewOptions(WithB3Propagator())
	assert.Equal(t, "b3", opts.Otel.TracingConfig.Propagator)
}

func TestWithPropagator(t *testing.T) {
	cases := []struct {
		name       string
		propagator string
	}{
		{
			name:       "custom propagator",
			propagator: "custom-propagator",
		},
		{
			name:       "empty propagator",
			propagator: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := NewOptions(WithPropagator(c.propagator))
			assert.Equal(t, c.propagator, opts.Otel.TracingConfig.Propagator)
		})
	}
}

func TestWithRatio(t *testing.T) {
	cases := []struct {
		name  string
		ratio float64
	}{
		{
			name:  "ratio 0.0",
			ratio: 0.0,
		},
		{
			name:  "ratio 0.5",
			ratio: 0.5,
		},
		{
			name:  "ratio 1.0",
			ratio: 1.0,
		},
		{
			name:  "ratio 0.25",
			ratio: 0.25,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := NewOptions(WithRatio(c.ratio))
			assert.InDelta(t, c.ratio, opts.Otel.TracingConfig.SampleRatio, 0.01)
		})
	}
}

func TestWithRatioMode(t *testing.T) {
	opts := NewOptions(WithRatioMode())
	assert.Equal(t, "ratio", opts.Otel.TracingConfig.SampleMode)
}

func TestWithAlwaysMode(t *testing.T) {
	opts := NewOptions(WithAlwaysMode())
	assert.Equal(t, "always", opts.Otel.TracingConfig.SampleMode)
}

func TestWithNeverMode(t *testing.T) {
	opts := NewOptions(WithNeverMode())
	assert.Equal(t, "never", opts.Otel.TracingConfig.SampleMode)
}

func TestWithMode(t *testing.T) {
	cases := []struct {
		name string
		mode string
	}{
		{
			name: "ratio mode",
			mode: "ratio",
		},
		{
			name: "always mode",
			mode: "always",
		},
		{
			name: "never mode",
			mode: "never",
		},
		{
			name: "custom mode",
			mode: "custom-mode",
		},
		{
			name: "empty mode",
			mode: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := NewOptions(WithMode(c.mode))
			assert.Equal(t, c.mode, opts.Otel.TracingConfig.SampleMode)
		})
	}
}

func TestWithEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "localhost endpoint",
			endpoint: "localhost:4318",
		},
		{
			name:     "http endpoint",
			endpoint: "http://localhost:4318",
		},
		{
			name:     "empty endpoint",
			endpoint: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := NewOptions(WithEndpoint(c.endpoint))
			assert.Equal(t, c.endpoint, opts.Otel.TracingConfig.Endpoint)
		})
	}
}

func TestWithInsecure(t *testing.T) {
	opts := NewOptions(WithInsecure())
	assert.True(t, opts.Otel.TracingConfig.Insecure)
}

func TestMultipleOptions(t *testing.T) {
	opts := NewOptions(
		WithEnabled(),
		WithJaegerExporter(),
		WithW3cPropagator(),
		WithRatioMode(),
		WithRatio(0.75),
		WithEndpoint("localhost:14268"),
		WithInsecure(),
	)

	assert.NotNil(t, opts.Otel.TracingConfig.Enable)
	assert.True(t, *opts.Otel.TracingConfig.Enable)
	assert.Equal(t, "jaeger", opts.Otel.TracingConfig.Exporter)
	assert.Equal(t, "w3c", opts.Otel.TracingConfig.Propagator)
	assert.Equal(t, "ratio", opts.Otel.TracingConfig.SampleMode)
	assert.InDelta(t, 0.75, opts.Otel.TracingConfig.SampleRatio, 0.01)
	assert.Equal(t, "localhost:14268", opts.Otel.TracingConfig.Endpoint)
	assert.True(t, opts.Otel.TracingConfig.Insecure)
}

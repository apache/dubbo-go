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
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"go.opentelemetry.io/otel/propagation"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestDefaultExporter_GetTracerProvider(t *testing.T) {
	tracerProvider := sdktrace.NewTracerProvider()
	exporter := &DefaultExporter{
		TracerProvider: tracerProvider,
	}

	result := exporter.GetTracerProvider()
	assert.Equal(t, tracerProvider, result)
}

func TestDefaultExporter_GetPropagator(t *testing.T) {
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	exporter := &DefaultExporter{
		Propagator: propagator,
	}

	result := exporter.GetPropagator()
	assert.Equal(t, propagator, result)
}

func TestNewExporter_NilConfig(t *testing.T) {
	customFunc := func() (sdktrace.SpanExporter, error) {
		return nil, nil
	}

	tracerProvider, propagator, err := NewExporter(nil, customFunc)
	assert.Nil(t, tracerProvider)
	assert.Nil(t, propagator)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "otel exporter config is nil")
}

func TestNewExporter_CustomFuncError(t *testing.T) {
	config := &ExporterConfig{
		Exporter:    "test",
		SampleMode:  "always",
		Propagator:  "w3c",
		ServiceName: "test-service",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return nil, errors.New("custom func error")
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.Nil(t, tracerProvider)
	assert.Nil(t, propagator)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create test exporter")
}

func TestNewExporter_InvalidSampleMode(t *testing.T) {
	config := &ExporterConfig{
		Exporter:    "test",
		SampleMode:  "invalid-mode",
		Propagator:  "w3c",
		ServiceName: "test-service",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		// Return a simple stdout exporter as mock
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.Nil(t, tracerProvider)
	assert.Nil(t, propagator)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "otel sample mode invalid-mode not supported")
}

func TestNewExporter_SampleModeRatio(t *testing.T) {
	config := &ExporterConfig{
		Exporter:       "test",
		SampleMode:     "ratio",
		SampleRatio:    0.5,
		Propagator:     "w3c",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	assert.NotNil(t, propagator)
}

func TestNewExporter_SampleModeAlways(t *testing.T) {
	config := &ExporterConfig{
		Exporter:       "test",
		SampleMode:     "always",
		Propagator:     "w3c",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	assert.NotNil(t, propagator)
}

func TestNewExporter_SampleModeNever(t *testing.T) {
	config := &ExporterConfig{
		Exporter:       "test",
		SampleMode:     "never",
		Propagator:     "w3c",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	assert.NotNil(t, propagator)
}

func TestNewExporter_PropagatorW3c(t *testing.T) {
	config := &ExporterConfig{
		Exporter:       "test",
		SampleMode:     "always",
		Propagator:     "w3c",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	assert.NotNil(t, propagator)
}

func TestNewExporter_PropagatorB3(t *testing.T) {
	config := &ExporterConfig{
		Exporter:       "test",
		SampleMode:     "always",
		Propagator:     "b3",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	assert.NotNil(t, propagator)
}

func TestNewExporter_PropagatorEmpty(t *testing.T) {
	config := &ExporterConfig{
		Exporter:       "test",
		SampleMode:     "always",
		Propagator:     "",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	// When propagator is empty or unknown, it should be nil
	assert.Nil(t, propagator)
}

func TestNewExporter_PropagatorUnknown(t *testing.T) {
	config := &ExporterConfig{
		Exporter:       "test",
		SampleMode:     "always",
		Propagator:     "unknown-propagator",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	// When propagator is unknown, it should be nil
	assert.Nil(t, propagator)
}

func TestNewExporter_WithServiceInfo(t *testing.T) {
	config := &ExporterConfig{
		Exporter:         "test",
		SampleMode:       "always",
		Propagator:       "w3c",
		ServiceNamespace: "test-namespace",
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	assert.NotNil(t, propagator)
}

func TestNewExporter_CompleteConfig(t *testing.T) {
	config := &ExporterConfig{
		Exporter:         "test",
		Endpoint:         "localhost:4318",
		SampleMode:       "ratio",
		SampleRatio:      0.75,
		Propagator:       "b3",
		ServiceNamespace: "test-namespace",
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
		Insecure:         true,
	}

	customFunc := func() (sdktrace.SpanExporter, error) {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	tracerProvider, propagator, err := NewExporter(config, customFunc)
	assert.NoError(t, err)
	assert.NotNil(t, tracerProvider)
	assert.NotNil(t, propagator)
}

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

package stdout

import (
	"testing"

	"dubbo.apache.org/dubbo-go/v3/otel/trace"
	"github.com/stretchr/testify/assert"
)

func TestNewStdoutExporter(t *testing.T) {
	config := &trace.ExporterConfig{
		Exporter:    "stdout",
		SampleMode:  "always",
		Propagator:  "w3c",
		ServiceName: "test-service",
	}

	exporter, err := newStdoutExporter(config)
	assert.NoError(t, err)
	assert.NotNil(t, exporter)
	assert.NotNil(t, exporter.GetTracerProvider())
	assert.NotNil(t, exporter.GetPropagator())
}

func TestNewStdoutExporter_WithConfig(t *testing.T) {
	config := &trace.ExporterConfig{
		Exporter:       "stdout",
		SampleMode:     "ratio",
		SampleRatio:    0.5,
		Propagator:     "b3",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	exporter, err := newStdoutExporter(config)
	assert.NoError(t, err)
	assert.NotNil(t, exporter)
	assert.NotNil(t, exporter.GetTracerProvider())
	assert.NotNil(t, exporter.GetPropagator())
}

func TestNewStdoutExporter_Singleton(t *testing.T) {
	config := &trace.ExporterConfig{
		Exporter:    "stdout",
		SampleMode:  "always",
		Propagator:  "w3c",
		ServiceName: "test-service",
	}

	exporter1, err1 := newStdoutExporter(config)
	assert.NoError(t, err1)
	assert.NotNil(t, exporter1)

	exporter2, err2 := newStdoutExporter(config)
	assert.NoError(t, err2)
	assert.NotNil(t, exporter2)

	// Should return the same instance due to sync.Once
	assert.Equal(t, exporter1, exporter2)
}


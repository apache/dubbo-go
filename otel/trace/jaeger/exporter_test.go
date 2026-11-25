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

package jaeger

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
)

type warnRecorder struct {
	logger.Logger
	mu    sync.Mutex
	warns []string
}

func (r *warnRecorder) Warn(args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.warns = append(r.warns, fmt.Sprint(args...))
}

func TestNewJaegerExporterWarnsDeprecation(t *testing.T) {
	prevLogger := logger.GetLogger()
	rec := &warnRecorder{Logger: prevLogger}
	logger.SetLogger(rec)
	initOnce, instance = sync.Once{}, nil
	t.Cleanup(func() {
		logger.SetLogger(prevLogger)
		initOnce, instance = sync.Once{}, nil
	})

	config := &trace.ExporterConfig{Exporter: "jaeger", Endpoint: "http://127.0.0.1:14268/api/traces", SampleMode: "always", Propagator: "w3c"}
	exporter, err := newJaegerExporter(config)
	require.NoError(t, err)
	require.NotNil(t, exporter)
	t.Cleanup(func() { _ = exporter.GetTracerProvider().Shutdown(context.Background()) })
	_, err = newJaegerExporter(config)
	require.NoError(t, err)

	require.Len(t, rec.warns, 1, "the deprecation warning is logged once")
	assert.Contains(t, rec.warns[0], "jaeger exporter is deprecated")
	assert.Contains(t, rec.warns[0], "otlp-http")
}

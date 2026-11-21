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
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"go.opentelemetry.io/otel/exporters/jaeger"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
)

var (
	initOnce sync.Once
	instance *Exporter
)

func init() {
	extension.SetTraceExporter("jaeger", newJaegerExporter)
}

type Exporter struct {
	*trace.DefaultExporter
}

// deprecationWarning is logged once when the jaeger exporter is created.
const deprecationWarning = "[OTel][Trace] the jaeger exporter is deprecated and will be removed in a future release: " +
	"its OpenTelemetry module is no longer maintained. Jaeger accepts OTLP, so use the otlp-http or otlp-grpc exporter " +
	"with Jaeger's OTLP endpoint (port 4318 or 4317) instead"

func newJaegerExporter(config *trace.ExporterConfig) (trace.Exporter, error) {
	var initError error
	if instance == nil {
		initOnce.Do(func() {
			logger.Warn(deprecationWarning)
			customFunc := func() (sdktrace.SpanExporter, error) {
				return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.Endpoint)))
			}

			tracerProvider, propagator, err := trace.NewExporter(config, customFunc)
			if err != nil {
				initError = err
				return
			}

			instance = &Exporter{
				DefaultExporter: &trace.DefaultExporter{
					TracerProvider: tracerProvider,
					Propagator:     propagator,
				},
			}
		})
	}
	return instance, initError
}

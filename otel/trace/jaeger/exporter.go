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
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
	"errors"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"sync"
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

func newJaegerExporter(config *trace.ExporterConfig) (trace.Exporter, error) {
	var initError error
	if instance == nil {
		initOnce.Do(func() {
			if config == nil {
				initError = errors.New("otel jaeger exporter config is nil")
				return
			}

			exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.Endpoint)))
			if err != nil {
				logger.Errorf("failed to create jaeger exporter: %v", err)
				initError = err
			}

			var samplerOption sdktrace.TracerProviderOption
			switch config.SampleMode {
			case "ratio":
				samplerOption = sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.SampleRatio)))
			case "always":
				samplerOption = sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample()))
			case "never":
				samplerOption = sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.NeverSample()))
			default:
				msg := fmt.Sprintf("otel jaeger sample mode %s not supported", config.SampleMode)
				logger.Error(msg)
				initError = errors.New(msg)
				return
			}

			traceProvider := sdktrace.NewTracerProvider(
				samplerOption,
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(resource.NewSchemaless(
					semconv.ServiceNamespaceKey.String(config.ServiceNamespace),
					semconv.ServiceNameKey.String(config.ServiceName),
					semconv.ServiceVersionKey.String(config.ServiceVersion),
				)),
			)

			var propagator propagation.TextMapPropagator
			switch config.Propagator {
			case "w3c":
				propagator = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
			case "b3":
				b3Propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))
				propagator = propagation.NewCompositeTextMapPropagator(b3Propagator, propagation.Baggage{})
			}

			instance = &Exporter{
				DefaultExporter: &trace.DefaultExporter{
					TraceProvider: traceProvider,
					Propagator:    propagator,
				},
			}
		})
	}
	return instance, initError
}

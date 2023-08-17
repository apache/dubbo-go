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
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
	"errors"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"sync"
)

var (
	initOnce              sync.Once
	traceProviderInstance *sdktrace.TracerProvider
)

func init() {
	extension.SetTraceProvider("jaeger", newJaegerTraceProvider)
}

func newJaegerTraceProvider(config *trace.TraceProviderConfig) (*sdktrace.TracerProvider, error) {
	var initError error
	if traceProviderInstance == nil {
		initOnce.Do(func() {
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

			traceProviderInstance = sdktrace.NewTracerProvider(
				samplerOption,
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(resource.NewSchemaless(
					semconv.ServiceNamespaceKey.String(config.ServiceNamespace),
					semconv.ServiceNameKey.String(config.ServiceName),
					semconv.ServiceVersionKey.String(config.ServiceVersion),
				)),
			)
		})
	}
	return traceProviderInstance, initError
}

// Shutdown shutdowns the jaeger provider after all the tracing data is exported.
// TODO: add it to graceful shutdown
func Shutdown() error {
	if traceProviderInstance != nil {
		return traceProviderInstance.Shutdown(context.TODO())
	}
	return nil
}

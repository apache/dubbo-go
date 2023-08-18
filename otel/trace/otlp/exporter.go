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

package otlp

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"sync"
)

var (
	initHttpOnce sync.Once
	httpInstance *Exporter

	initGrpcOnce sync.Once
	grpcInstance *Exporter
)

func init() {
	extension.SetTraceExporter("otlp-http", newHttpExporter)
	extension.SetTraceExporter("otlp-grpc", newHttpExporter)
}

type Exporter struct {
	*trace.DefaultExporter
}

func newHttpExporter(config *trace.ExporterConfig) (trace.Exporter, error) {
	var initError error
	if httpInstance == nil {
		initHttpOnce.Do(func() {
			customFunc := func() (sdktrace.SpanExporter, error) {
				client := otlptracehttp.NewClient(otlptracehttp.WithEndpoint(config.Endpoint))
				return otlptrace.New(context.Background(), client)
			}

			tracerProvider, propagator, err := trace.NewExporter(config, customFunc)
			if err != nil {
				initError = err
				return
			}

			httpInstance = &Exporter{
				DefaultExporter: &trace.DefaultExporter{
					TracerProvider: tracerProvider,
					Propagator:     propagator,
				},
			}
		})
	}
	return httpInstance, initError
}

func newGrpcExporter(config *trace.ExporterConfig) (trace.Exporter, error) {
	var initError error
	if grpcInstance == nil {
		initGrpcOnce.Do(func() {
			customFunc := func() (sdktrace.SpanExporter, error) {
				client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(config.Endpoint))
				return otlptrace.New(context.Background(), client)
			}

			tracerProvider, propagator, err := trace.NewExporter(config, customFunc)
			if err != nil {
				initError = err
				return
			}

			grpcInstance = &Exporter{
				DefaultExporter: &trace.DefaultExporter{
					TracerProvider: tracerProvider,
					Propagator:     propagator,
				},
			}
		})
	}
	return grpcInstance, initError
}

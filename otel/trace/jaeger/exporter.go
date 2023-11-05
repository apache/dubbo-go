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

func newJaegerExporter(config *trace.ExporterConfig) (trace.Exporter, error) {
	var initError error
	if instance == nil {
		initOnce.Do(func() {
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

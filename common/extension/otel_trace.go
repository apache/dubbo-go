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

package extension

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
	"github.com/dubbogo/gost/log/logger"
)

var traceExporterMap = make(map[string]func(config *trace.ExporterConfig) (trace.Exporter, error), 4)

func SetTraceExporter(name string, createFunc func(config *trace.ExporterConfig) (trace.Exporter, error)) {
	traceExporterMap[name] = createFunc
}

func GetTraceExporter(name string, config *trace.ExporterConfig) (trace.Exporter, error) {
	createFunc, ok := traceExporterMap[name]
	if !ok {
		panic("Cannot find the trace provider with name " + name)
	}
	return createFunc(config)
}

func GetTraceShutdownCallback() func() {
	return func() {
		for name, createFunc := range traceExporterMap {
			if exporter, err := createFunc(nil); err == nil {
				if err := exporter.GetTracerProvider().Shutdown(context.Background()); err != nil {
					logger.Errorf("Graceful shutdown --- Failed to shutdown trace provider %s, error: %s", name, err.Error())
				} else {
					logger.Infof("Graceful shutdown --- Tracer provider of %s", name)
				}
			}
		}
	}
}

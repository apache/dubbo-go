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
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var traceProviderMap = make(map[string]func(config *trace.TraceProviderConfig) (*sdktrace.TracerProvider, error), 4)

func SetTraceProvider(name string, f func(config *trace.TraceProviderConfig) (*sdktrace.TracerProvider, error)) {
	traceProviderMap[name] = f
}

func GetTraceProvider(name string, config *trace.TraceProviderConfig) (*sdktrace.TracerProvider, error) {
	f, ok := traceProviderMap[name]
	if !ok {
		panic("Cannot find the trace provider with name " + name)
	}
	return f(config)
}

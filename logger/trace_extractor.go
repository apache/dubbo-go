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

package logger

import (
	"context"
)

import (
	"go.opentelemetry.io/otel/trace"
)

// TraceFields contains trace information extracted from context.
type TraceFields struct {
	TraceID    string
	SpanID     string
	TraceFlags string
}

// ExtractTraceFields extracts trace information from the given context.
func ExtractTraceFields(ctx context.Context) TraceFields {
	if ctx == nil {
		return TraceFields{}
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return TraceFields{}
	}

	return TraceFields{
		TraceID:    spanCtx.TraceID().String(),
		SpanID:     spanCtx.SpanID().String(),
		TraceFlags: spanCtx.TraceFlags().String(),
	}
}

func HasTrace(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	return spanCtx.IsValid()
}

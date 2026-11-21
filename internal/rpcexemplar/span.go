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

// Package rpcexemplar associates local RPC spans with invocations without
// registering tracing or metrics extensions.
package rpcexemplar

import (
	"context"
	"reflect"
	"sync/atomic"
)

import (
	"go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

var enabled atomic.Bool

// Enable activates span capture for filters built after RPC metrics initialize.
func Enable() { enabled.Store(true) }

// Enabled is read when constructing filters, outside the invocation path.
func Enabled() bool { return enabled.Load() }

type exemplarSpanKey struct{}

// IDs is the trace identity retained by an RPC latency observation.
type IDs struct {
	TraceID trace.TraceID
	SpanID  trace.SpanID
}

type exemplarSpan struct {
	invocation base.Invocation
	kind       trace.SpanKind
	ids        IDs
}

// WithSpan associates a span started by the local tracing filter with its invocation.
// Only non-nil pointer invocations can carry an exemplar. Equal values may
// represent unrelated calls, so a value invocation retains ordinary metrics.
func WithSpan(ctx context.Context, invocation base.Invocation, span trace.Span, kind trace.SpanKind) context.Context {
	if !span.IsRecording() || !hasPointerIdentity(invocation) {
		return ctx
	}
	spanContext := span.SpanContext()
	if !spanContext.IsValid() || !spanContext.IsSampled() {
		return ctx
	}
	return context.WithValue(ctx, exemplarSpanKey{}, exemplarSpan{
		invocation: invocation,
		kind:       kind,
		ids:        IDs{TraceID: spanContext.TraceID(), SpanID: spanContext.SpanID()},
	})
}

// Rebind carries span ownership across an internal rewrite of the same RPC.
// Matching the old invocation prevents a nested call from claiming its parent's Span.
func Rebind(ctx context.Context, original, rewritten base.Invocation) context.Context {
	local, ok := ctx.Value(exemplarSpanKey{}).(exemplarSpan)
	if !ok || local.invocation != original || !hasPointerIdentity(rewritten) {
		return ctx
	}
	local.invocation = rewritten
	return context.WithValue(ctx, exemplarSpanKey{}, local)
}

func hasPointerIdentity(invocation base.Invocation) bool {
	identity := reflect.ValueOf(invocation)
	return identity.Kind() == reflect.Pointer && !identity.IsNil()
}

// ForInvocation returns the IDs of the Span that owns this local invocation.
func ForInvocation(ctx context.Context, invocation base.Invocation, role string) *IDs {
	local, ok := ctx.Value(exemplarSpanKey{}).(exemplarSpan)
	if !ok || local.invocation != invocation {
		return nil
	}
	if (local.kind == trace.SpanKindClient && role == constant.SideConsumer) ||
		(local.kind == trace.SpanKindServer && role == constant.SideProvider) {
		snapshot := local.ids
		return &snapshot
	}
	return nil
}

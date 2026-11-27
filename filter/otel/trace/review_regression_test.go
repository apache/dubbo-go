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

package trace

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

import (
	gost "github.com/dubbogo/gost/log/logger"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/propagation"

	sdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/filter"
	llogger "dubbo.apache.org/dubbo-go/v3/logger/core/logrus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestTracePanicEndsSpanAndObservation(t *testing.T) {
	for _, server := range []bool{false, true} {
		t.Run(map[bool]string{false: "client", true: "server"}[server], func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdk.NewTracerProvider(sdk.WithSpanProcessor(recorder))
			defer tp.Shutdown(context.Background())
			var f filter.Filter = &otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
			if server {
				f = &otelServerFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
			}
			url, _ := common.NewURL("dubbo://127.0.0.1/Fixture")
			var o *rpcerror.Observation
			var completions []rpcerror.Completion
			invoker := evidenceInvoker{Invoker: base.NewBaseInvoker(url), invoke: func(ctx context.Context, _ base.Invocation) result.Result {
				o = rpcerror.ObservationFromContext(ctx)
				o.BeginAsync(ctx, time.Minute)
				o.OnComplete(func(c rpcerror.Completion) { completions = append(completions, c) })
				panic("downstream")
			}}
			func() {
				defer func() {
					if recover() != "downstream" {
						t.Error("panic changed")
					}
				}()
				f.Invoke(context.Background(), invoker, invocation.NewRPCInvocation("Call", nil, nil))
			}()
			if len(recorder.Ended()) != 1 || len(completions) != 1 || completions[0].Known {
				t.Fatal("panic leaked Span/observation")
			}
			o.Complete(errors.New("late"))
			if len(recorder.Ended()) != 1 || len(completions) != 1 {
				t.Fatal("late response completed twice")
			}
		})
	}
}

func TestTraceTypedNilIsSuccess(t *testing.T) {
	for _, server := range []bool{false, true} {
		recorder := tracetest.NewSpanRecorder()
		tp := sdk.NewTracerProvider(sdk.WithSpanProcessor(recorder))
		var f filter.Filter = &otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
		if server {
			f = &otelServerFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
		}
		url, _ := common.NewURL("dubbo://127.0.0.1/Fixture")
		var typedNil *tri.Error
		i := evidenceInvoker{Invoker: base.NewBaseInvoker(url), invoke: func(context.Context, base.Invocation) result.Result { return &result.RPCResult{Err: typedNil} }}
		f.Invoke(context.Background(), i, invocation.NewRPCInvocation("Call", nil, nil))
		spans := recorder.Ended()
		if len(spans) != 1 || spans[0].Status().Code != codes.Ok || len(spans[0].Events()) != 0 {
			t.Fatal("typed nil was recorded as failure or prevented span.End")
		}
		tp.Shutdown(context.Background())
	}
}

func TestLogrusOwnedServerSpanAndIndependentChild(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdk.NewTracerProvider(sdk.WithSpanProcessor(recorder))
	defer tp.Shutdown(context.Background())
	log := logrus.New()
	log.SetOutput(io.Discard)
	l := llogger.NewLogrusCtxLogger(&gost.DubboLogger{Logger: log}, true)
	url, _ := common.NewURL("dubbo://127.0.0.1/Fixture")
	i := evidenceInvoker{Invoker: base.NewBaseInvoker(url), invoke: func(ctx context.Context, _ base.Invocation) result.Result {
		l.CtxError(ctx, "same RPC failed")
		l.CtxErrorf(ctx, "same RPC %s", "failed")
		childCtx, child := tp.Tracer("test").Start(ctx, "independent-child")
		l.CtxError(childCtx, "child failed")
		child.End()
		rpcerror.RecordKind(ctx, rpcerror.KindBusiness)
		return &result.RPCResult{Err: errors.New("business")}
	}}
	(&otelServerFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}).Invoke(context.Background(), i, invocation.NewRPCInvocation("Call", nil, nil))
	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("span count: %d", len(spans))
	}
	for _, span := range spans {
		if len(span.Events()) != 1 || span.Status().Code != codes.Error {
			t.Fatalf("%s duplicate/missing errors: %d", span.Name(), len(span.Events()))
		}
		if span.Name() == "Call" && span.Status().Description != "business" {
			t.Fatal("logger overwrote RPC status")
		}
	}
}

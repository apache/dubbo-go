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
	"testing"
	"time"
)

import (
	gost "github.com/dubbogo/gost/log/logger"

	"go.opentelemetry.io/otel/propagation"

	sdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.uber.org/zap"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	zlogger "dubbo.apache.org/dubbo-go/v3/logger/core/zap"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type evidenceInvoker struct {
	base.Invoker
	invoke func(context.Context, base.Invocation) result.Result
}

func (i evidenceInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	return i.invoke(ctx, inv)
}

func TestTraceWithoutMetricsAndLoggerOwnership(t *testing.T) {
	for _, server := range []bool{false, true} {
		for _, origin := range []rpcerror.Origin{rpcerror.Unknown, rpcerror.Codec} {
			recorder := tracetest.NewSpanRecorder()
			tp := sdk.NewTracerProvider(sdk.WithSpanProcessor(recorder))
			url, _ := common.NewURL("dubbo://127.0.0.1/Fixture")
			log := zlogger.NewZapCtxLogger(&gost.DubboLogger{Logger: zap.NewNop().Sugar()}, true)
			invoker := evidenceInvoker{Invoker: base.NewBaseInvoker(url), invoke: func(ctx context.Context, inv base.Invocation) result.Result {
				observation := rpcerror.ObservationFromContext(ctx)
				if observation == nil {
					t.Fatal("trace requires metrics")
				}
				observation.BindSession("session")
				observation.Record("session", rpcerror.Mark(errors.New("decode"), origin))
				if server {
					rpcerror.RecordKind(ctx, rpcerror.KindBusiness)
				} else {
					rpcerror.RecordKindFor(ctx, inv, rpcerror.KindBusiness)
				}
				log.CtxError(ctx, "same RPC failed")
				return &result.RPCResult{Err: errors.New("application error")}
			}}
			inv := invocation.NewRPCInvocation("Call", nil, nil)
			if server {
				(&otelServerFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}).Invoke(context.Background(), invoker, inv)
			} else {
				(&otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}).Invoke(context.Background(), invoker, inv)
			}
			spans := recorder.Ended()
			if len(spans) != 1 || len(spans[0].Events()) != 1 {
				t.Fatalf("duplicate/missing RPC exception: %v", spans)
			}
			attrs := map[string]string{}
			for _, a := range spans[0].Attributes() {
				attrs[string(a.Key)] = a.Value.AsString()
			}
			want := "business"
			if origin == rpcerror.Codec {
				want = "codec"
			}
			if attrs["error.type"] != want || attrs["rpc.error.origin"] != origin.String() {
				t.Fatalf("attributes: %v", attrs)
			}
			tp.Shutdown(context.Background())
		}
	}
}

func TestTraceAsyncCompletion(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdk.NewTracerProvider(sdk.WithSpanProcessor(recorder))
	defer tp.Shutdown(context.Background())
	url, _ := common.NewURL("dubbo://127.0.0.1/Fixture")
	var observed *rpcerror.Observation
	invoker := evidenceInvoker{Invoker: base.NewBaseInvoker(url), invoke: func(ctx context.Context, inv base.Invocation) result.Result {
		observed = rpcerror.ClaimObservation(ctx, inv)
		if observed == nil {
			t.Fatal("unclaimable observation")
		}
		observed.BeginAsync(ctx, time.Minute)
		return &result.RPCResult{}
	}}
	(&otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}).Invoke(context.Background(), invoker, invocation.NewRPCInvocation("Call", nil, nil))
	if len(recorder.Ended()) != 0 {
		t.Fatal("span ended at async submission")
	}
	observed.RecordKind(rpcerror.KindTimeout)
	observed.Complete(context.DeadlineExceeded)
	observed.Complete(context.DeadlineExceeded)
	if spans := recorder.Ended(); len(spans) != 1 || len(spans[0].Events()) != 1 {
		t.Fatalf("bad async spans: %v", spans)
	}
}

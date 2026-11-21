/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	metricsProm "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestClassifyErrorDeclaredKinds(t *testing.T) {
	cause := errors.New("fixture")
	for kind, want := range map[rpcerror.Kind]ErrorType{
		rpcerror.KindTimeout:            ErrorTypeTimeout,
		rpcerror.KindLimit:              ErrorTypeLimit,
		rpcerror.KindServiceUnavailable: ErrorTypeServiceUnavailable,
		rpcerror.KindBusiness:           ErrorTypeBusinessFailed,
		rpcerror.KindCodec:              ErrorTypeCodec,
	} {
		err := fmt.Errorf("wrapped: %w", rpcerror.WithKind(cause, kind))
		if got := classifyError(err); got != want {
			t.Fatalf("%v: got %v want %v", kind, got, want)
		}
		if got := classifyError(err).String(); got != kind.String() {
			t.Fatalf("taxonomy names diverged: %q vs %q", got, kind.String())
		}
	}
}

func TestClassifyErrorPriority(t *testing.T) {
	cause := errors.New("fixture")
	for name, tc := range map[string]struct {
		err  error
		want ErrorType
	}{
		"triple timeout beats declared kind":    {tri.NewError(tri.CodeDeadlineExceeded, rpcerror.WithKind(cause, rpcerror.KindBusiness)), ErrorTypeTimeout},
		"triple business beats declared kind":   {tri.NewError(tri.CodeBizError, rpcerror.WithKind(cause, rpcerror.KindTimeout)), ErrorTypeBusinessFailed},
		"local origin beats a declared symptom": {rpcerror.WithKind(rpcerror.Mark(cause, rpcerror.Network), rpcerror.KindTimeout), ErrorTypeNetworkFailure},
		"declared kind beats unavailable":       {tri.NewError(tri.CodeUnavailable, rpcerror.WithKind(cause, rpcerror.KindLimit)), ErrorTypeLimit},
		"origin still applies without kind":     {rpcerror.Mark(cause, rpcerror.Network), ErrorTypeNetworkFailure},
		"deadline is a timeout":                 {fmt.Errorf("call: %w", context.DeadlineExceeded), ErrorTypeTimeout},
		"conflicting kinds stay unknown":        {errors.Join(rpcerror.WithKind(cause, rpcerror.KindLimit), rpcerror.WithKind(cause, rpcerror.KindBusiness)), ErrorTypeUnknown},
		"invoker sentinel":                      {base.ErrClientClosed, ErrorTypeServiceUnavailable},
		"destroyed invoker sentinel":            {fmt.Errorf("x: %w", base.ErrDestroyedInvoker), ErrorTypeServiceUnavailable},
		"missing reply stays unknown":           {base.ErrNoReply, ErrorTypeUnknown},
	} {
		if got := classifyError(tc.err); got != tc.want {
			t.Errorf("%s: got %v want %v", name, got, tc.want)
		}
	}
}

func TestClassifyEventUsesRequestScopedKind(t *testing.T) {
	plain := errors.New("biz")
	if got := classifyEvent(&metricsEvent{err: plain, kind: rpcerror.KindBusiness}); got != ErrorTypeBusinessFailed {
		t.Fatalf("observation kind ignored: %v", got)
	}
	declared := rpcerror.WithKind(plain, rpcerror.KindTimeout)
	if got := classifyEvent(&metricsEvent{err: declared, kind: rpcerror.KindBusiness}); got != ErrorTypeTimeout {
		t.Fatalf("a kind declared on the error must win: %v", got)
	}
	timeout := rpcerror.WithKind(plain, rpcerror.KindTimeout)
	if got := classifyEvent(&metricsEvent{err: timeout, origin: rpcerror.Codec}); got != ErrorTypeCodec {
		t.Fatalf("observed codec root cause must win over the timeout it caused: %v", got)
	}
	if got := classifyEvent(&metricsEvent{err: plain, origin: rpcerror.Codec}); got != ErrorTypeCodec {
		t.Fatalf("origin-only classification changed: %v", got)
	}
}

func TestErrorTypeNamesMatchFailureMetrics(t *testing.T) {
	want := map[ErrorType]string{
		ErrorTypeUnknown: "unknown", ErrorTypeTimeout: "timeout", ErrorTypeLimit: "limit",
		ErrorTypeServiceUnavailable: "service_unavailable", ErrorTypeBusinessFailed: "business",
		ErrorTypeNetworkFailure: "network", ErrorTypeCodec: "codec",
	}
	if len(want) != len(failureSuffixes) {
		t.Fatal("every failure metric needs a stable name")
	}
	for errType, name := range want {
		if errType.String() != name {
			t.Errorf("%d: got %q want %q", errType, errType.String(), name)
		}
	}
}

// TestCompletionKindReachesCounters covers the path used by the metrics
// filter: kind evidence recorded on the observation (e.g. by a provider proxy
// invoker) is counted in the dedicated failure metric on both sides.
func TestCompletionKindReachesCounters(t *testing.T) {
	for _, role := range []int{common.PROVIDER, common.CONSUMER} {
		t.Run(strconv.Itoa(role), func(t *testing.T) {
			reg := prom.NewRegistry()
			url, _ := common.NewURL("tri://127.0.0.1:20000/Fixture?application=fixture&registry.role=" + strconv.Itoa(role))
			registry := metricsProm.NewPromMetricRegistry(reg, url)
			c := &rpcCollector{registry: registry, metricSet: buildMetricSet(registry)}
			before := NewBeforeInvokeEvent(base.NewBaseInvoker(url), invocation.NewRPCInvocation("Call", nil, nil))
			c.beforeInvokeHandler(before.(*metricsEvent))
			completion := rpcerror.Completion{Err: errors.New("biz"), Known: true, Kind: rpcerror.KindBusiness, At: time.Now()}
			c.afterInvokeHandler(NewCompletionEvent(before, time.Millisecond, completion).(*metricsEvent))

			side := constant.SideConsumer
			if role == common.PROVIDER {
				side = constant.SideProvider
			}
			values := gatheredCounterAndGaugeValues(t, reg)
			prefix := "dubbo_" + side + "_requests_"
			if values[prefix+"business_failed_total"] != 1 || values[prefix+"failed_total"] != 1 || values[prefix+"unknown_failed_total"] != 0 {
				t.Fatalf("unexpected counters: %v", values)
			}
			assertProtocolLabel(t, reg, prefix+"failed_total", "tri")
		})
	}
}

func assertProtocolLabel(t *testing.T, reg *prom.Registry, name, want string) {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
		for _, m := range f.Metric {
			for _, l := range m.Label {
				if l.GetName() == constant.TagProtocol && l.GetValue() == want {
					return
				}
			}
		}
	}
	t.Fatalf("%s has no protocol=%q label", name, want)
}

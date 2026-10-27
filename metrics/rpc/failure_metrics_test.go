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
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestFailureOriginClassification(t *testing.T) {
	cause := errors.New("fixture")
	for _, tc := range []struct {
		code   tri.Code
		origin rpcerror.Origin
		want   ErrorType
	}{
		{tri.CodeInternal, rpcerror.Codec, ErrorTypeCodec},
		{tri.CodeInvalidArgument, rpcerror.Codec, ErrorTypeCodec},
		{tri.CodeUnavailable, rpcerror.Network, ErrorTypeNetworkFailure},
		{tri.CodeUnavailable, rpcerror.Unknown, ErrorTypeServiceUnavailable},
		{tri.CodePermissionDenied, rpcerror.Network, ErrorTypeServiceUnavailable},
		{tri.CodeDeadlineExceeded, rpcerror.Network, ErrorTypeTimeout},
		{tri.CodeResourceExhausted, rpcerror.Codec, ErrorTypeLimit},
		{tri.CodeBizError, rpcerror.Network, ErrorTypeBusinessFailed},
		{tri.CodeInternal, rpcerror.Unknown, ErrorTypeUnknown},
	} {
		err := fmt.Errorf("wrapped: %w", tri.NewError(tc.code, rpcerror.Mark(cause, tc.origin)))
		if got := classifyError(err); got != tc.want {
			t.Fatalf("%v/%v: got %v want %v", tc.code, tc.origin, got, tc.want)
		}
	}
}

func TestFailureCountersAndSnapshot(t *testing.T) {
	for _, role := range []int{common.PROVIDER, common.CONSUMER} {
		t.Run(strconv.Itoa(role), func(t *testing.T) {
			reg := prom.NewRegistry()
			url, _ := common.NewURL("dubbo://127.0.0.1:20880/Fixture?application=fixture&registry.role=" + strconv.Itoa(role))
			registry := metricsProm.NewPromMetricRegistry(reg, url)
			c := &rpcCollector{registry: registry, metricSet: buildMetricSet(registry)}
			invoker := base.NewBaseInvoker(url)
			inv := invocation.NewRPCInvocation("Call", nil, nil)
			for _, origin := range []rpcerror.Origin{rpcerror.Codec, rpcerror.Network} {
				res := &result.RPCResult{Err: rpcerror.Mark(errors.New("failure"), origin)}
				before := NewBeforeInvokeEvent(invoker, inv).(*metricsEvent)
				event := NewAfterInvokeEvent(invoker, inv, time.Millisecond, res).(*metricsEvent)
				res.Err = nil // Collector must not inspect this mutable result again.
				c.beforeInvokeHandler(before)
				c.afterInvokeHandler(event)
			}
			families, err := reg.Gather()
			if err != nil {
				t.Fatal(err)
			}
			values := map[string]float64{}
			for _, f := range families {
				for _, m := range f.Metric {
					if m.Counter != nil {
						values[f.GetName()] += m.Counter.GetValue()
					}
					if m.Gauge != nil {
						values[f.GetName()] += m.Gauge.GetValue()
					}
				}
			}
			side := constant.SideConsumer
			if role == common.PROVIDER {
				side = constant.SideProvider
			}
			prefix := "dubbo_" + side + "_requests_"
			for suffix, want := range map[string]float64{
				"failed_total": 2, "failed_codec_total": 1, "failed_network_total": 1,
				"failed_codec_total_aggregate": 1, "failed_network_total_aggregate": 1, "processing_total": 0,
			} {
				if values[prefix+suffix] != want {
					t.Errorf("%s: got %v want %v", prefix+suffix, values[prefix+suffix], want)
				}
			}
			if values[prefix+"unknown_failed_total"] != 0 {
				t.Fatal("double classification")
			}
		})
	}
}

func TestObservedOriginDoesNotChangeBusinessResult(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1/Fixture?registry.role=" + strconv.Itoa(common.CONSUMER))
	invoker := base.NewBaseInvoker(url)
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	cause := errors.New("existing wait error")
	res := &result.RPCResult{Err: cause}
	event := NewAfterInvokeEventWithOrigin(invoker, inv, time.Second, res, rpcerror.Codec).(*metricsEvent)
	if res.Err != cause || event.err != cause || classifyError(rpcerror.Mark(event.err, event.origin)) != ErrorTypeCodec {
		t.Fatal("sidecar evidence must not replace the business error")
	}
	success := NewAfterInvokeEventWithOrigin(invoker, inv, time.Second, &result.RPCResult{}, rpcerror.Codec).(*metricsEvent)
	if success.err != nil {
		t.Fatal("stale evidence must not turn success into failure")
	}
}

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

package metrics

import (
	"context"
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	metric "dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type streamInvoker struct {
	*base.BaseInvoker
	openErr error
	end     func(error)
}

func (i *streamInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	if i.openErr != nil {
		return &result.RPCResult{Err: i.openErr}
	}
	i.end = base.ClaimStreamWatches(ctx, inv)
	return &result.RPCResult{Rest: "stream"}
}

func newStreamTestInvocation(callType string) base.Invocation {
	inv := invocation.NewRPCInvocation("Chat", nil, nil)
	inv.SetAttribute(constant.CallTypeKey, callType)
	return inv
}

func TestFilterRecordsConsumerStreamWhenItEnds(t *testing.T) {
	ch := make(chan metric.MetricsEvent, 8)
	metric.Subscribe(constant.MetricsRpc, ch)
	defer metric.Unsubscribe(constant.MetricsRpc)
	url, _ := common.NewURL("tri://127.0.0.1/Fixture?registry.role=0")
	invoker := &streamInvoker{BaseInvoker: base.NewBaseInvoker(url)}

	res := (&metricsFilter{}).Invoke(context.Background(), invoker, newStreamTestInvocation(constant.CallBidiStream))
	require.NoError(t, res.Error())
	require.NotNil(t, invoker.end)
	require.Len(t, ch, 1, "the stream was recorded when it opened")

	invoker.end(errors.New("stream failed"))
	require.Len(t, ch, 2)
	invoker.end(nil)
	require.Len(t, ch, 2, "a stream is recorded once")
}

func TestFilterRecordsWithoutWaiting(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		callType string
		openErr  error
	}{
		{name: "consumer stream that fails to open", url: "tri://127.0.0.1/Fixture?registry.role=0",
			callType: constant.CallServerStream, openErr: errors.New("open failed")},
		{name: "consumer unary call", url: "tri://127.0.0.1/Fixture?registry.role=0", callType: constant.CallUnary},
		{name: "provider stream", url: "tri://127.0.0.1/Fixture?registry.role=3", callType: constant.CallBidiStream},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch := make(chan metric.MetricsEvent, 8)
			metric.Subscribe(constant.MetricsRpc, ch)
			defer metric.Unsubscribe(constant.MetricsRpc)
			url, _ := common.NewURL(test.url)
			invoker := &streamInvoker{BaseInvoker: base.NewBaseInvoker(url), openErr: test.openErr}

			(&metricsFilter{}).Invoke(context.Background(), invoker, newStreamTestInvocation(test.callType))
			require.Len(t, ch, 2)
			require.Nil(t, invoker.end, "nothing watches this call")
		})
	}
}

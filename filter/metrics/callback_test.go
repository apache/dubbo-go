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

package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	metric "dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type callbackInvoker struct {
	*base.BaseInvoker
	early       bool
	observation *rpcerror.Observation
}

func (i *callbackInvoker) Invoke(ctx context.Context, _ base.Invocation) result.Result {
	o := rpcerror.ObservationFromContext(ctx)
	i.observation = o
	o.BeginAsync(ctx, time.Minute)
	if i.early {
		o.Complete(errors.New("response"))
	}
	return &result.RPCResult{}
}
func TestFilterCallbackCompletion(t *testing.T) {
	for _, early := range []bool{false, true} {
		ch := make(chan metric.MetricsEvent, 8)
		metric.Subscribe(constant.MetricsRpc, ch)
		url, _ := common.NewURL("dubbo://127.0.0.1/Fixture?registry.role=0")
		i := &callbackInvoker{BaseInvoker: base.NewBaseInvoker(url), early: early}
		res := (&metricsFilter{}).Invoke(context.Background(), i, invocation.NewRPCInvocation("Call", nil, nil))
		if res.Error() != nil {
			t.Fatal("modified submission result")
		}
		if !early {
			if len(ch) != 1 {
				t.Fatal("premature completion")
			}
			i.observation.Complete(nil)
		}
		if len(ch) != 2 {
			t.Fatalf("expected before + final, got %d", len(ch))
		}
		i.observation.Complete(nil)
		if len(ch) != 2 {
			t.Fatal("duplicate publication")
		}
		metric.Unsubscribe(constant.MetricsRpc)
	}
}

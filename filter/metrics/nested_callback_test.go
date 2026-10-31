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
	"fmt"
	"strconv"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	metric "dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// A downstream reference may disable Metrics while retaining the incoming ctx.
// Its callback must not take over the parent invocation's completion or origin.
func TestNestedCallbackObservationOwnership(t *testing.T) {
	for _, role := range []int{common.PROVIDER, common.CONSUMER} {
		for _, childMetrics := range []bool{false, true} {
			for _, early := range []bool{false, true} {
				t.Run(fmt.Sprintf("role=%d/childMetrics=%v/early=%v", role, childMetrics, early), func(t *testing.T) {
					checkNestedCallbackOwnership(t, role, childMetrics, early)
				})
			}
		}
	}
}

// Each scenario owns its context, subscription and pending request lifecycle.
func checkNestedCallbackOwnership(t *testing.T, role int, childMetrics, early bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan metric.MetricsEvent, 8)
	metric.Subscribe(constant.MetricsRpc, ch)
	defer metric.Unsubscribe(constant.MetricsRpc)

	parentURL, err := common.NewURL("dubbo://127.0.0.1:20880/Parent?registry.role=" + strconv.Itoa(role))
	if err != nil {
		t.Fatal(err)
	}
	filterName := "-metrics"
	if childMetrics {
		filterName = constant.MetricsFilterKey
	}
	childURL, err := common.NewURL("dubbo://127.0.0.1:20881/Child?registry.role=0&reference.filter=" + filterName)
	if err != nil {
		t.Fatal(err)
	}
	cause := rpcerror.Mark(errors.New("downstream response codec failure"), rpcerror.Codec)
	transport := &nestedCallbackTransport{early: early, err: cause}
	exchange := remoting.NewExchangeClient(childURL, transport, time.Second, false)
	defer exchange.Close()
	child := protocolwrapper.BuildInvokerChain(dubbo.NewDubboInvoker(childURL, exchange), constant.ReferenceFilterKey)
	callbacks := make(chan struct{}, 2)
	var parentObservation *rpcerror.Observation
	parent := &nestedCallbackInvoker{
		BaseInvoker: base.NewBaseInvoker(parentURL),
		call: func(ctx context.Context, _ base.Invocation) result.Result {
			parentObservation = rpcerror.ObservationFromContext(ctx)
			inv := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("Child"), invocation.WithReply(new(string)),
				invocation.WithAttachment(constant.AsyncKey, "true"),
				invocation.WithCallBack(func(common.CallbackResponse) { callbacks <- struct{}{} }))
			if res := child.Invoke(ctx, inv); res.Error() != nil {
				t.Fatal(res.Error())
			}
			// The provider handles its own outcome, independently of the child.
			return &result.RPCResult{}
		},
	}
	res := (&metricsFilter{}).Invoke(ctx, parent, invocation.NewRPCInvocation("Parent", nil, nil))
	defer remoting.RemovePendingResponse(remoting.SequenceType(transport.id))
	if res.Error() != nil {
		t.Fatal("changed parent result", res.Error())
	}
	assertChildObservation(t, transport.pending.Observation, parentObservation, childMetrics)
	assertParentCompleted(t, len(ch), childMetrics, early)
	if !early {
		if !childMetrics {
			cancel() // Ending the parent ctx must not suppress the original child callback.
		}
		(&remoting.Response{ID: transport.id, Error: cause}).Handle()
	}
	if len(callbacks) != 1 {
		t.Fatalf("expected one user callback, got %d", len(callbacks))
	}
	want := 2
	if childMetrics {
		want = 4
	}
	if len(ch) != want {
		t.Fatalf("child changed parent accounting: got %d events, want %d", len(ch), want)
	}
}

func assertChildObservation(t *testing.T, child, parent *rpcerror.Observation, enabled bool) {
	t.Helper()
	if enabled {
		if child == nil || child == parent {
			t.Fatal("child did not get an independent observation")
		}
	} else if child != nil {
		t.Fatal("unmonitored child claimed the parent observation")
	}
}

func assertParentCompleted(t *testing.T, count int, childMetrics, early bool) {
	t.Helper()
	want := 2 // Parent BeforeInvoke and completed success, even if child is pending.
	if childMetrics {
		want++
		if early {
			want++
		}
	}
	if count != want {
		t.Fatalf("parent completion was deferred: got %d events, want %d", count, want)
	}
}

func TestProviderObservationCannotBeClaimedByRemoting(t *testing.T) {
	ch := make(chan metric.MetricsEvent, 4)
	metric.Subscribe(constant.MetricsRpc, ch)
	defer metric.Unsubscribe(constant.MetricsRpc)
	url, err := common.NewURL("dubbo://127.0.0.1:20880/Parent?registry.role=3")
	if err != nil {
		t.Fatal(err)
	}
	transport := &nestedCallbackTransport{early: true, err: errors.New("child failure")}
	exchange := remoting.NewExchangeClient(url, transport, time.Second, false)
	defer exchange.Close()
	parent := &nestedCallbackInvoker{
		BaseInvoker: base.NewBaseInvoker(url),
		call: func(ctx context.Context, inv base.Invocation) result.Result {
			// Even reusing the same invocation cannot hand provider monitoring to a client.
			err := exchange.AsyncRequestContext(ctx, &inv, url, time.Second, func(common.CallbackResponse) {}, &result.RPCResult{})
			if err != nil {
				t.Fatal(err)
			}
			return &result.RPCResult{}
		},
	}
	res := (&metricsFilter{}).Invoke(context.Background(), parent, invocation.NewRPCInvocation("Parent", nil, nil))
	defer remoting.RemovePendingResponse(remoting.SequenceType(transport.id))
	if res.Error() != nil || transport.pending.Observation != nil || len(ch) != 2 {
		t.Fatal("provider observation was claimed by an outgoing request")
	}
}

type nestedCallbackInvoker struct {
	*base.BaseInvoker
	call func(context.Context, base.Invocation) result.Result
}

func (i *nestedCallbackInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	return i.call(ctx, inv)
}

type nestedCallbackTransport struct {
	early   bool
	err     error
	id      int64
	pending *remoting.PendingResponse
}

func (*nestedCallbackTransport) SetExchangeClient(*remoting.ExchangeClient) {}
func (*nestedCallbackTransport) Connect(*common.URL) error                  { return nil }
func (*nestedCallbackTransport) Close()                                     {}
func (*nestedCallbackTransport) IsAvailable() bool                          { return true }
func (c *nestedCallbackTransport) Request(req *remoting.Request, _ time.Duration, p *remoting.PendingResponse) error {
	c.id, c.pending = req.ID, p
	p.Observation.BindSession(c)
	if c.early {
		(&remoting.Response{ID: req.ID, Error: c.err}).Handle()
	}
	return nil
}

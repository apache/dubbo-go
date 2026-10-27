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

package integration_test

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
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// Unlike the opt-in wire fixture, this deterministic transport avoids Getty's
// known teardown races while retaining real Exchange/Filter/Collector behavior.
func TestCallbackMetricsThroughExchange(t *testing.T) {
	reg, mf := initializeMetrics(t)
	url := fixtureURL(t, "127.0.0.1:1")
	transport := &controlledTransport{sent: make(chan int64, 4)}
	exchange := remoting.NewExchangeClient(url, transport, time.Second, false)
	invoker := dubbo.NewDubboInvoker(url, exchange)
	for _, early := range []bool{true, false} {
		method := "DelayedFailure"
		if early {
			method = "EarlyFailure"
		}
		cause := rpcerror.Mark(errors.New("codec fixture"), rpcerror.Codec)
		transport.earlyError = nil
		if early {
			transport.earlyError = cause
		}
		called := make(chan common.CallbackResponse, 1)
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(method), invocation.WithReply(new(string)),
			invocation.WithAttachment(constant.AsyncKey, "true"),
			invocation.WithCallBack(func(c common.CallbackResponse) { called <- c }))
		res := mf.Invoke(context.Background(), invoker, inv)
		id := <-transport.sent
		if res.Error() != nil {
			t.Fatal("changed submission result", res.Error())
		}
		if !early {
			await(t, func() bool { return metricValue(t, reg, "requests_processing_total", method) == 1 })
			if metricValue(t, reg, "requests_total", method) != 0 {
				t.Fatal("send counted as completed success")
			}
			(&remoting.Response{ID: id, Error: cause}).Handle()
		}
		select {
		case callback := <-called:
			// Existing callback contract has Cause=nil on this path.
			if callback.(remoting.AsyncCallbackResponse).Cause != nil {
				t.Fatal("changed user-visible callback Cause")
			}
		case <-time.After(time.Second):
			t.Fatal("callback was not delivered")
		}
		checkMetrics(t, reg, method, 1, 1, 1, 0)
		// A duplicate terminal has no waiter and must not count again.
		(&remoting.Response{ID: id, Error: cause}).Handle()
		checkMetrics(t, reg, method, 1, 1, 1, 0)
		if len(called) != 0 {
			t.Fatal("duplicate user callback")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport.earlyError = nil
	called := make(chan common.CallbackResponse, 1)
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("CancelledObservation"), invocation.WithReply(new(string)),
		invocation.WithAttachment(constant.AsyncKey, "true"),
		invocation.WithCallBack(func(c common.CallbackResponse) { called <- c }))
	if res := mf.Invoke(ctx, invoker, inv); res.Error() != nil {
		t.Fatal(res.Error())
	}
	id := <-transport.sent
	await(t, func() bool { return metricValue(t, reg, "requests_processing_total", "CancelledObservation") == 1 })
	cancel()
	await(t, func() bool { return metricValue(t, reg, "requests_processing_total", "CancelledObservation") == 0 })
	checkMetrics(t, reg, "CancelledObservation", 0, 0, 0, 0)
	if remoting.GetPendingResponse(remoting.SequenceType(id)) == nil {
		t.Fatal("monitoring suppressed late callback")
	}
	(&remoting.Response{ID: id, Result: &result.RPCResult{}}).Handle()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("late callback not delivered")
	}
	checkMetrics(t, reg, "CancelledObservation", 0, 0, 0, 0)
}

type controlledTransport struct {
	sent       chan int64
	earlyError error
}

func (*controlledTransport) SetExchangeClient(*remoting.ExchangeClient) {}
func (*controlledTransport) Connect(*common.URL) error                  { return nil }
func (*controlledTransport) Close()                                     {}
func (*controlledTransport) IsAvailable() bool                          { return true }
func (c *controlledTransport) Request(req *remoting.Request, timeout time.Duration, p *remoting.PendingResponse) error {
	return c.RequestContext(context.Background(), req, timeout, p)
}
func (c *controlledTransport) RequestContext(_ context.Context, req *remoting.Request, _ time.Duration, p *remoting.PendingResponse) error {
	p.Observation.BindSession(c)
	c.sent <- req.ID
	if c.earlyError != nil {
		(&remoting.Response{ID: req.ID, Error: c.earlyError}).Handle()
	}
	return nil
}

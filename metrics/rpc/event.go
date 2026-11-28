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

package rpc

import (
	"context"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/internal/rpcexemplar"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// metricsEvent is the event defined for rpc metrics
type metricsEvent struct {
	collector *rpcCollector // snapshot readiness; a call begun before Init stays unrecorded
	costTime  time.Duration
	role      string
	labels    map[string]string
	retry     bool // a consumer attempt that retries a failed attempt of the same call
	err       error
	exemplar  *rpcexemplar.IDs
	name      metricsName
	hasResult bool
	origin    rpcerror.Origin
	kind      rpcerror.Kind
}

// Type returns the type of the event, it is used for metrics bus to dispatch the event to rpc collector
func (m metricsEvent) Type() string {
	return constant.MetricsRpc
}

type metricsName uint8

const (
	BeforeInvoke metricsName = iota
	AfterInvoke
	ObservationExpired
)

func NewBeforeInvokeEvent(invoker base.Invoker, invocation base.Invocation) metrics.MetricsEvent {
	return NewBeforeInvokeEventWithContext(context.Background(), invoker, invocation)
}

// NewBeforeInvokeEventWithExemplar snapshots the local RPC span for completion.
func NewBeforeInvokeEventWithExemplar(ctx context.Context, invoker base.Invoker, invocation base.Invocation) metrics.MetricsEvent {
	event := newBeforeInvokeEvent(ctx, invoker, invocation)
	event.exemplar = rpcexemplar.ForInvocation(ctx, invocation, event.role)
	return event
}

// NewBeforeInvokeEventWithContext also records whether a cluster invoker
// marked this attempt of invocation as a retry in ctx.
func NewBeforeInvokeEventWithContext(ctx context.Context, invoker base.Invoker, invocation base.Invocation) metrics.MetricsEvent {
	return newBeforeInvokeEvent(ctx, invoker, invocation)
}

func newBeforeInvokeEvent(ctx context.Context, invoker base.Invoker, invocation base.Invocation) *metricsEvent {
	return &metricsEvent{
		collector: activeCollector.Load(),
		name:      BeforeInvoke,
		role:      getRole(invoker.GetURL()),
		labels:    buildLabels(invoker.GetURL(), invocation),
		retry:     base.IsRetryAttempt(ctx, invocation),
	}
}

// NewCompletionEvent uses the identity captured before sending, so callbacks do
// not retain or read a mutable invocation, result, URL, or session.
func NewCompletionEvent(before metrics.MetricsEvent, duration time.Duration, c rpcerror.Completion) metrics.MetricsEvent {
	start := before.(*metricsEvent)
	event := &metricsEvent{collector: start.collector, costTime: duration, role: start.role, labels: start.labels, retry: start.retry, err: c.Err, hasResult: c.Known, origin: c.Origin, kind: c.Kind}
	if c.Known {
		event.name = AfterInvoke
		event.exemplar = start.exemplar
	} else {
		event.name = ObservationExpired
	}
	return event
}

func NewAfterInvokeEvent(invoker base.Invoker, invocation base.Invocation, costTime time.Duration, result result.Result) metrics.MetricsEvent {
	return NewAfterInvokeEventWithOrigin(invoker, invocation, costTime, result, rpcerror.Unknown)
}

// NewAfterInvokeEventWithOrigin snapshots local request evidence independently
// of the error returned to the user. Success always remains success.
func NewAfterInvokeEventWithOrigin(invoker base.Invoker, invocation base.Invocation, costTime time.Duration, result result.Result, origin rpcerror.Origin) metrics.MetricsEvent {
	var err error
	if result != nil {
		err = result.Error()
	}
	return &metricsEvent{
		collector: activeCollector.Load(),
		name:      AfterInvoke,
		costTime:  costTime,
		role:      getRole(invoker.GetURL()),
		labels:    buildLabels(invoker.GetURL(), invocation),
		err:       err,
		hasResult: result != nil,
		origin:    origin,
	}
}

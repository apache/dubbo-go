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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// metricsEvent is the event defined for rpc metrics
type metricsEvent struct {
	name      metricsName
	costTime  time.Duration
	role      string
	labels    map[string]string
	err       error
	hasResult bool
	origin    rpcerror.Origin
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
	return &metricsEvent{
		name:   BeforeInvoke,
		role:   getRole(invoker.GetURL()),
		labels: buildLabels(invoker.GetURL(), invocation),
	}
}

// NewCompletionEvent uses the identity captured before sending, so callbacks do
// not retain or read a mutable invocation, result, URL, or session.
func NewCompletionEvent(before metrics.MetricsEvent, duration time.Duration, c rpcerror.Completion) metrics.MetricsEvent {
	start := before.(*metricsEvent)
	name := AfterInvoke
	if !c.Known {
		name = ObservationExpired
	}
	return &metricsEvent{name: name, costTime: duration, role: start.role, labels: start.labels, err: c.Err, hasResult: c.Known, origin: c.Origin}
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
		name:      AfterInvoke,
		costTime:  costTime,
		role:      getRole(invoker.GetURL()),
		labels:    buildLabels(invoker.GetURL(), invocation),
		err:       err,
		hasResult: result != nil,
		origin:    origin,
	}
}

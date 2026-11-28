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
	"strconv"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/internal/rpcexemplar"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/metrics/rpc"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func init() {
	extension.SetFilter(constant.MetricsFilterKey, newFilter)
}

// metricsFilter will report RPC metrics to the metrics bus and implements the filter.Filter interface
type metricsFilter struct {
	captureExemplars bool
}

// Invoke publish the BeforeInvokeEvent and AfterInvokeEvent to metrics bus
func (mf *metricsFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	var observation *rpcerror.Observation
	if invoker.GetURL().GetParam(constant.RegistryRoleKey, "") == strconv.Itoa(common.CONSUMER) {
		ctx, observation = rpcerror.WithInvocationObservation(ctx, invocation)
	} else {
		ctx, observation = rpcerror.WithObservation(ctx)
	}
	var before metrics.MetricsEvent
	if mf.captureExemplars {
		before = rpc.NewBeforeInvokeEventWithExemplar(ctx, invoker, invocation)
	} else {
		before = rpc.NewBeforeInvokeEventWithContext(ctx, invoker, invocation)
	}
	metrics.Publish(before)
	start := time.Now()
	observation.OnComplete(func(c rpcerror.Completion) {
		metrics.Publish(rpc.NewCompletionEvent(before, c.At.Sub(start), c))
	})
	res := invoker.Invoke(ctx, invocation)
	var err error
	if res != nil {
		err = res.Error()
	}
	observation.FinishInvoke(err, res != nil)
	return res
}

// OnResponse do nothing and return the result
func (mf *metricsFilter) OnResponse(ctx context.Context, res result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	return res
}

func newFilter() filter.Filter {
	return &metricsFilter{captureExemplars: rpcexemplar.Enabled()}
}

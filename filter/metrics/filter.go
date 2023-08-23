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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/metrics/rpc"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// must initialize before using the filter and after loading configuration
var metricFilterInstance *metricsFilter

func init() {
	extension.SetFilter(constant.MetricsFilterKey, newFilter)
}

// metricsFilter will report RPC metrics to the metrics bus and implements the filter.Filter interface
type metricsFilter struct{}

// Invoke publish the BeforeInvokeEvent and AfterInvokeEvent to metrics bus
func (mf *metricsFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	metrics.Publish(rpc.NewBeforeInvokeEvent(invoker, invocation))
	start := time.Now()
	res := invoker.Invoke(ctx, invocation)
	end := time.Now()
	duration := end.Sub(start)
	metrics.Publish(rpc.NewAfterInvokeEvent(invoker, invocation, duration, res))
	return res
}

// OnResponse do nothing and return the result
func (mf *metricsFilter) OnResponse(ctx context.Context, res protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return res
}

// newFilter creates a new metricsFilter instance.
//
// It's lazy initialization,
// and make sure that the configuration had been loaded before invoking this method.
func newFilter() filter.Filter {
	if metricFilterInstance == nil {
		metricFilterInstance = &metricsFilter{}
	}
	return metricFilterInstance
}

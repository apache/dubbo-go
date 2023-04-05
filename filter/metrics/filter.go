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

// Package metrics provides metrics collection filter.
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
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// must initialized before using the filter and after loading configuration
var metricFilterInstance *Filter

func init() {
	extension.SetFilter(constant.MetricsFilterKey, newFilter)
}

// Filter will calculate the invocation's duration and the report to the reporters
// more info please take a look at dubbo-samples projects
type Filter struct {
	reporters []metrics.Reporter
}

// Invoke collect the duration of invocation and then report the duration by using goroutine
func (p *Filter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	start := time.Now()
	res := invoker.Invoke(ctx, invocation)
	end := time.Now()
	duration := end.Sub(start)
	go func() {
		for _, reporter := range p.reporters {
			reporter.Report(ctx, invoker, invocation, duration, res)
		}
	}()
	return res
}

// OnResponse do nothing and return the result
func (p *Filter) OnResponse(ctx context.Context, res protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return res
}

// newFilter the Filter is singleton.
// it's lazy initialization
// make sure that the configuration had been loaded before invoking this method.
func newFilter() filter.Filter {
	if metricFilterInstance == nil {
		reporters := make([]metrics.Reporter, 0, 1)
		reporters = append(reporters, extension.GetMetricReporter("prometheus", metrics.NewReporterConfig()))
		metricFilterInstance = &Filter{
			reporters: reporters,
		}
	}
	return metricFilterInstance
}

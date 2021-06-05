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

package filter_impl

import (
	"context"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	metricFilterName = "metrics"
)

var metricFilterInstance filter.Filter

// must initialized before using the filter and after loading configuration
func init() {
	extension.SetFilter(metricFilterName, newMetricsFilter)
}

// metricFilter will calculate the invocation's duration and the report to the reporters
// If you want to use this filter to collect the metrics,
// Adding this into your configuration file, like:
// filter: "metrics"
// metrics:
//   reporter:
//     - "your reporter" # here you should specify the reporter, for example 'prometheus'
// more info please take a look at dubbo-samples projects
type metricsFilter struct {
	reporters []metrics.Reporter
}

// Invoke collect the duration of invocation and then report the duration by using goroutine
func (p *metricsFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
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
func (p *metricsFilter) OnResponse(ctx context.Context, res protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return res
}

// newMetricsFilter the metricsFilter is singleton.
// it's lazy initialization
// make sure that the configuration had been loaded before invoking this method.
func newMetricsFilter() filter.Filter {
	if metricFilterInstance == nil {
		reporterNames := config.GetMetricConfig().Reporters
		reporters := make([]metrics.Reporter, 0, len(reporterNames))
		for _, name := range reporterNames {
			reporters = append(reporters, extension.GetMetricReporter(name))
		}
		metricFilterInstance = &metricsFilter{
			reporters: reporters,
		}
	}

	return metricFilterInstance
}

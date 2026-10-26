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
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

var (
	providerRTHistogram = metrics.NewMetricKey("dubbo_provider_rt_milliseconds_histogram_seconds", "Provider RPC latency in seconds")
	consumerRTHistogram = metrics.NewMetricKey("dubbo_consumer_rt_milliseconds_histogram_seconds", "Consumer RPC latency in seconds")
)

func rpcHistogramOpts(raw string) *metrics.HistogramOpts {
	defaults := &metrics.HistogramOpts{Buckets: []float64{0.1, 0.3, 0.5, 1, 3, 5, 10}}
	if strings.TrimSpace(raw) == "" {
		return defaults
	}
	opts := &metrics.HistogramOpts{}
	for part := range strings.SplitSeq(raw, ",") {
		bound, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			logger.Warnf("[Metrics][RPC] invalid histogram buckets %q: %v; using default RPC latency buckets in seconds", raw, err)
			return defaults
		}
		opts.Buckets = append(opts.Buckets, bound)
	}
	if err := opts.Validate(); err != nil {
		logger.Warnf("[Metrics][RPC] invalid histogram buckets %q: %v; using default RPC latency buckets in seconds", raw, err)
		return defaults
	}
	return opts
}

func (c *rpcCollector) reportRTHistogram(role string, labels map[string]string, seconds float64) {
	if c.histogramRegistry == nil {
		return
	}
	var key *metrics.MetricKey
	switch role {
	case constant.SideProvider:
		key = providerRTHistogram
	case constant.SideConsumer:
		key = consumerRTHistogram
	default:
		return
	}
	c.histogramRegistry.HistogramWithOptions(metrics.NewMetricIdByLabels(key, labels), c.histogramOpts).Observe(seconds)
}

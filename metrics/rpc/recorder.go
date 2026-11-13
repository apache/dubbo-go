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
	"maps"
	"sync"
	"sync/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"
)

// A struct key preserves every existing label without delimiter collisions or
// JSON encoding on each measurement. Labels are captured before RPC execution.
type seriesKey struct {
	role, application, applicationVersion, hostname, ip, service, method, group, version string
}

func keyOf(e *metricsEvent) seriesKey {
	l := e.labels
	return seriesKey{e.role, l[constant.TagApplicationName], l[constant.TagApplicationVersion],
		l[constant.TagHostname], l[constant.TagIp], l[constant.TagInterface], l[constant.TagMethod],
		l[constant.TagGroup], l[constant.TagVersion]}
}

type counterPair struct {
	counter metrics.CounterMetric
	window  *aggregate.TimeWindowCounter
}

func (p *counterPair) inc() { p.counter.Inc(); p.window.Inc() }

type completionMetrics struct {
	completed    *counterPair
	rt, rtWindow metrics.ObservableMetric
	quantiles    *aggregate.TimeWindowQuantile
}

// Each cached series references the real registered metrics, not a second copy.
// Only the first record of each result/category binds its counter and window.
type rpcSeries struct {
	registry    metrics.MetricRegistry
	gauges      metrics.GaugeFuncRegistry
	definitions map[string]*metrics.MetricKey
	prefix      string
	labels      map[string]string
	processing  metrics.GaugeMetric
	qps         *aggregate.TimeWindowCounter
	mu          sync.Mutex
	completion  atomic.Pointer[completionMetrics]
	succeeded   atomic.Pointer[counterPair]
	failed      atomic.Pointer[counterPair]
	failures    [7]atomic.Pointer[counterPair]
}

func (c *rpcCollector) getSeries(e *metricsEvent) *rpcSeries {
	key := keyOf(e)
	if v, ok := c.series.Load(key); ok {
		return v.(*rpcSeries)
	}
	c.bindMu.Lock()
	defer c.bindMu.Unlock()
	if v, ok := c.series.Load(key); ok {
		return v.(*rpcSeries)
	}
	common := &c.metricSet.consumer.rpcCommonMetrics
	if e.role == constant.SideProvider {
		common = &c.metricSet.provider.rpcCommonMetrics
	}
	s := &rpcSeries{registry: c.registry, gauges: c.registry.(metrics.GaugeFuncRegistry),
		definitions: common.keys, prefix: "dubbo_" + e.role + "_", labels: maps.Clone(e.labels),
		qps: aggregate.NewTimeWindowCounter(metrics.DefaultBucketNum, metrics.DefaultTimeWindowSeconds)}
	s.processing = s.registry.Gauge(s.id("requests_processing_total"))
	s.gauges.GaugeFunc(s.id("qps_total"), s.qps.QPS)
	// Publish only a fully bound series. On initialization failure, later calls
	// may retry; the error is isolated by the collector's record boundary.
	c.series.Store(key, s)
	return s
}

func (s *rpcSeries) id(suffix string) *metrics.MetricId {
	return metrics.NewMetricIdByLabels(s.definitions[s.prefix+suffix], s.labels)
}
func (s *rpcSeries) newCounter(suffix string) *counterPair {
	p := &counterPair{counter: s.registry.Counter(s.id(suffix)), window: aggregate.NewTimeWindowCounter(metrics.DefaultBucketNum, metrics.DefaultTimeWindowSeconds)}
	s.gauges.GaugeFunc(s.id(suffix+"_aggregate"), p.window.Count)
	return p
}
func (s *rpcSeries) counter(slot *atomic.Pointer[counterPair], suffix string) *counterPair {
	if p := slot.Load(); p != nil {
		return p
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := slot.Load(); p != nil {
		return p
	}
	p := s.newCounter(suffix)
	slot.Store(p)
	return p
}
func (s *rpcSeries) completed() *completionMetrics {
	if m := s.completion.Load(); m != nil {
		return m
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := s.completion.Load(); m != nil {
		return m
	}
	m := &completionMetrics{completed: s.newCounter("requests_total"),
		rt:        s.registry.Rt(s.id("rt_milliseconds"), &metrics.RtOpts{}),
		rtWindow:  s.registry.Rt(s.id("rt"), &metrics.RtOpts{Aggregate: true, BucketNum: metrics.DefaultBucketNum, TimeWindowSeconds: metrics.DefaultTimeWindowSeconds}),
		quantiles: aggregate.NewTimeWindowQuantile(metrics.DefaultCompression, metrics.DefaultBucketNum, metrics.DefaultTimeWindowSeconds)}
	for i, q := range []float64{.5, .9, .95, .99} {
		suffix := []string{"rt_milliseconds_p50", "rt_milliseconds_p90", "rt_milliseconds_p95", "rt_milliseconds_p99"}[i]
		s.gauges.GaugeFunc(s.id(suffix), func() float64 { return m.quantiles.Quantile(q) })
	}
	s.completion.Store(m)
	return m
}

var failureSuffixes = [...]string{"requests_unknown_failed_total", "requests_timeout_total",
	"requests_limit_total", "requests_failed_service_unavailable_total", "requests_business_failed_total",
	"requests_failed_network_total", "requests_failed_codec_total"}

func (s *rpcSeries) record(e *metricsEvent) {
	switch e.name {
	case BeforeInvoke:
		s.qps.Inc()
		s.processing.Inc()
	case ObservationExpired:
		s.processing.Dec()
	case AfterInvoke:
		m := s.completed()
		m.completed.inc()
		s.processing.Dec()
		if e.hasResult {
			if e.err == nil {
				s.counter(&s.succeeded, "requests_succeed_total").inc()
			} else {
				s.counter(&s.failed, "requests_failed_total").inc()
				kind := classifyError(rpcerror.Mark(e.err, e.origin))
				s.counter(&s.failures[kind], failureSuffixes[kind]).inc()
			}
		}
		cost := float64(e.costTime.Milliseconds())
		m.rt.Observe(cost)
		m.rtWindow.Observe(cost)
		m.quantiles.Add(cost)
	}
}

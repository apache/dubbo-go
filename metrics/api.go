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

var registries = make(map[string]func(*ReporterConfig) MetricRegistry)
var collectors = make([]CollectorFunc, 0)
var registry MetricRegistry

// CollectorFunc used to extend more indicators
type CollectorFunc func(MetricRegistry, *ReporterConfig)

const defaultRegistry = "prometheus"
// Init Metrics module
func Init(config *ReporterConfig) {
	regFunc, ok := registries[config.Protocol]
	if !ok {
		regFunc = registries[defaultRegistry] // default
	}
	registry = regFunc(config)
	for _, co := range collectors {
		co(registry, config)
	}
	registry.Export()
}

// SetRegistry extend more MetricRegistry, default PrometheusRegistry
func SetRegistry(name string, v func(*ReporterConfig) MetricRegistry) {
	registries[name] = v
}

// AddCollector add more indicators, like  metadata、sla、configcenter、metadata etc
func AddCollector(name string, fun func(MetricRegistry, *ReporterConfig)) {
	collectors = append(collectors, fun)
}

// MetricRegistry data container，data compute、expose、agg
type MetricRegistry interface {
	Counter(*MetricId) CounterMetric     // add or update a counter
	Gauge(*MetricId) GaugeMetric         // add or update a gauge
	Histogram(*MetricId) HistogramMetric // add a metric num to a histogram
	Summary(*MetricId) SummaryMetric     // add a metric num to a summary
	Export()                             // expose metric data， such as Prometheus http exporter
	// GetMetrics() []*MetricSample // get all metric data
	// GetMetricsString() (string, error) // get text format metric data
}

// multi registry，like micrometer CompositeMeterRegistry
// type CompositeRegistry struct {
// 	rs []MetricRegistry
// }

// Type metric type, save with micrometer
type Type uint8

const (
	Counter Type = iota
	Gauge
	LongTaskTimer
	Timer
	DistributionSummary
	Other
)

// MetricId
// # HELP dubbo_metadata_store_provider_succeed_total Succeed Store Provider Metadata
// # TYPE dubbo_metadata_store_provider_succeed_total gauge
// dubbo_metadata_store_provider_succeed_total{application_name="provider",hostname="localhost",interface="org.example.DemoService",ip="10.252.156.213",} 1.0
// other properties except value
type MetricId struct {
	Name string
	Desc string
	Tags map[string]string
	Type Type
}

func (m *MetricId) TagKeys() []string {
	keys := make([]string, 0, len(m.Tags))
	for k := range m.Tags {
		keys = append(keys, k)
	}
	return keys
}

func NewMetricId(key *MetricKey, level MetricLevel) *MetricId {
	return &MetricId{Name: key.Name, Desc: key.Desc, Tags: level.Tags()}
}

// MetricSample a metric sample，This is the final data presentation, 
// not an intermediate result(like summary，histogram they will export to a set of MetricSample)
type MetricSample struct {
	*MetricId
	value float64
}

// CounterMetric counter metric
type CounterMetric interface {
	Inc()
	Add(float64)
}

// GaugeMetric gauge metric
type GaugeMetric interface {
	Set(float64)
	// Inc()
	// Dec()
	// Add(float64)
	// Sub(float64)
}

// HistogramMetric histogram metric
type HistogramMetric interface {
	Record(float64)
}

// SummaryMetric summary metric
type SummaryMetric interface {
	Record(float64)
}

// StatesMetrics multi metrics，include total,success num, fail num，call MetricsRegistry save data
type StatesMetrics interface {
	Success()
	AddSuccess(float64)
	Fail()
	AddFailed(float64)
	Inc(succ bool)
}

func NewStatesMetrics(total func() *MetricId, succ func() *MetricId, fail func() *MetricId, reg MetricRegistry) StatesMetrics {
	return &DefaultStatesMetric{total: total, succ: succ, fail: fail, r: reg}
}

type DefaultStatesMetric struct {
	r     MetricRegistry
	total func() *MetricId
	succ  func() *MetricId
	fail  func() *MetricId
}

func (c DefaultStatesMetric) Inc(succ bool) {
	if succ {
		c.Success()
	} else {
		c.Fail()
	}
}
func (c DefaultStatesMetric) Success() {
	c.r.Counter(c.total()).Inc()
	c.r.Counter(c.succ()).Inc()
}

func (c DefaultStatesMetric) AddSuccess(v float64) {
	c.r.Counter(c.total()).Add(v)
	c.r.Counter(c.succ()).Add(v)
}

func (c DefaultStatesMetric) Fail() {
	c.r.Counter(c.total()).Inc()
	c.r.Counter(c.fail()).Inc()
}

func (c DefaultStatesMetric) AddFailed(v float64) {
	c.r.Counter(c.total()).Add(v)
	c.r.Counter(c.fail()).Add(v)
}

// TimeMetrics muliti metrics, include min(Gauge)、max(Gauge)、avg(Gauge)、sum(Gauge)、last(Gauge)，call MetricRegistry to expose
// see dubbo-java org.apache.dubbo.metrics.aggregate.TimeWindowAggregator 
type TimeMetrics interface {
	Record(float64)
}

// NewTimeMetrics init and write all data to registry
func NewTimeMetrics(name string, l MetricLevel) {
	
}

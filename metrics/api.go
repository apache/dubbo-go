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
	"encoding/json"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"
)

var (
	registries = make(map[string]func(*ReporterConfig) MetricRegistry)
	collectors = make([]CollectorFunc, 0)
	registry   MetricRegistry
)

// CollectorFunc used to extend more indicators
type CollectorFunc func(MetricRegistry, *ReporterConfig)

// Init Metrics module
func Init(config *ReporterConfig) {
	if config.Enable {
		// default protocol is already set in metricConfig
		regFunc, ok := registries[config.Protocol]
		if ok {
			registry = regFunc(config)
			for _, co := range collectors {
				co(registry, config)
			}
			registry.Export()
		}
	}
}

// SetRegistry extend more MetricRegistry, default PrometheusRegistry
func SetRegistry(name string, v func(*ReporterConfig) MetricRegistry) {
	registries[name] = v
}

// AddCollector add more indicators, like metadata, sla, config-center etc.
func AddCollector(name string, fun CollectorFunc) {
	collectors = append(collectors, fun)
}

// MetricRegistry data container，data compute、expose、agg
type MetricRegistry interface {
	Counter(*MetricId) CounterMetric        // add or update a counter
	Gauge(*MetricId) GaugeMetric            // add or update a gauge
	Histogram(*MetricId) ObservableMetric   // add a metric num to a histogram
	Summary(*MetricId) ObservableMetric     // add a metric num to a summary
	Rt(*MetricId, *RtOpts) ObservableMetric // add a metric num to a rt
	Export()                                // expose metric data， such as Prometheus http exporter
	// GetMetrics() []*MetricSample // get all metric data
	// GetMetricsString() (string, error) // get text format metric data
}

type RtOpts struct {
	Aggregate         bool
	BucketNum         int   // only for aggRt
	TimeWindowSeconds int64 // only for aggRt
}

// multi registry，like micrometer CompositeMeterRegistry
// type CompositeRegistry struct {
// 	rs []MetricRegistry
// }

// Type metric type, save with micrometer
type Type uint8 // TODO check if Type is is useful

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
	Tags map[string]string // also named label
	Type Type              // TODO check if this field is useful
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

func NewMetricIdByLabels(key *MetricKey, labels map[string]string) *MetricId {
	return &MetricId{Name: key.Name, Desc: key.Desc, Tags: labels}
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
	Inc()
	Dec()
	Add(float64)
	Sub(float64)
}

// histogram summary rt metric
type ObservableMetric interface {
	Observe(float64)
}

type BaseCollector struct {
	R MetricRegistry
}

func (c *BaseCollector) StateCount(total, succ, fail *MetricKey, level MetricLevel, succed bool) {
	c.R.Counter(NewMetricId(total, level)).Inc()
	if succed {
		c.R.Counter(NewMetricId(succ, level)).Inc()
	} else {
		c.R.Counter(NewMetricId(fail, level)).Inc()
	}
}

type CounterVec interface {
	Inc(labels map[string]string)
	Add(labels map[string]string, v float64)
}

func NewCounterVec(metricKey *MetricKey, metricRegistry MetricRegistry) CounterVec {
	return &DefaultCounterVec{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
	}
}

type DefaultCounterVec struct {
	metricRegistry MetricRegistry
	metricKey      *MetricKey
}

func (d *DefaultCounterVec) Inc(labels map[string]string) {
	d.metricRegistry.Counter(NewMetricIdByLabels(d.metricKey, labels)).Inc()
}

func (d *DefaultCounterVec) Add(labels map[string]string, v float64) {
	d.metricRegistry.Counter(NewMetricIdByLabels(d.metricKey, labels)).Add(v)
}

type GaugeVec interface {
	Set(labels map[string]string, v float64)
	Inc(labels map[string]string)
	Dec(labels map[string]string)
	Add(labels map[string]string, v float64)
	Sub(labels map[string]string, v float64)
}

func NewGaugeVec(metricKey *MetricKey, metricRegistry MetricRegistry) GaugeVec {
	return &DefaultGaugeVec{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
	}
}

type DefaultGaugeVec struct {
	metricRegistry MetricRegistry
	metricKey      *MetricKey
}

func (d *DefaultGaugeVec) Set(labels map[string]string, v float64) {
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Set(v)
}

func (d *DefaultGaugeVec) Inc(labels map[string]string) {
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Inc()
}

func (d *DefaultGaugeVec) Dec(labels map[string]string) {
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Dec()
}

func (d *DefaultGaugeVec) Add(labels map[string]string, v float64) {
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Add(v)
}

func (d *DefaultGaugeVec) Sub(labels map[string]string, v float64) {
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Sub(v)
}

func labelsToString(labels map[string]string) string {
	labelsJson, err := json.Marshal(labels)
	if err != nil {
		logger.Errorf("json.Marshal(labels) = error:%v", err)
		return ""
	}
	return string(labelsJson)
}

type QpsMetric interface {
	Record(labels map[string]string)
}

func NewQpsMetric(metricKey *MetricKey, metricRegistry MetricRegistry) QpsMetric {
	return &DefaultQpsMetric{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
		mux:            sync.RWMutex{},
		cache:          make(map[string]*aggregate.TimeWindowCounter),
	}
}

type DefaultQpsMetric struct {
	metricRegistry MetricRegistry
	metricKey      *MetricKey
	mux            sync.RWMutex
	cache          map[string]*aggregate.TimeWindowCounter // key: metrics labels, value: TimeWindowCounter
}

func (d *DefaultQpsMetric) Record(labels map[string]string) {
	key := labelsToString(labels)
	if key == "" {
		return
	}
	d.mux.RLock()
	twc, ok := d.cache[key]
	d.mux.RUnlock()
	if !ok {
		d.mux.Lock()
		twc, ok = d.cache[key]
		if !ok {
			twc = aggregate.NewTimeWindowCounter(defaultBucketNum, defaultTimeWindowSeconds)
			d.cache[key] = twc
		}
		d.mux.Unlock()
	}
	twc.Inc()
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Set(twc.Count() / float64(twc.LivedSeconds()))
}

type AggregateCounterMetric interface {
	Inc(labels map[string]string)
}

func NewAggregateCounterMetric(metricKey *MetricKey, metricRegistry MetricRegistry) AggregateCounterMetric {
	return &DefaultAggregateCounterMetric{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
		mux:            sync.RWMutex{},
		cache:          make(map[string]*aggregate.TimeWindowCounter),
	}
}

type DefaultAggregateCounterMetric struct {
	metricRegistry MetricRegistry
	metricKey      *MetricKey
	mux            sync.RWMutex
	cache          map[string]*aggregate.TimeWindowCounter // key: metrics labels, value: TimeWindowCounter
}

func (d *DefaultAggregateCounterMetric) Inc(labels map[string]string) {
	key := labelsToString(labels)
	if key == "" {
		return
	}
	d.mux.RLock()
	twc, ok := d.cache[key]
	d.mux.RUnlock()
	if !ok {
		d.mux.Lock()
		twc, ok = d.cache[key]
		if !ok {
			twc = aggregate.NewTimeWindowCounter(defaultBucketNum, defaultTimeWindowSeconds)
			d.cache[key] = twc
		}
		d.mux.Unlock()
	}
	twc.Inc()
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Set(twc.Count())
}

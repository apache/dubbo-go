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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"
)

const (
	DefaultCompression       = 100
	DefaultBucketNum         = 10
	DefaultTimeWindowSeconds = 120
)

var (
	registries = make(map[string]func(*common.URL) MetricRegistry)
	collectors = make([]CollectorFunc, 0)
	registry   MetricRegistry
	once       sync.Once
)

// CollectorFunc used to extend more indicators
type CollectorFunc func(MetricRegistry, *common.URL)

// Init Metrics module
func Init(url *common.URL) {
	once.Do(func() {
		InitAppInfo(url.GetParam(constant.ApplicationKey, ""), url.GetParam(constant.AppVersionKey, ""))
		// default protocol is already set in metricConfig
		regFunc, ok := registries[url.Protocol]
		if ok {
			registry = regFunc(url)
			for _, co := range collectors {
				co(registry, url)
			}
			registry.Export()
		}
	})
}

// SetRegistry extend more MetricRegistry, default PrometheusRegistry
func SetRegistry(name string, v func(*common.URL) MetricRegistry) {
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

// Type metric type, same with micrometer
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

// NewMetricIdByLabels create a MetricId by key and labels
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

// CounterVec means a set of counters with the same metricKey but different labels
type CounterVec interface {
	Inc(labels map[string]string)
	Add(labels map[string]string, v float64)
}

// NewCounterVec create a CounterVec default implementation.
func NewCounterVec(metricRegistry MetricRegistry, metricKey *MetricKey) CounterVec {
	return &DefaultCounterVec{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
	}
}

// DefaultCounterVec is a default CounterVec implementation.
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

// GaugeVec means a set of gauges with the same metricKey but different labels
type GaugeVec interface {
	Set(labels map[string]string, v float64)
	Inc(labels map[string]string)
	Dec(labels map[string]string)
	Add(labels map[string]string, v float64)
	Sub(labels map[string]string, v float64)
}

// NewGaugeVec create a GaugeVec default implementation.
func NewGaugeVec(metricRegistry MetricRegistry, metricKey *MetricKey) GaugeVec {
	return &DefaultGaugeVec{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
	}
}

// DefaultGaugeVec is a default GaugeVec implementation.
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

// RtVec means a set of rt metrics with the same metricKey but different labels
type RtVec interface {
	Record(labels map[string]string, v float64)
}

// NewRtVec create a RtVec default implementation DefaultRtVec.
func NewRtVec(metricRegistry MetricRegistry, metricKey *MetricKey, rtOpts *RtOpts) RtVec {
	return &DefaultRtVec{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
		rtOpts:         rtOpts,
	}
}

// DefaultRtVec is a default RtVec implementation.
//
// If rtOpts.Aggregate is true, it will use the aggregate.TimeWindowAggregator with local aggregation,
// else it will use the aggregate.Result without aggregation.
type DefaultRtVec struct {
	metricRegistry MetricRegistry
	metricKey      *MetricKey
	rtOpts         *RtOpts
}

func (d *DefaultRtVec) Record(labels map[string]string, v float64) {
	d.metricRegistry.Rt(NewMetricIdByLabels(d.metricKey, labels), d.rtOpts).Observe(v)
}

// labelsToString convert @labels to json format string for cache key
func labelsToString(labels map[string]string) string {
	labelsJson, err := json.Marshal(labels)
	if err != nil {
		logger.Errorf("json.Marshal(labels) = error:%v", err)
		return ""
	}
	return string(labelsJson)
}

// QpsMetricVec means a set of qps metrics with the same metricKey but different labels.
type QpsMetricVec interface {
	Record(labels map[string]string)
}

func NewQpsMetricVec(metricRegistry MetricRegistry, metricKey *MetricKey) QpsMetricVec {
	return &DefaultQpsMetricVec{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
		mux:            sync.RWMutex{},
		cache:          make(map[string]*aggregate.TimeWindowCounter),
	}
}

// DefaultQpsMetricVec is a default QpsMetricVec implementation.
//
// It is concurrent safe, and it uses the aggregate.TimeWindowCounter to store and calculate the qps metrics.
type DefaultQpsMetricVec struct {
	metricRegistry MetricRegistry
	metricKey      *MetricKey
	mux            sync.RWMutex
	cache          map[string]*aggregate.TimeWindowCounter // key: metrics labels, value: TimeWindowCounter
}

func (d *DefaultQpsMetricVec) Record(labels map[string]string) {
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
			twc = aggregate.NewTimeWindowCounter(DefaultBucketNum, DefaultTimeWindowSeconds)
			d.cache[key] = twc
		}
		d.mux.Unlock()
	}
	twc.Inc()
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Set(twc.Count() / float64(twc.LivedSeconds()))
}

// AggregateCounterVec means a set of aggregate counter metrics with the same metricKey but different labels.
type AggregateCounterVec interface {
	Inc(labels map[string]string)
}

func NewAggregateCounterVec(metricRegistry MetricRegistry, metricKey *MetricKey) AggregateCounterVec {
	return &DefaultAggregateCounterVec{
		metricRegistry: metricRegistry,
		metricKey:      metricKey,
		mux:            sync.RWMutex{},
		cache:          make(map[string]*aggregate.TimeWindowCounter),
	}
}

// DefaultAggregateCounterVec is a default AggregateCounterVec implementation.
//
// It is concurrent safe, and it uses the aggregate.TimeWindowCounter to store and calculate the aggregate counter metrics.
type DefaultAggregateCounterVec struct {
	metricRegistry MetricRegistry
	metricKey      *MetricKey
	mux            sync.RWMutex
	cache          map[string]*aggregate.TimeWindowCounter // key: metrics labels, value: TimeWindowCounter
}

func (d *DefaultAggregateCounterVec) Inc(labels map[string]string) {
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
			twc = aggregate.NewTimeWindowCounter(DefaultBucketNum, DefaultTimeWindowSeconds)
			d.cache[key] = twc
		}
		d.mux.Unlock()
	}
	twc.Inc()
	d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKey, labels)).Set(twc.Count())
}

// QuantileMetricVec means a set of quantile metrics with the same metricKey but different labels.
type QuantileMetricVec interface {
	Record(labels map[string]string, v float64)
}

func NewQuantileMetricVec(metricRegistry MetricRegistry, metricKeys []*MetricKey, quantiles []float64) QuantileMetricVec {
	return &DefaultQuantileMetricVec{
		metricRegistry: metricRegistry,
		metricKeys:     metricKeys,
		mux:            sync.RWMutex{},
		cache:          make(map[string]*aggregate.TimeWindowQuantile),
		quantiles:      quantiles,
	}
}

// DefaultQuantileMetricVec is a default QuantileMetricVec implementation.
//
// It is concurrent safe, and it uses the aggregate.TimeWindowQuantile to store and calculate the quantile metrics.
type DefaultQuantileMetricVec struct {
	metricRegistry MetricRegistry
	metricKeys     []*MetricKey
	mux            sync.RWMutex
	cache          map[string]*aggregate.TimeWindowQuantile // key: metrics labels, value: TimeWindowQuantile
	quantiles      []float64
}

func (d *DefaultQuantileMetricVec) Record(labels map[string]string, v float64) {
	key := labelsToString(labels)
	if key == "" {
		return
	}
	d.mux.RLock()
	twq, ok := d.cache[key]
	d.mux.RUnlock()
	if !ok {
		d.mux.Lock()
		twq, ok = d.cache[key]
		if !ok {
			twq = aggregate.NewTimeWindowQuantile(DefaultCompression, DefaultBucketNum, DefaultTimeWindowSeconds)
			d.cache[key] = twq
		}
		d.mux.Unlock()
	}
	twq.Add(v)

	for i, q := range twq.Quantiles(d.quantiles) {
		d.metricRegistry.Gauge(NewMetricIdByLabels(d.metricKeys[i], labels)).Set(q)
	}
}

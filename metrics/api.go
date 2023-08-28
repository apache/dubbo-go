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
		// defalut protocol is already set in metricConfig
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

// AddCollector add more indicators, like  metadata、sla、configcenter etc
func AddCollector(name string, fun func(MetricRegistry, *ReporterConfig)) {
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

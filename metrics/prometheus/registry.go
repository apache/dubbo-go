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

package prometheus

import (
	"bytes"
	"sync"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func init() {
	metrics.SetRegistry("prometheus", func(rc *metrics.ReporterConfig) metrics.MetricRegistry {
		return &promMetricRegistry{r: prom.DefaultRegisterer}
	})
}

type promMetricRegistry struct {
	r prom.Registerer // for convenience of testing
	vecs sync.Map
}

func (p *promMetricRegistry) getOrComputeVec(key string, supplier func()interface{}) interface{} {
	v, ok := p.vecs.Load(key)
	if !ok {
		v, ok = p.vecs.LoadOrStore(key, supplier())
		if !ok {
			p.r.MustRegister(v.(prom.Collector))// only registe collector which stored success
		}
	}
	return v
}

func (p *promMetricRegistry) Counter(m *metrics.MetricId) metrics.CounterMetric {
	vec := p.getOrComputeVec(m.Name, func() interface{} {
		return prom.NewCounterVec(prom.CounterOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.CounterVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Gauge(m *metrics.MetricId) metrics.GaugeMetric {
	vec := p.getOrComputeVec(m.Name, func() interface{} {
		return prom.NewGaugeVec(prom.GaugeOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.GaugeVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Histogram(m *metrics.MetricId) metrics.ObservableMetric {
	vec := p.getOrComputeVec(m.Name, func() interface{} {
		return prom.NewHistogramVec(prom.HistogramOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.HistogramVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Summary(m *metrics.MetricId) metrics.ObservableMetric {
	vec := p.getOrComputeVec(m.Name, func() interface{} {
		return prom.NewSummaryVec(prom.SummaryOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
	}).(*prom.SummaryVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Rt(m *metrics.MetricId) metrics.ObservableMetric {
	vec := p.getOrComputeVec(m.Name, func() interface{} {
		return NewRtVec(&RtOpts{
			Name: m.Name,
			Help: m.Desc,
			bucketNum: 10, // TODO configurable
			timeWindowSeconds: 120, // TODO configurable
		}, m.TagKeys())
	}).(*RtVec)
	return vec.With(m.Tags)
}

func (p *promMetricRegistry) Export() {
	// use promauto export global, TODO move here
}

func (p *promMetricRegistry) Scrape() (string, error) {
	r := p.r.(prom.Gatherer)
	gathering, err := r.Gather()
	if err != nil {
		return "", err
	}
	out := &bytes.Buffer{}
	for _, mf := range gathering {
		if _, err := expfmt.MetricFamilyToText(out, mf); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}

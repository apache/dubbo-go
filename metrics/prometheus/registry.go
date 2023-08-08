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
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/common/expfmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func init() {
	metrics.SetRegistry("prometheus", func(rc *metrics.ReporterConfig) metrics.MetricRegistry {
		return &promMetricRegistry{
			cvm: make(map[string]*prom.CounterVec),
			gvm: make(map[string]*prom.GaugeVec),
			hvm: make(map[string]*prom.HistogramVec),
			svm: make(map[string]*prom.SummaryVec),
		}
	})
}

type promMetricRegistry struct {
	mtx sync.RWMutex                  // Protects metrics.
	cvm map[string]*prom.CounterVec   // prom.CounterVec
	gvm map[string]*prom.GaugeVec     // prom.GaugeVec
	hvm map[string]*prom.HistogramVec // prom.HistogramVec
	svm map[string]*prom.SummaryVec   // prom.SummaryVec
}

func (p *promMetricRegistry) Counter(m *metrics.MetricId) metrics.CounterMetric {
	p.mtx.RLock()
	vec, ok := p.cvm[m.Name]
	p.mtx.RUnlock()
	if !ok {
		p.mtx.Lock()
		vec = promauto.NewCounterVec(prom.CounterOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
		p.cvm[m.Name] = vec
		p.mtx.Unlock()
	}
	c := vec.With(m.Tags)
	return &counter{pc: c}
}

func (p *promMetricRegistry) Gauge(m *metrics.MetricId) metrics.GaugeMetric {
	p.mtx.RLock()
	vec, ok := p.gvm[m.Name]
	p.mtx.RUnlock()
	if !ok {
		p.mtx.Lock()
		vec = promauto.NewGaugeVec(prom.GaugeOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
		p.gvm[m.Name] = vec
		p.mtx.Unlock()
	}
	g := vec.With(m.Tags)
	return &gauge{pg: g}
}

func (p *promMetricRegistry) Histogram(m *metrics.MetricId) metrics.HistogramMetric {
	p.mtx.RLock()
	vec, ok := p.hvm[m.Name]
	p.mtx.RUnlock()
	if !ok {
		p.mtx.Lock()
		vec = promauto.NewHistogramVec(prom.HistogramOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
		p.hvm[m.Name] = vec
		p.mtx.Unlock()
	}
	h := vec.With(m.Tags)
	return &histogram{ph: h.(prom.Histogram)}
}

func (p *promMetricRegistry) Summary(m *metrics.MetricId) metrics.SummaryMetric {
	p.mtx.RLock()
	vec, ok := p.svm[m.Name]
	p.mtx.RUnlock()
	if !ok {
		p.mtx.Lock()
		vec = promauto.NewSummaryVec(prom.SummaryOpts{
			Name: m.Name,
			Help: m.Desc,
		}, m.TagKeys())
		p.svm[m.Name] = vec
		p.mtx.Unlock()
	}
	s := vec.With(m.Tags)
	return &summary{ps: s.(prom.Summary)}
}

func (p *promMetricRegistry) Export() {

}

func (p *promMetricRegistry) Scrape() (string, error) {
	r := prom.DefaultRegisterer.(*prom.Registry)
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

type counter struct {
	pc prom.Counter
}

func (c *counter) Inc() {
	c.pc.Inc()
}
func (c *counter) Add(v float64) {
	c.pc.Add(v)
}

type gauge struct {
	pg prom.Gauge
}

//	func (g *gauge) Inc() {
//		g.pg.Inc()
//	}
//
//	func (g *gauge) Dec() {
//		g.pg.Dec()
//	}
func (g *gauge) Set(v float64) {
	g.pg.Set(v)
}

// func (g *gauge) Add(v float64) {
// 	g.pg.Add(v)
// }
// func (g *gauge) Sub(v float64) {
// 	g.pg.Sub(v)
// }

type histogram struct {
	ph prom.Histogram
}

func (h *histogram) Record(v float64) {
	h.ph.Observe(v)
}

type summary struct {
	ps prom.Summary
}

func (s *summary) Record(v float64) {
	s.ps.Observe(v)
}

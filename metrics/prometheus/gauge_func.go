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
	"encoding/json"
	"sort"
	"sync"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

// gaugeFuncVec keeps one callback per label set without changing ordinary gauges.
type gaugeFuncVec struct {
	desc       *prom.Desc
	labelNames []string
	mu         sync.RWMutex
	values     map[string]gaugeFuncValue
}

type gaugeFuncValue struct {
	labels []string
	value  func() float64
}

func newGaugeFuncVec(m *metrics.MetricId) *gaugeFuncVec {
	names := m.TagKeys()
	sort.Strings(names)
	return &gaugeFuncVec{
		desc:       prom.NewDesc(m.Name, m.Desc, names, nil),
		labelNames: names,
		values:     make(map[string]gaugeFuncValue),
	}
}

func (v *gaugeFuncVec) add(tags map[string]string, value func() float64) {
	if value == nil {
		panic("metrics: nil gauge function")
	}
	if len(tags) != len(v.labelNames) {
		panic("metrics: inconsistent gauge label cardinality")
	}
	labels := make([]string, len(v.labelNames))
	for i, name := range v.labelNames {
		var ok bool
		labels[i], ok = tags[name]
		if !ok {
			panic("metrics: inconsistent gauge label names")
		}
	}
	// Encoding label values avoids collisions when they contain separators.
	key, _ := json.Marshal(labels)
	v.mu.Lock()
	defer v.mu.Unlock()
	// A recreated window vector owns a new window for the same series.
	// Replace the callback instead of continuing to collect the old window.
	v.values[string(key)] = gaugeFuncValue{labels: labels, value: value}
}

func (v *gaugeFuncVec) Describe(ch chan<- *prom.Desc) {
	ch <- v.desc
}

func (v *gaugeFuncVec) Collect(ch chan<- prom.Metric) {
	v.mu.RLock()
	values := make([]gaugeFuncValue, 0, len(v.values))
	for _, value := range v.values {
		values = append(values, value)
	}
	v.mu.RUnlock()
	// Do not hold the series lock while evaluating or sending metrics.
	// A collection already using this snapshot may finish with an old callback.
	for _, value := range values {
		ch <- prom.MustNewConstMetric(v.desc, prom.GaugeValue, value.value(), value.labels...)
	}
}

var _ metrics.GaugeFuncRegistry = (*promMetricRegistry)(nil)

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
	"math"
	"sync"
	"testing"
)

import (
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func TestHistogramWithOptions(t *testing.T) {
	reg := prom.NewRegistry()
	p := NewPromMetricRegistry(reg, url)
	buckets := []float64{0.1, 0.3, 1}
	id := &metrics.MetricId{Name: "configured_seconds", Desc: "Configured latency", Tags: map[string]string{"method": "Get"}}
	h := p.HistogramWithOptions(id, &metrics.HistogramOpts{Buckets: buckets})
	// Neither caller mutation nor subsequent options may change registered bounds.
	buckets[0] = 0.9
	h.Observe(0.0005)
	p.HistogramWithOptions(id, &metrics.HistogramOpts{Buckets: []float64{2, 3}}).Observe(0.2)
	id2 := &metrics.MetricId{Name: id.Name, Desc: id.Desc, Tags: map[string]string{"method": "Create"}}
	p.HistogramWithOptions(id2, nil).Observe(0.8)

	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Len(t, families[0].Metric, 2)
	for _, m := range families[0].Metric {
		hist := m.GetHistogram()
		require.Len(t, hist.Bucket, 3)
		assert.InDelta(t, 0.1, hist.Bucket[0].GetUpperBound(), 1e-12)
		assert.InDelta(t, 0.3, hist.Bucket[1].GetUpperBound(), 1e-12)
		assert.InDelta(t, 1, hist.Bucket[2].GetUpperBound(), 1e-12)
		if m.Label[0].GetValue() == "Get" {
			assert.Equal(t, uint64(2), hist.GetSampleCount())
			assert.InDelta(t, 0.2005, hist.GetSampleSum(), 1e-12)
			assert.Equal(t, uint64(1), hist.Bucket[0].GetCumulativeCount())
			assert.Equal(t, uint64(2), hist.Bucket[1].GetCumulativeCount())
		}
	}
	first, err := p.Scrape()
	require.NoError(t, err)
	second, err := p.Scrape()
	require.NoError(t, err)
	assert.Equal(t, first, second, "scraping must not observe or reset samples")
	assert.Contains(t, first, `configured_seconds_bucket{method="Get",le="+Inf"} 2`)

	// An unrelated histogram retains the client's existing default buckets.
	p.Histogram(metricId).Observe(0.01)
	text, err := p.Scrape()
	require.NoError(t, err)
	assert.Contains(t, text, `dubbo_request_bucket{app="dubbo",version="1.0.0",le="0.005"} 0`)
}

func TestHistogramWithOptionsDefaults(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts *metrics.HistogramOpts
	}{
		{"nil", nil},
		{"empty", &metrics.HistogramOpts{}},
		{"invalid", &metrics.HistogramOpts{Buckets: []float64{math.NaN()}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reg := prom.NewRegistry()
			p := NewPromMetricRegistry(reg, url)
			p.HistogramWithOptions(metricId, tt.opts).Observe(0.01)
			families, err := reg.Gather()
			require.NoError(t, err)
			hist := families[0].Metric[0].GetHistogram()
			require.Len(t, hist.Bucket, len(prom.DefBuckets))
			for i, bound := range prom.DefBuckets {
				assert.InDelta(t, bound, hist.Bucket[i].GetUpperBound(), 1e-12)
			}
		})
	}
}

func TestHistogramWithOptionsConcurrent(t *testing.T) {
	reg := prom.NewRegistry()
	p := NewPromMetricRegistry(reg, url)
	opts := &metrics.HistogramOpts{Buckets: []float64{0.1, 1}}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 100 {
				p.HistogramWithOptions(metricId, opts).Observe(0.0005)
			}
		})
	}
	wg.Wait()
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	hist := families[0].Metric[0].GetHistogram()
	assert.Equal(t, uint64(2000), hist.GetSampleCount())
	assert.InDelta(t, 1, hist.GetSampleSum(), 1e-10)
	assert.Equal(t, uint64(2000), hist.Bucket[0].GetCumulativeCount())
}

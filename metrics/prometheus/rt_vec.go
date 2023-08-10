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
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"
)

type rtMetric struct {
	nameSuffix string
	helpPrefix string
	valueFunc  func(*aggregate.Result) float64
}

func (m *rtMetric) desc(opts *RtOpts, labels []string) *prom.Desc {
	return prom.NewDesc(prom.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name+m.nameSuffix), m.helpPrefix+opts.Help, labels, opts.ConstLabels)
}

var rtMetrics = make([]*rtMetric, 5)

func init() {
	rtMetrics[0] = &rtMetric{nameSuffix: "_sum", helpPrefix: "Sum ", valueFunc: func(r *aggregate.Result) float64 { return r.Total }}
	rtMetrics[1] = &rtMetric{nameSuffix: "_last", helpPrefix: "Last ", valueFunc: func(r *aggregate.Result) float64 { return r.Last }}
	rtMetrics[2] = &rtMetric{nameSuffix: "_min", helpPrefix: "Min ", valueFunc: func(r *aggregate.Result) float64 { return r.Min }}
	rtMetrics[3] = &rtMetric{nameSuffix: "_max", helpPrefix: "Max ", valueFunc: func(r *aggregate.Result) float64 { return r.Max }}
	rtMetrics[4] = &rtMetric{nameSuffix: "_avg", helpPrefix: "Average ", valueFunc: func(r *aggregate.Result) float64 { return r.Avg }}
}

type RtOpts struct {
	Namespace         string
	Subsystem         string
	Name              string
	Help              string
	ConstLabels       prom.Labels
	bucketNum         int
	timeWindowSeconds int64
}

type valueAgg struct {
	tags map[string]string
	agg  *aggregate.TimeWindowAggregator
}

func (v *valueAgg) Observe(val float64) {
	v.agg.Add(val)
}

func buildKey(m map[string]string, labNames []string) string {
	var buffer bytes.Buffer
	for _, label := range labNames {
		if buffer.Len() != 0 {
			buffer.WriteString("_")
		}
		buffer.WriteString(m[label])
	}
	return buffer.String()
}

func buildLabelValues(m map[string]string, labNames []string) []string {
	values := make([]string, len(labNames))
	for i, label := range labNames {
		values[i] = m[label]
	}
	return values
}

type RtVec struct {
	opts       *RtOpts
	labelNames []string
	aggMap     sync.Map
}

func NewRtVec(opts *RtOpts, labNames []string) *RtVec {
	return &RtVec{
		opts:       opts,
		labelNames: labNames,
	}
}

func (r *RtVec) With(tags map[string]string) prom.Observer {
	k := buildKey(tags, r.labelNames)
	return r.computeIfAbsent(k, func() *valueAgg {
		return &valueAgg{
			tags: tags,
			agg:  aggregate.NewTimeWindowAggregator(r.opts.bucketNum, r.opts.timeWindowSeconds),
		}
	})
}

func (r *RtVec) computeIfAbsent(k string, supplier func() *valueAgg) *valueAgg {
	v, ok := r.aggMap.Load(k)
	if !ok {
		v, _ = r.aggMap.LoadOrStore(k, supplier())
	}
	return v.(*valueAgg)
}

func (r *RtVec) Collect(ch chan<- prom.Metric) {
	r.aggMap.Range(func(key, val interface{}) bool {
		v := val.(*valueAgg)
		res := v.agg.Result()
		for _, m := range rtMetrics {
			ch <- prom.MustNewConstMetric(m.desc(r.opts, r.labelNames), prom.GaugeValue, m.valueFunc(res), buildLabelValues(v.tags, r.labelNames)...)
		}
		return true
	})
}

func (r *RtVec) Describe(ch chan<- *prom.Desc) {
	for _, m := range rtMetrics {
		ch <- m.desc(r.opts, r.labelNames)
	}
}

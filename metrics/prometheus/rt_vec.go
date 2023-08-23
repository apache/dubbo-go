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
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"

	prom "github.com/prometheus/client_golang/prometheus"
)

type rtMetric struct {
	nameSuffix string
	helpPrefix string
	valueFunc  func(*aggregate.Result) float64
}

func (m *rtMetric) desc(opts *RtOpts, labels []string) *prom.Desc {
	return prom.NewDesc(prom.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name+m.nameSuffix), m.helpPrefix+opts.Help, labels, opts.ConstLabels)
}

var (
	rtMetrics  = make([]*rtMetric, 5)
	aggMetrics = make([]*rtMetric, 3)
)

func init() {
	rtMetrics[0] = &rtMetric{nameSuffix: "_sum", helpPrefix: "Sum ", valueFunc: func(r *aggregate.Result) float64 { return r.Total }}
	rtMetrics[1] = &rtMetric{nameSuffix: "_last", helpPrefix: "Last ", valueFunc: func(r *aggregate.Result) float64 { return r.Last }}
	rtMetrics[2] = &rtMetric{nameSuffix: "_min", helpPrefix: "Min ", valueFunc: func(r *aggregate.Result) float64 { return r.Min }}
	rtMetrics[3] = &rtMetric{nameSuffix: "_max", helpPrefix: "Max ", valueFunc: func(r *aggregate.Result) float64 { return r.Max }}
	rtMetrics[4] = &rtMetric{nameSuffix: "_avg", helpPrefix: "Average ", valueFunc: func(r *aggregate.Result) float64 { return r.Avg }}

	aggMetrics[0] = &rtMetric{nameSuffix: "_avg_milliseconds_aggregate", helpPrefix: "The average ", valueFunc: func(r *aggregate.Result) float64 { return r.Avg }}
	aggMetrics[1] = &rtMetric{nameSuffix: "_min_milliseconds_aggregate", helpPrefix: "The minimum ", valueFunc: func(r *aggregate.Result) float64 { return r.Min }}
	aggMetrics[2] = &rtMetric{nameSuffix: "_max_milliseconds_aggregate", helpPrefix: "The maximum ", valueFunc: func(r *aggregate.Result) float64 { return r.Max }}
}

type RtOpts struct {
	Namespace         string
	Subsystem         string
	Name              string
	Help              string
	ConstLabels       prom.Labels
	bucketNum         int   // only for aggRt
	timeWindowSeconds int64 // only for aggRt
}

type observer interface {
	Observe(val float64)
	result() *aggregate.Result
}

type aggResult struct {
	agg *aggregate.TimeWindowAggregator
}

func (r *aggResult) Observe(val float64) {
	r.agg.Add(val)
}

func (r *aggResult) result() *aggregate.Result {
	return r.agg.Result()
}

type valueResult struct {
	mtx sync.RWMutex
	val *aggregate.Result
}

func (r *valueResult) Observe(val float64) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.val.Update(val)
}

func (r *valueResult) result() *aggregate.Result {
	res := aggregate.NewResult()
	r.mtx.RLock()
	res.Merge(r.val)
	r.mtx.RUnlock()
	return res.Get()
}

type Rt struct {
	tags map[string]string
	obs  observer
}

func (r *Rt) Observe(val float64) {
	r.obs.Observe(val)
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
	initFunc   func(tags map[string]string) *Rt
	metrics    []*rtMetric
}

func NewRtVec(opts *RtOpts, labNames []string) *RtVec {
	return &RtVec{
		opts:       opts,
		labelNames: labNames,
		metrics:    rtMetrics,
		initFunc: func(tags map[string]string) *Rt {
			return &Rt{
				tags: tags,
				obs:  &valueResult{val: aggregate.NewResult()},
			}
		},
	}
}

func NewAggRtVec(opts *RtOpts, labNames []string) *RtVec {
	return &RtVec{
		opts:       opts,
		labelNames: labNames,
		metrics:    aggMetrics,
		initFunc: func(tags map[string]string) *Rt {
			return &Rt{
				tags: tags,
				obs:  &aggResult{agg: aggregate.NewTimeWindowAggregator(opts.bucketNum, opts.timeWindowSeconds)},
			}
		},
	}
}

func (r *RtVec) With(tags map[string]string) prom.Observer {
	k := buildKey(tags, r.labelNames)
	return r.computeIfAbsent(k, func() *Rt {
		return r.initFunc(tags)
	})
}

func (r *RtVec) computeIfAbsent(k string, supplier func() *Rt) *Rt {
	v, ok := r.aggMap.Load(k)
	if !ok {
		v, _ = r.aggMap.LoadOrStore(k, supplier())
	}
	return v.(*Rt)
}

func (r *RtVec) Collect(ch chan<- prom.Metric) {
	r.aggMap.Range(func(_, val interface{}) bool {
		v := val.(*Rt)
		res := v.obs.result()
		for _, m := range r.metrics {
			ch <- prom.MustNewConstMetric(m.desc(r.opts, r.labelNames), prom.GaugeValue, m.valueFunc(res), buildLabelValues(v.tags, r.labelNames)...)
		}
		return true
	})
}

func (r *RtVec) Describe(ch chan<- *prom.Desc) {
	for _, m := range r.metrics {
		ch <- m.desc(r.opts, r.labelNames)
	}
}

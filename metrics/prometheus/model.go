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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metrics/util/aggregate"
)

func newHistogramVec(name, namespace string, labels []string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      name,
			Buckets:   defaultHistogramBucket,
		},
		labels)
}

func newCounter(name, namespace string) prometheus.Counter {
	return prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      name,
		})
}

func newCounterVec(name, namespace string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

func newGauge(name, namespace string) prometheus.Gauge {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      name,
			Namespace: namespace,
		})
}

func newGaugeVec(name, namespace string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

func newSummary(name, namespace string) prometheus.Summary {
	return prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:      name,
			Namespace: namespace,
		})
}

// newSummaryVec create SummaryVec, the Namespace is dubbo
// the objectives is from my experience.
func newSummaryVec(name, namespace string, labels []string, maxAge int64) *prometheus.SummaryVec {
	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Name:      name,
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.75:  0.01,
				0.90:  0.005,
				0.98:  0.002,
				0.99:  0.001,
				0.999: 0.0001,
			},
			MaxAge: time.Duration(maxAge),
		},
		labels,
	)
}

// create an auto register histogram vec
func newAutoHistogramVec(name, namespace string, labels []string) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      name,
			Buckets:   defaultHistogramBucket,
		},
		labels)
}

// create an auto register counter vec
func newAutoCounterVec(name, namespace string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

// create an auto register gauge vec
func newAutoGaugeVec(name, namespace string, labels []string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

// create an auto register summary vec
func newAutoSummaryVec(name, namespace string, labels []string, maxAge int64) *prometheus.SummaryVec {
	return promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Name:      name,
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.75:  0.01,
				0.90:  0.005,
				0.98:  0.002,
				0.99:  0.001,
				0.999: 0.0001,
			},
			MaxAge: time.Duration(maxAge),
		},
		labels,
	)
}

type GaugeVecWithSyncMap struct {
	GaugeVec *prometheus.GaugeVec
	SyncMap  *sync.Map // key: labels, value: *atomic.Value
}

func newAutoGaugeVecWithSyncMap(name, namespace string, labels []string) *GaugeVecWithSyncMap {
	return &GaugeVecWithSyncMap{
		GaugeVec: newAutoGaugeVec(name, namespace, labels),
		SyncMap:  &sync.Map{},
	}
}

func convertLabelsToMapKey(labels prometheus.Labels) string {
	return strings.Join([]string{
		labels[applicationNameKey],
		labels[groupKey],
		labels[hostnameKey],
		labels[interfaceKey],
		labels[ipKey],
		labels[versionKey],
		labels[methodKey],
	}, "_")
}

func (gv *GaugeVecWithSyncMap) updateMin(labels *prometheus.Labels, curValue int64) {
	key := convertLabelsToMapKey(*labels)
	cur := &atomic.Value{} // for first store
	cur.Store(curValue)
	for {
		if actual, loaded := gv.SyncMap.LoadOrStore(key, cur); loaded {
			store := actual.(*atomic.Value)
			storeValue := store.Load().(int64)
			if curValue < storeValue {
				if store.CompareAndSwap(storeValue, curValue) {
					// value is not changed, should update
					gv.GaugeVec.With(*labels).Set(float64(curValue))
					break
				}
				// value has changed, continue for loop
			} else {
				// no need to update
				break
			}
		} else {
			// store current curValue as this labels' init value
			gv.GaugeVec.With(*labels).Set(float64(curValue))
			break
		}
	}
}

func (gv *GaugeVecWithSyncMap) updateMax(labels *prometheus.Labels, curValue int64) {
	key := convertLabelsToMapKey(*labels)
	cur := &atomic.Value{} // for first store
	cur.Store(curValue)
	for {
		if actual, loaded := gv.SyncMap.LoadOrStore(key, cur); loaded {
			store := actual.(*atomic.Value)
			storeValue := store.Load().(int64)
			if curValue > storeValue {
				if store.CompareAndSwap(storeValue, curValue) {
					// value is not changed, should update
					gv.GaugeVec.With(*labels).Set(float64(curValue))
					break
				}
				// value has changed, continue for loop
			} else {
				// no need to update
				break
			}
		} else {
			// store current curValue as this labels' init value
			gv.GaugeVec.With(*labels).Set(float64(curValue))
			break
		}
	}
}

func (gv *GaugeVecWithSyncMap) updateAvg(labels *prometheus.Labels, curValue int64) {
	key := convertLabelsToMapKey(*labels)
	cur := &atomic.Value{} // for first store
	type avgPair struct {
		Sum int64
		N   int64
	}
	cur.Store(avgPair{Sum: curValue, N: 1})

	for {
		if actual, loaded := gv.SyncMap.LoadOrStore(key, cur); loaded {
			store := actual.(*atomic.Value)
			storeValue := store.Load().(avgPair)
			newValue := avgPair{Sum: storeValue.Sum + curValue, N: storeValue.N + 1}
			if store.CompareAndSwap(storeValue, newValue) {
				// value is not changed, should update
				gv.GaugeVec.With(*labels).Set(float64(newValue.Sum / newValue.N))
				break
			}
		} else {
			// store current curValue as this labels' init value
			gv.GaugeVec.With(*labels).Set(float64(curValue))
			break
		}
	}
}

type quantileGaugeVec struct {
	gaugeVecSlice []*prometheus.GaugeVec
	quantiles     []float64
	syncMap       *sync.Map // key: labels string, value: TimeWindowQuantile
}

// Notice: names and quantiles should be the same length and same order.
func newQuantileGaugeVec(names []string, namespace string, labels []string, quantiles []float64) *quantileGaugeVec {
	gvs := make([]*prometheus.GaugeVec, len(names))
	for i, name := range names {
		gvs[i] = newAutoGaugeVec(name, namespace, labels)
	}
	gv := &quantileGaugeVec{
		gaugeVecSlice: gvs,
		quantiles:     quantiles,
		syncMap:       &sync.Map{},
	}
	return gv
}

func (gv *quantileGaugeVec) updateQuantile(labels *prometheus.Labels, curValue int64) {
	key := convertLabelsToMapKey(*labels)
	cur := aggregate.NewTimeWindowQuantile(100, 10, 120)
	cur.Add(float64(curValue))

	updateFunc := func(td *aggregate.TimeWindowQuantile) {
		qs := td.Quantiles(gv.quantiles)
		for i, q := range qs {
			gv.gaugeVecSlice[i].With(*labels).Set(q)
		}
	}

	if actual, loaded := gv.syncMap.LoadOrStore(key, cur); loaded {
		store := actual.(*aggregate.TimeWindowQuantile)
		store.Add(float64(curValue))
		updateFunc(store)
	} else {
		updateFunc(cur)
	}
}

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
)

import (
	"github.com/prometheus/client_golang/prometheus"
)

type GaugeVecWithSyncMap struct {
	GaugeVec *prometheus.GaugeVec
	SyncMap  *sync.Map
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

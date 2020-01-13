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

package impl

import (
	"sync"
	"sync/atomic"
)

import (
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metrics"
)

type DefaultMetricRegistry struct {
	metricsMap sync.Map

	// record the metricsCount to avoid iterate the metricsMap
	metricsCount int32

	maxMetricCount int
}

func (mri *DefaultMetricRegistry) GetCompasses() map[string]*metrics.MetricNameToMetricEntry {
	return mri.getMetrics(func(entry *metrics.MetricNameToMetricEntry) bool {
		_, ok := entry.Metric.(metrics.Compass)
		return ok
	})
}

func (mri *DefaultMetricRegistry) GetFastCompasses() map[string]*metrics.MetricNameToMetricEntry {
	return mri.getMetrics(func(entry *metrics.MetricNameToMetricEntry) bool {
		_, ok := entry.Metric.(metrics.FastCompass)
		return ok
	})
}

func (mri *DefaultMetricRegistry) getMetrics(
	matchType func(entry *metrics.MetricNameToMetricEntry) bool) map[string]*metrics.MetricNameToMetricEntry {
	result := make(map[string]*metrics.MetricNameToMetricEntry, 16)
	mri.metricsMap.Range(func(key, value interface{}) bool {
		entry := value.(*metrics.MetricNameToMetricEntry)
		ok := matchType(entry)
		if ok {
			result[key.(string)] = entry
		}
		return true
	})
	return result
}

func (mri *DefaultMetricRegistry) GetMetrics() map[string]*metrics.MetricNameToMetricEntry {
	result := make(map[string]*metrics.MetricNameToMetricEntry, mri.metricsCount)
	mri.metricsMap.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*metrics.MetricNameToMetricEntry)
		return true
	})
	return result
}

func (mri *DefaultMetricRegistry) LastUpdateTime() int64 {
	result := int64(0)
	mri.metricsMap.Range(func(_, value interface{}) bool {
		entry := value.(*metrics.MetricNameToMetricEntry)
		if result < entry.Metric.LastUpdateTime() {
			result = entry.Metric.LastUpdateTime()
		}
		return true
	})
	return result
}

func (mri *DefaultMetricRegistry) GetCompass(name *metrics.MetricName) metrics.Compass {
	mc := config.GetMetricConfig()
	return mri.getMetric(
		name,
		func() metrics.Metric {
			return NewCompassWithType(
				mc.GetReservoirType(),
				metrics.DefaultClock,
				mc.GetBucketCount(),
				mc.GetLevelInterval(int(name.Level)),
				mc.GetMaxCompassAddonCount(),
				mc.GetMaxCompassAddonCount())
		},
		nopCompass).(metrics.Compass)
}

func (mri *DefaultMetricRegistry) GetFastCompass(name *metrics.MetricName) metrics.FastCompass {
	mc := config.GetMetricConfig()
	return mri.getMetric(
		name,
		func() metrics.Metric {
			return newFastCompass(mc.GetLevelInterval(int(name.Level)), mc.GetBucketCount())
		},
		nopFastCompass).(metrics.FastCompass)
}

func (mri *DefaultMetricRegistry) getMetric(
	name *metrics.MetricName,
	creator func() metrics.Metric,
	nopMetric metrics.Metric) metrics.Metric {
	result, found := mri.metricsMap.Load(name.HashKey())
	// fast path
	if found {
		return result.(*metrics.MetricNameToMetricEntry).Metric
	}

	// slow path
	newMetric := creator()

	// because the metricsCount increase monotonically, so the check and do something works well
	if int(mri.metricsCount) >= mri.maxMetricCount {
		// we are over the limitation of max metric count per registry
		newMetric = nopMetric
	}

	result, loaded := mri.metricsMap.LoadOrStore(name.HashKey(), &metrics.MetricNameToMetricEntry{
		MetricName: name,
		Metric:     newMetric,
	})
	if !loaded {
		// we store the new metric
		atomic.AddInt32(&mri.metricsCount, 1)
	}
	return result.(*metrics.MetricNameToMetricEntry).Metric
}

func NewMetricRegistry(maxMetricCount int) metrics.MetricRegistry {
	return &DefaultMetricRegistry{
		maxMetricCount: maxMetricCount,
		metricsMap:     sync.Map{},
	}
}

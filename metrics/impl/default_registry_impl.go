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

	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metrics"
)

type MetricRegistryImpl struct {
	metricsMap sync.Map

	// record the metricsCount to avoid iterate the metricsMap
	metricsCount int32

	maxMetricCount int
}

func (mri *MetricRegistryImpl) LastUpdateTime() int64 {
	panic("implement me")
}

func (mri *MetricRegistryImpl) GetMetrics() map[string]metrics.Metric {
	result := make(map[string]metrics.Metric, mri.metricsCount)
	mri.metricsMap.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(metrics.Metric)
		return true
	})
	return result
}

func (mri *MetricRegistryImpl) GetFastCompass(name metrics.MetricName) metrics.FastCompass {
	result, found := mri.metricsMap.Load(name.HashKey())
	// fast path
	if found {
		return result.(metrics.FastCompass)
	}

	// slow path
	newFastCmps := GetNopFastCompass()

	// because the metricsCount increase monotonically, so the check and do something works well
	if int(mri.metricsCount) < mri.maxMetricCount {
		// we are not over the limitation of max metric count per registry
		newFastCmps = newFastCompass(config.GetMetricConfig().GetLevelInterval(int(name.Level)))
	}

	result, loaded := mri.metricsMap.LoadOrStore(name.HashKey(), newFastCmps)
	if !loaded {
		// we store the new metric
		atomic.AddInt32(&mri.metricsCount, 1)
	}
	return result.(metrics.FastCompass)
}

func NewMetricRegistry() metrics.MetricRegistry {
	return &MetricRegistryImpl{
		maxMetricCount: config.GetMetricConfig().GetMaxMetricCountPerRegistry(),
	}
}




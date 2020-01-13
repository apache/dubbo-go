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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

func TestMetricRegistryImpl_GetFastCompass(t *testing.T) {
	registry := NewMetricRegistry(2)
	result := registry.GetMetrics()
	assert.Equal(t, 0, len(result))
	assert.Equal(t, int64(0), registry.LastUpdateTime())

	name := metrics.NewMetricName("Test", nil, metrics.Minor)
	fastCompass := registry.GetFastCompass(name)
	assert.NotNil(t, fastCompass)
	assert.IsType(t, &FastCompassImpl{}, fastCompass)

	fastCompass1 := registry.GetFastCompass(name)
	assert.Equal(t, fastCompass, fastCompass1)

	result = registry.GetMetrics()
	assert.Equal(t, 1, len(result))
	entry, found := result["Test"]
	assert.True(t, found)
	assert.Equal(t, fastCompass, entry.Metric)
	assert.Equal(t, name, entry.MetricName)

	fastCompass.Record(10*time.Second, "Test")
	assert.Equal(t, fastCompass.LastUpdateTime(), registry.LastUpdateTime())
	assert.True(t, fastCompass.LastUpdateTime() > 0)

	// over max count
	registry.GetFastCompass(metrics.NewMetricName("Test1", nil, metrics.Minor))
	fastCompass = registry.GetFastCompass(metrics.NewMetricName("Test2", nil, metrics.Minor))
	assert.Equal(t, nopFastCompass, fastCompass)
}

func TestMetricRegistryImpl_GetCompass(t *testing.T) {
	registry := NewMetricRegistry(2)
	compass := registry.GetCompass(metrics.NewMetricName("Test1", nil, metrics.Minor))
	assert.NotNil(t, compass)
}

func TestDefaultMetricRegistry_GetCompasses(t *testing.T) {
	registry := NewMetricRegistry(2)
	compassMetric := metrics.NewMetricName("Test1", nil, metrics.Minor)
	compass := registry.GetCompass(compassMetric)

	fastCompassMetric := metrics.NewMetricName("Test2", nil, metrics.Minor)
	fastCompass := registry.GetFastCompass(fastCompassMetric)

	cmMap := registry.GetCompasses()
	fcmMap := registry.GetFastCompasses()
	assert.Equal(t, 1, len(cmMap))
	assert.Equal(t, 1, len(fcmMap))

	assert.Equal(t, compass, cmMap[compassMetric.HashKey()].Metric)
	assert.Equal(t, fastCompass, fcmMap[fastCompassMetric.HashKey()].Metric)
}

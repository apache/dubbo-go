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

package config

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

func TestMetricConfig_GetEnableMetrics(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, 0, len(config.GetEnableMetrics()))
	config.Enables = []string{"compass", "all", "fastCompass"}
	assert.Equal(t, allMetrics, config.GetEnableMetrics())

	config.Enables = []string{"compass"}
	assert.Equal(t, 1, len(config.GetEnableMetrics()))
	assert.Equal(t, "compass", config.GetEnableMetrics()[0])
}

func TestMetricConfig_GetMaxCompassAddonCount(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, defaultMaxCompassAddonCount, config.GetMaxCompassAddonCount())
	config.MaxCompassAddonCount = -1
	assert.Equal(t, defaultMaxCompassAddonCount, config.GetMaxCompassAddonCount())
	config.MaxCompassAddonCount = 23
	assert.Equal(t, 23, config.GetMaxCompassAddonCount())
}

func TestMetricConfig_GetMaxCompassErrorCodeCount(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, defaultMaxCompassErrorCodeCount, config.GetMaxCompassErrorCodeCount())
	config.MaxCompassErrorCodeCount = -1
	assert.Equal(t, defaultMaxCompassErrorCodeCount, config.GetMaxCompassErrorCodeCount())
	config.MaxCompassErrorCodeCount = 13
	assert.Equal(t, 13, config.GetMaxCompassErrorCodeCount())
}

func TestMetricConfig_GetGlobalInterval(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, defaultGlobalInterval, config.GetGlobalInterval())

	// the interval is < 1s, so the default value will be used
	config.GlobalInterval = 10 * time.Millisecond
	assert.Equal(t, defaultGlobalInterval, config.GetGlobalInterval())

	config.GlobalInterval = 20 * time.Second
	assert.Equal(t, 20*time.Second, config.GetGlobalInterval())
}

func TestMetricConfig_GetLevelInterval(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, defaultGlobalInterval, config.GetLevelInterval(0))

	config.LevelInterval = make(map[int]time.Duration, 2)
	assert.Equal(t, defaultGlobalInterval, config.GetLevelInterval(0))

	// less then 1s, Global value will be returned
	config.LevelInterval[12] = 12 * time.Millisecond
	assert.Equal(t, config.GetGlobalInterval(), config.GetLevelInterval(12))

	config.LevelInterval[10] = 5 * time.Second
	assert.Equal(t, 5*time.Second, config.GetLevelInterval(10))
}

func TestMetricConfig_GetMaxMetricCountPerRegistry(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, defaultMaxMetricCountPerRegistry, config.GetMaxMetricCountPerRegistry())
	config.MaxMetricCountPerRegistry = 120
	assert.Equal(t, 120, config.GetMaxMetricCountPerRegistry())
}

func TestMetricConfig_GetMaxSubCategoryCount(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, defaultMaxSubCategoryCount, config.GetMaxSubCategoryCount())
	config.MaxSubCategoryCount = 13
	assert.Equal(t, 13, config.GetMaxSubCategoryCount())
}

func TestMetricConfig_GetMetricManagerName(t *testing.T) {
	config := MetricConfig{}
	assert.Equal(t, constant.DEFAULT_KEY, config.GetMetricManagerName())

	config.Manager = "mock"
	assert.Equal(t, "mock", config.GetMetricManagerName())
}

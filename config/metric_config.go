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
	"github.com/apache/dubbo-go/common/constant"
)

const (
	defaultMaxSubCategoryCount = 20
	defaultGlobalInterval      = 60
)

type MetricConfig struct {
	/**
	 * the MetricManager's name. You can use 'default' to use the default implementation.
	 */
	Manager string `yaml:"manager" json:"manager,omitempty"`
	/**
	 * the max sub category count, it's same with com.alibaba.metrics.maxSubCategoryCount
	 */
	MaxSubCategoryCount int `default:"20" yaml:"max_subcategory_count" json:"max_subcategory_count,omitempty"`

	/**
	 * the interval of collecting data, or report data, and so on...
	 * the unit is second
	 * see Interval
	 * default value is 60(s)
	 */
	GlobalInterval int `default:"60" yaml:"global_interval" json:"global_interval,omitempty"`

	/**
	 * MetricLevel -> interval
	 * we will use this map to find out the interval of the MetricLevel.
	 */
	LevelInterval map[int]int `yaml:"level_interval" json:"level_interval,omitempty"`
}

func (mc *MetricConfig) GetLevelInterval(metricLevel int) int {
	if mc.LevelInterval == nil {
		return mc.GetGlobalInterval()
	}
	result, found := mc.LevelInterval[metricLevel]
	if found {
		return result
	}
	return mc.GetGlobalInterval()
}

func (mc *MetricConfig) GetGlobalInterval() int {
	if mc.GlobalInterval <= 0 {
		return defaultGlobalInterval
	}
	return mc.GlobalInterval
}

func (mc *MetricConfig) GetMetricManagerName() string {
	if len(mc.Manager) <= 0 {
		return constant.DEFAULT_KEY
	}
	return mc.Manager
}

func (mc *MetricConfig) GetMaxSubCategoryCount() int {
	if mc.MaxSubCategoryCount <= 0 {
		return defaultMaxSubCategoryCount
	}
	return mc.MaxSubCategoryCount
}

/**
 * If the application is both consumer and provider, the provider's metric configuration will be used.
 * If and only if the application is just consumer, consumer's metric configuration wll be used.
 * Never return nil
 */
func GetMetricConfig() *MetricConfig {
	result := GetProviderConfig().MetricConfig
	if result == nil {
		result = GetConsumerConfig().MetricConfig
	}

	if result == nil {
		result = &MetricConfig{}
	}
	return result
}

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
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metrics"
)

func init() {
	extension.SetMetricManager(constant.DEFAULT_KEY, newDefaultMetricManager())
}

/*
 * this implementation dependent on  the MetricRegistry.
 * com.alibaba.metrics.AliMetricManager
 */
type DefaultMetricManager struct {
	// group name -> MetricRegistry
	metricRegistryMap sync.Map
	enable            bool
}

func (d *DefaultMetricManager) GetCompass(groupName string, metricName *metrics.MetricName) metrics.Compass {
	if !d.enable {
		return nopCompass
	}
	registry := d.getMetricRegistry(groupName)
	return registry.GetCompass(metricName)
}

func (d *DefaultMetricManager) IsEnable() bool {
	return d.enable
}

func (d *DefaultMetricManager) SetEnable(enable bool) {
	d.enable = enable
}

func (d *DefaultMetricManager) GetFastCompass(groupName string, metricName *metrics.MetricName) metrics.FastCompass {
	if !d.enable {
		return nopFastCompass
	}
	registry := d.getMetricRegistry(groupName)
	return registry.GetFastCompass(metricName)
}

func (d *DefaultMetricManager) getMetricRegistry(group string) metrics.MetricRegistry {
	// fast path, avoid creating the  MetricRegistry
	result, load := d.metricRegistryMap.Load(group)
	if load {
		return result.(metrics.MetricRegistry)
	}
	result, _ = d.metricRegistryMap.LoadOrStore(group,
		NewMetricRegistry(config.GetMetricConfig().GetMaxMetricCountPerRegistry()))
	return result.(metrics.MetricRegistry)
}

func newDefaultMetricManager() metrics.MetricManager {
	return &DefaultMetricManager{
		enable: true,
	}
}

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

package rolling

import (
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics/utils"
	"dubbo.apache.org/dubbo-go/v3/common"
)

var _ Metrics = (*EMAMetrics)(nil)

type EMAMetrics struct {
	opts    EMAOpts
	metrics sync.Map
}

func NewEMAMetrics(opts EMAOpts) *EMAMetrics {
	return &EMAMetrics{
		opts: opts,
	}
}

func (m *EMAMetrics) GetMethodMetrics(url *common.URL, methodName, key string) (float64, error) {
	metricsKey := utils.GetMethodMetricsKey(url, methodName, key)
	if metrics, ok := m.metrics.Load(metricsKey); ok {
		return metrics.(*EMA).Value(), nil
	}
	return 0, utils.ErrMetricsNotFound
}

func (m *EMAMetrics) AppendMethodMetrics(url *common.URL, methodName, key string, val float64) error {
	metricsKey := utils.GetMethodMetricsKey(url, methodName, key)
	if metrics, ok := m.metrics.Load(metricsKey); ok {
		metrics.(*EMA).Append(val)
	} else {
		metrics, _ = m.metrics.LoadOrStore(metricsKey, NewEMA(m.opts))
		metrics.(*EMA).Append(val)
	}
	return nil
}

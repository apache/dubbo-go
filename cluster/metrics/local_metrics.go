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

package metrics

import (
	"fmt"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

var LocalMetrics Metrics

func init() {
	LocalMetrics = newLocalMetrics()
}

var _ Metrics = (*localMetrics)(nil)

type localMetrics struct {
	// protect metrics
	lock    *sync.RWMutex
	metrics map[string]interface{}
}

func newLocalMetrics() *localMetrics {
	return &localMetrics{
		lock:    new(sync.RWMutex),
		metrics: make(map[string]interface{}),
	}
}

func (m *localMetrics) GetMethodMetrics(url *common.URL, methodName, key string) (interface{}, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	if metrics, ok := m.metrics[metricsKey]; ok {
		return metrics, nil
	}
	return nil, ErrMetricsNotFound
}

func (m *localMetrics) SetMethodMetrics(url *common.URL, methodName, key string, value interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	m.metrics[metricsKey] = value
	return nil
}

func (m *localMetrics) GetInvokerMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *localMetrics) SetInvokerMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}

func (m *localMetrics) GetInstanceMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *localMetrics) SetInstanceMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}

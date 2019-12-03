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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
)

func TestGetMetricManager(t *testing.T) {
	mockManager := &MockMetricsManager{}
	extension.SetMetricManager("mock", mockManager)

	mock1Manager := &MockMetricsManager{}
	extension.SetMetricManager("mock1", mock1Manager)

	// metric manager configured in consumer config
	consumerConfig := config.ConsumerConfig{
		MetricManager: "mock",
	}
	config.SetConsumerConfig(consumerConfig)
	result := GetMetricManager()
	assert.Equal(t, mockManager, result)

	// metric manager configured in provider config and consumer config
	providerConfig := config.ProviderConfig{
		MetricManager: "mock1",
	}
	config.SetProviderConfig(providerConfig)
	result = GetMetricManager()
	assert.Equal(t, mock1Manager, result)
}

type MockMetricsManager struct {
	mock.Mock
}

func (m *MockMetricsManager) GetFastCompass(name string, metricName MetricName) FastCompass {
	m.Called()
	return &MockFastCompass{}
}



type MockFastCompass struct {
	mock.Mock
}

func (m *MockFastCompass) Record(duration int64, subCategory string) {
	m.Called()
}

func (m *MockFastCompass) LastUpdateTime() int64 {
	m.Called()
	return time.Now().UnixNano()
}








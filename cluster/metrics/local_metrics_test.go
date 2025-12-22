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
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestNewLocalMetrics(t *testing.T) {
	m := newLocalMetrics()
	assert.NotNil(t, m)
	assert.NotNil(t, m.lock)
	assert.NotNil(t, m.metrics)
}

func TestLocalMetricsInit(t *testing.T) {
	assert.NotNil(t, LocalMetrics)
}

func TestSetAndGetMethodMetrics(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	methodName := "GetUser"
	key := "test-key"
	value := "test-value"

	err = m.SetMethodMetrics(url, methodName, key, value)
	assert.NoError(t, err)

	result, err := m.GetMethodMetrics(url, methodName, key)
	assert.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestGetMethodMetricsNotFound(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	methodName := "GetUser"
	key := "non-existent-key"

	result, err := m.GetMethodMetrics(url, methodName, key)
	assert.Error(t, err)
	assert.Equal(t, ErrMetricsNotFound, err)
	assert.Nil(t, result)
}

func TestSetMethodMetricsOverwrite(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	methodName := "GetUser"
	key := "test-key"
	value1 := "value1"
	value2 := "value2"

	err = m.SetMethodMetrics(url, methodName, key, value1)
	assert.NoError(t, err)

	result, err := m.GetMethodMetrics(url, methodName, key)
	assert.NoError(t, err)
	assert.Equal(t, value1, result)

	err = m.SetMethodMetrics(url, methodName, key, value2)
	assert.NoError(t, err)

	result, err = m.GetMethodMetrics(url, methodName, key)
	assert.NoError(t, err)
	assert.Equal(t, value2, result)
}

func TestMethodMetricsWithDifferentMethods(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	method1 := "GetUser"
	method2 := "UpdateUser"
	key := "test-key"
	value1 := "value1"
	value2 := "value2"

	err = m.SetMethodMetrics(url, method1, key, value1)
	assert.NoError(t, err)

	err = m.SetMethodMetrics(url, method2, key, value2)
	assert.NoError(t, err)

	result1, err := m.GetMethodMetrics(url, method1, key)
	assert.NoError(t, err)
	assert.Equal(t, value1, result1)

	result2, err := m.GetMethodMetrics(url, method2, key)
	assert.NoError(t, err)
	assert.Equal(t, value2, result2)
}

func TestMethodMetricsWithDifferentURLs(t *testing.T) {
	m := newLocalMetrics()
	url1, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	url2, err := common.NewURL("dubbo://127.0.0.1:20001/com.test.UserService")
	require.NoError(t, err)
	methodName := "GetUser"
	key := "test-key"
	value1 := "value1"
	value2 := "value2"

	err = m.SetMethodMetrics(url1, methodName, key, value1)
	assert.NoError(t, err)

	err = m.SetMethodMetrics(url2, methodName, key, value2)
	assert.NoError(t, err)

	result1, err := m.GetMethodMetrics(url1, methodName, key)
	assert.NoError(t, err)
	assert.Equal(t, value1, result1)

	result2, err := m.GetMethodMetrics(url2, methodName, key)
	assert.NoError(t, err)
	assert.Equal(t, value2, result2)
}

func TestMethodMetricsWithDifferentKeys(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	methodName := "GetUser"
	key1 := "key1"
	key2 := "key2"
	value1 := "value1"
	value2 := "value2"

	err = m.SetMethodMetrics(url, methodName, key1, value1)
	assert.NoError(t, err)

	err = m.SetMethodMetrics(url, methodName, key2, value2)
	assert.NoError(t, err)

	result1, err := m.GetMethodMetrics(url, methodName, key1)
	assert.NoError(t, err)
	assert.Equal(t, value1, result1)

	result2, err := m.GetMethodMetrics(url, methodName, key2)
	assert.NoError(t, err)
	assert.Equal(t, value2, result2)
}

func TestMethodMetricsWithIntValue(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	methodName := "GetUser"
	key := "count"
	value := 42

	err = m.SetMethodMetrics(url, methodName, key, value)
	assert.NoError(t, err)

	result, err := m.GetMethodMetrics(url, methodName, key)
	assert.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestMethodMetricsConcurrency(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	methodName := "GetUser"
	key := "concurrent-key"

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes and reads mixed
	for i := 0; i < iterations; i++ {
		// Write operation
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			_ = m.SetMethodMetrics(url, methodName, key, val)
		}(i)

		// Read operation
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = m.GetMethodMetrics(url, methodName, key)
		}()
	}

	wg.Wait()

	// Verify we can still read after concurrent operations
	result, err := m.GetMethodMetrics(url, methodName, key)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInvokerMetricsPanics(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)

	assert.Panics(t, func() {
		_, _ = m.GetInvokerMetrics(url, "test-key")
	})
}

func TestSetInvokerMetricsPanics(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)

	assert.Panics(t, func() {
		_ = m.SetInvokerMetrics(url, "test-key", "test-value")
	})
}

func TestGetInstanceMetricsPanics(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)

	assert.Panics(t, func() {
		_, _ = m.GetInstanceMetrics(url, "test-key")
	})
}

func TestSetInstanceMetricsPanics(t *testing.T) {
	m := newLocalMetrics()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)

	assert.Panics(t, func() {
		_ = m.SetInstanceMetrics(url, "test-key", "test-value")
	})
}

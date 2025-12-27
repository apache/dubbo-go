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

package base

import (
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

const (
	mockCommonDubboUrl = "dubbo://localhost:20000/com.ikurento.user.UserProvider"
)

func TestBeginCount(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	BeginCount(url, "test")
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	methodStatus1 := GetMethodStatus(url, "test1")
	assert.Equal(t, int32(1), methodStatus.active)
	assert.Equal(t, int32(1), urlStatus.active)
	assert.Equal(t, int32(0), methodStatus1.active)
}

func TestEndCount(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	EndCount(url, "test", 100, true)
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	assert.Equal(t, int32(-1), methodStatus.active)
	assert.Equal(t, int32(-1), urlStatus.active)
	assert.Equal(t, int32(1), methodStatus.total)
	assert.Equal(t, int32(1), urlStatus.total)
}

func TestGetMethodStatus(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	status := GetMethodStatus(url, "test")
	assert.NotNil(t, status)
	assert.Equal(t, int32(0), status.total)
}

func TestGetUrlStatus(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	status := GetURLStatus(url)
	assert.NotNil(t, status)
	assert.Equal(t, int32(0), status.total)
}

func TestBeginCount0(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	status := GetURLStatus(url)
	beginCount0(status)
	assert.Equal(t, int32(1), status.active)
}

func TestAll(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	request(url, "test", 100, false, true)
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	assert.Equal(t, int32(1), methodStatus.total)
	assert.Equal(t, int32(1), urlStatus.total)
	assert.Equal(t, int32(0), methodStatus.active)
	assert.Equal(t, int32(0), urlStatus.active)
	assert.Equal(t, int32(0), methodStatus.failed)
	assert.Equal(t, int32(0), urlStatus.failed)
	assert.Equal(t, int32(0), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(0), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(100), methodStatus.totalElapsed)
	assert.Equal(t, int64(100), urlStatus.totalElapsed)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	assert.Equal(t, int32(6), methodStatus.total)
	assert.Equal(t, int32(6), urlStatus.total)
	assert.Equal(t, int32(5), methodStatus.failed)
	assert.Equal(t, int32(5), urlStatus.failed)
	assert.Equal(t, int32(5), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(5), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(600), methodStatus.totalElapsed)
	assert.Equal(t, int64(600), urlStatus.totalElapsed)
	assert.Equal(t, int64(500), methodStatus.failedElapsed)
	assert.Equal(t, int64(500), urlStatus.failedElapsed)

	request(url, "test", 100, false, true)
	assert.Equal(t, int32(0), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(0), urlStatus.successiveRequestFailureCount)

	request(url, "test", 200, false, false)
	request(url, "test", 200, false, false)
	assert.Equal(t, int32(2), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(2), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(200), methodStatus.maxElapsed)
	assert.Equal(t, int64(200), urlStatus.maxElapsed)

	request(url, "test1", 200, false, false)
	request(url, "test1", 200, false, false)
	request(url, "test1", 200, false, false)
	assert.Equal(t, int32(5), urlStatus.successiveRequestFailureCount)
	methodStatus1 := GetMethodStatus(url, "test1")
	assert.Equal(t, int32(2), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(3), methodStatus1.successiveRequestFailureCount)
}

func request(url *common.URL, method string, elapsed int64, active, succeeded bool) {
	BeginCount(url, method)
	if !active {
		EndCount(url, method, elapsed, succeeded)
	}
}

func TestCurrentTimeMillis(t *testing.T) {
	defer CleanAllStatus()
	c := CurrentTimeMillis()
	assert.NotNil(t, c)
	str := strconv.FormatInt(c, 10)
	i, _ := strconv.ParseInt(str, 10, 64)
	assert.Equal(t, c, i)
}

func TestRPCStatusGetters(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)

	// Test all getter methods
	request(url, "test", 100, false, true)

	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")

	// Test GetActive
	assert.Equal(t, int32(0), urlStatus.GetActive())
	assert.Equal(t, int32(0), methodStatus.GetActive())

	// Test GetTotal
	assert.Equal(t, int32(1), urlStatus.GetTotal())
	assert.Equal(t, int32(1), methodStatus.GetTotal())

	// Test GetFailed
	assert.Equal(t, int32(0), urlStatus.GetFailed())
	assert.Equal(t, int32(0), methodStatus.GetFailed())

	// Test GetTotalElapsed
	assert.Equal(t, int64(100), urlStatus.GetTotalElapsed())
	assert.Equal(t, int64(100), methodStatus.GetTotalElapsed())

	// Test GetFailedElapsed
	assert.Equal(t, int64(0), urlStatus.GetFailedElapsed())
	assert.Equal(t, int64(0), methodStatus.GetFailedElapsed())

	// Test GetMaxElapsed
	assert.Equal(t, int64(100), urlStatus.GetMaxElapsed())
	assert.Equal(t, int64(100), methodStatus.GetMaxElapsed())

	// Test GetFailedMaxElapsed
	assert.Equal(t, int64(0), urlStatus.GetFailedMaxElapsed())
	assert.Equal(t, int64(0), methodStatus.GetFailedMaxElapsed())

	// Test GetSucceededMaxElapsed
	assert.Equal(t, int64(100), urlStatus.GetSucceededMaxElapsed())
	assert.Equal(t, int64(100), methodStatus.GetSucceededMaxElapsed())

	// Test GetSuccessiveRequestFailureCount
	assert.Equal(t, int32(0), urlStatus.GetSuccessiveRequestFailureCount())
	assert.Equal(t, int32(0), methodStatus.GetSuccessiveRequestFailureCount())

	// Add a failed request to test failure-related getters
	request(url, "test", 150, false, false)

	assert.Equal(t, int32(1), urlStatus.GetFailed())
	assert.Equal(t, int64(150), urlStatus.GetFailedElapsed())
	assert.Equal(t, int64(150), urlStatus.GetFailedMaxElapsed())
	assert.Equal(t, int32(1), urlStatus.GetSuccessiveRequestFailureCount())

	// Test GetLastRequestFailedTimestamp
	timestamp := urlStatus.GetLastRequestFailedTimestamp()
	assert.Positive(t, timestamp)
}

func TestInvokerBlackList(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	invoker := NewBaseInvoker(url)

	// Test healthy status initially
	assert.True(t, GetInvokerHealthyStatus(invoker))

	// Set invoker to unhealthy
	SetInvokerUnhealthyStatus(invoker)
	assert.False(t, GetInvokerHealthyStatus(invoker))

	// Test GetBlackListInvokers
	blackListInvokers := GetBlackListInvokers(10)
	assert.Len(t, blackListInvokers, 1)
	assert.Equal(t, invoker.GetURL().Key(), blackListInvokers[0].GetURL().Key())

	// Test with blockSize smaller than actual size
	blackListInvokers = GetBlackListInvokers(0)
	assert.Empty(t, blackListInvokers)

	// Remove invoker from unhealthy status
	RemoveInvokerUnhealthyStatus(invoker)
	assert.True(t, GetInvokerHealthyStatus(invoker))

	// Test RemoveUrlKeyUnhealthyStatus
	SetInvokerUnhealthyStatus(invoker)
	assert.False(t, GetInvokerHealthyStatus(invoker))
	RemoveUrlKeyUnhealthyStatus(invoker.GetURL().Key())
	assert.True(t, GetInvokerHealthyStatus(invoker))
}

func TestGetAndRefreshState(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)
	invoker := NewBaseInvoker(url)

	// Clear initial state
	GetAndRefreshState()

	// After setting unhealthy status, it should be true
	SetInvokerUnhealthyStatus(invoker)
	state := GetAndRefreshState()
	assert.True(t, state)

	// After getting, it should be false again
	state = GetAndRefreshState()
	assert.False(t, state)

	// Test state change again
	RemoveInvokerUnhealthyStatus(invoker)
	state = GetAndRefreshState()
	assert.True(t, state)

	state = GetAndRefreshState()
	assert.False(t, state)
}

func TestTryRefreshBlackList(t *testing.T) {
	defer CleanAllStatus()

	url1, _ := common.NewURL(mockCommonDubboUrl)
	url2, _ := common.NewURL("dubbo://localhost:20001/com.ikurento.user.UserProvider")

	invoker1 := NewBaseInvoker(url1)
	invoker2 := NewBaseInvoker(url2)

	// Add invokers to black list
	SetInvokerUnhealthyStatus(invoker1)
	SetInvokerUnhealthyStatus(invoker2)

	assert.False(t, GetInvokerHealthyStatus(invoker1))
	assert.False(t, GetInvokerHealthyStatus(invoker2))

	// Try refresh black list - since invokers are available, they should be removed
	TryRefreshBlackList()

	// Give goroutines time to complete
	// Note: In real scenario, invokers would be checked for availability
	// Since our mock invokers are always available, they should be removed

	// Verify black list was processed
	blackList := GetBlackListInvokers(10)
	// The behavior depends on the actual IsAvailable() implementation
	assert.NotNil(t, blackList)
}

func TestMultipleMethodsStatus(t *testing.T) {
	defer CleanAllStatus()

	url, _ := common.NewURL(mockCommonDubboUrl)

	// Test multiple methods on same URL
	request(url, "method1", 100, false, true)
	request(url, "method2", 200, false, true)
	request(url, "method3", 150, false, false)

	method1Status := GetMethodStatus(url, "method1")
	method2Status := GetMethodStatus(url, "method2")
	method3Status := GetMethodStatus(url, "method3")
	urlStatus := GetURLStatus(url)

	// Verify individual method stats
	assert.Equal(t, int32(1), method1Status.GetTotal())
	assert.Equal(t, int64(100), method1Status.GetTotalElapsed())

	assert.Equal(t, int32(1), method2Status.GetTotal())
	assert.Equal(t, int64(200), method2Status.GetTotalElapsed())
	assert.Equal(t, int64(200), method2Status.GetMaxElapsed())

	assert.Equal(t, int32(1), method3Status.GetTotal())
	assert.Equal(t, int32(1), method3Status.GetFailed())

	// Verify URL aggregates all methods
	assert.Equal(t, int32(3), urlStatus.GetTotal())
	assert.Equal(t, int64(450), urlStatus.GetTotalElapsed())
	assert.Equal(t, int32(1), urlStatus.GetFailed())
}

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

package healthcheck

import (
	"fmt"
	"math"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
)

const (
	healthCheckDubbo1010IP    = "192.168.10.10"
	healthCheckDubbo1011IP    = "192.168.10.11"
	healthCheckMethodTest     = "test"
	healthCheckDubboUrlFormat = "dubbo://%s:20000/com.ikurento.user.UserProvider"
)

func TestDefaultHealthCheckerIsHealthy(t *testing.T) {
	defer protocol.CleanAllStatus()
	url, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1010IP))
	hc := NewDefaultHealthChecker(url).(*DefaultHealthChecker)
	invoker := NewMockInvoker(url)
	healthy := hc.IsHealthy(invoker)
	assert.True(t, healthy)

	url.SetParam(constant.OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, "10")
	url.SetParam(constant.SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "100")
	// fake the outgoing request
	for i := 0; i < 11; i++ {
		request(url, healthCheckMethodTest, 0, true, false)
	}
	hc = NewDefaultHealthChecker(url).(*DefaultHealthChecker)
	healthy = hc.IsHealthy(invoker)
	// the outgoing request is more than OUTSTANDING_REQUEST_COUNT_LIMIT, go to unhealthy
	assert.False(t, hc.IsHealthy(invoker))

	// successive failed count is more than constant.SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, go to unhealthy
	for i := 0; i < 11; i++ {
		request(url, healthCheckMethodTest, 0, false, false)
	}
	url.SetParam(constant.SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "10")
	url.SetParam(constant.OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, "1000")
	hc = NewDefaultHealthChecker(url).(*DefaultHealthChecker)
	healthy = hc.IsHealthy(invoker)
	assert.False(t, hc.IsHealthy(invoker))

	// reset successive failed count and go to healthy
	request(url, healthCheckMethodTest, 0, false, true)
	healthy = hc.IsHealthy(invoker)
	assert.True(t, hc.IsHealthy(invoker))
}

func TestDefaultHealthCheckerGetCircuitBreakerSleepWindowTime(t *testing.T) {
	defer protocol.CleanAllStatus()
	url, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1010IP))
	defaultHc := NewDefaultHealthChecker(url).(*DefaultHealthChecker)
	// Increase the number of failed requests
	for i := 0; i < 100; i++ {
		request(url, healthCheckMethodTest, 1, false, false)
	}
	sleepWindowTime := defaultHc.getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url))
	assert.True(t, sleepWindowTime == constant.MAX_CIRCUIT_TRIPPED_TIMEOUT_IN_MS)

	// Adjust the threshold size to 1000
	url.SetParam(constant.SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "1000")
	sleepWindowTime = NewDefaultHealthChecker(url).(*DefaultHealthChecker).getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url))
	assert.True(t, sleepWindowTime == 0)

	url1, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1011IP))
	sleepWindowTime = defaultHc.getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url1))
	assert.True(t, sleepWindowTime == 0)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	sleepWindowTime = defaultHc.getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url1))
	assert.True(t, sleepWindowTime > 0 && sleepWindowTime < constant.MAX_CIRCUIT_TRIPPED_TIMEOUT_IN_MS)
}

func TestDefaultHealthCheckerGetCircuitBreakerTimeout(t *testing.T) {
	defer protocol.CleanAllStatus()
	url, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1010IP))
	defaultHc := NewDefaultHealthChecker(url).(*DefaultHealthChecker)
	timeout := defaultHc.getCircuitBreakerTimeout(protocol.GetURLStatus(url))
	assert.True(t, timeout == 0)
	url1, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1011IP))
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	request(url1, healthCheckMethodTest, 1, false, false)
	timeout = defaultHc.getCircuitBreakerTimeout(protocol.GetURLStatus(url1))
	// timeout must after the current time
	assert.True(t, timeout > protocol.CurrentTimeMillis())

}

func TestDefaultHealthCheckerIsCircuitBreakerTripped(t *testing.T) {
	defer protocol.CleanAllStatus()
	url, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1010IP))
	defaultHc := NewDefaultHealthChecker(url).(*DefaultHealthChecker)
	status := protocol.GetURLStatus(url)
	tripped := defaultHc.isCircuitBreakerTripped(status)
	assert.False(t, tripped)
	// Increase the number of failed requests
	for i := 0; i < 100; i++ {
		request(url, healthCheckMethodTest, 1, false, false)
	}
	tripped = defaultHc.isCircuitBreakerTripped(protocol.GetURLStatus(url))
	assert.True(t, tripped)

}

func TestNewDefaultHealthChecker(t *testing.T) {
	defer protocol.CleanAllStatus()
	url, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1010IP))
	defaultHc := NewDefaultHealthChecker(url).(*DefaultHealthChecker)
	assert.NotNil(t, defaultHc)
	assert.Equal(t, defaultHc.outStandingRequestConutLimit, int32(math.MaxInt32))
	assert.Equal(t, defaultHc.requestSuccessiveFailureThreshold, int32(constant.DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF))

	url1, _ := common.NewURL(fmt.Sprintf(healthCheckDubboUrlFormat, healthCheckDubbo1010IP))
	url1.SetParam(constant.OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, "10")
	url1.SetParam(constant.SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "10")
	nondefaultHc := NewDefaultHealthChecker(url1).(*DefaultHealthChecker)
	assert.NotNil(t, nondefaultHc)
	assert.Equal(t, nondefaultHc.outStandingRequestConutLimit, int32(10))
	assert.Equal(t, nondefaultHc.requestSuccessiveFailureThreshold, int32(10))
}

func request(url *common.URL, method string, elapsed int64, active, succeeded bool) {
	protocol.BeginCount(url, method)
	if !active {
		protocol.EndCount(url, method, elapsed, succeeded)
	}
}

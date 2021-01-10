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
	"math"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

func init() {
	extension.SetHealthChecker(constant.DEFAULT_HEALTH_CHECKER, NewDefaultHealthChecker)
}

// DefaultHealthChecker is the default implementation of HealthChecker, which determines the health status of
// the invoker based on the number of successive bad request and the current active request.
type DefaultHealthChecker struct {
	// the limit of outstanding request
	outStandingRequestConutLimit int32
	// the threshold of successive-failure-request
	requestSuccessiveFailureThreshold int32
	// value of circuit-tripped timeout factor
	circuitTrippedTimeoutFactor int32
}

// IsHealthy evaluates the healthy state on the given Invoker based on the number of successive bad request
// and the current active request
func (c *DefaultHealthChecker) IsHealthy(invoker protocol.Invoker) bool {
	if !invoker.IsAvailable() {
		return false
	}

	urlStatus := protocol.GetURLStatus(invoker.GetUrl())
	if c.isCircuitBreakerTripped(urlStatus) || urlStatus.GetActive() > c.GetOutStandingRequestCountLimit() {
		logger.Debugf("Invoker [%s] is currently in circuitbreaker tripped state", invoker.GetUrl().Key())
		return false
	}
	return true
}

// isCircuitBreakerTripped determine whether the invoker is in the tripped state by the number of successive bad request
func (c *DefaultHealthChecker) isCircuitBreakerTripped(status *protocol.RPCStatus) bool {
	circuitBreakerTimeout := c.getCircuitBreakerTimeout(status)
	currentTime := protocol.CurrentTimeMillis()
	if circuitBreakerTimeout <= 0 {
		return false
	}
	return circuitBreakerTimeout > currentTime
}

// getCircuitBreakerTimeout get the timestamp recovered from tripped state, the unit is millisecond
func (c *DefaultHealthChecker) getCircuitBreakerTimeout(status *protocol.RPCStatus) int64 {
	sleepWindow := c.getCircuitBreakerSleepWindowTime(status)
	if sleepWindow <= 0 {
		return 0
	}
	return status.GetLastRequestFailedTimestamp() + sleepWindow
}

// getCircuitBreakerSleepWindowTime get the sleep window time of invoker, the unit is millisecond
func (c *DefaultHealthChecker) getCircuitBreakerSleepWindowTime(status *protocol.RPCStatus) int64 {

	successiveFailureCount := status.GetSuccessiveRequestFailureCount()
	diff := successiveFailureCount - c.GetRequestSuccessiveFailureThreshold()
	if diff < 0 {
		return 0
	} else if diff > constant.DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF {
		diff = constant.DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF
	}
	sleepWindow := (1 << uint(diff)) * c.GetCircuitTrippedTimeoutFactor()
	if sleepWindow > constant.MAX_CIRCUIT_TRIPPED_TIMEOUT_IN_MS {
		sleepWindow = constant.MAX_CIRCUIT_TRIPPED_TIMEOUT_IN_MS
	}
	return int64(sleepWindow)
}

// GetRequestSuccessiveFailureThreshold return the requestSuccessiveFailureThreshold bound to this DefaultHealthChecker
func (c *DefaultHealthChecker) GetRequestSuccessiveFailureThreshold() int32 {
	return c.requestSuccessiveFailureThreshold
}

// GetCircuitTrippedTimeoutFactor return the circuitTrippedTimeoutFactor bound to this DefaultHealthChecker
func (c *DefaultHealthChecker) GetCircuitTrippedTimeoutFactor() int32 {
	return c.circuitTrippedTimeoutFactor
}

// GetOutStandingRequestCountLimit return the outStandingRequestConutLimit bound to this DefaultHealthChecker
func (c *DefaultHealthChecker) GetOutStandingRequestCountLimit() int32 {
	return c.outStandingRequestConutLimit
}

// NewDefaultHealthChecker constructs a new DefaultHealthChecker based on the url
func NewDefaultHealthChecker(url *common.URL) router.HealthChecker {
	return &DefaultHealthChecker{
		outStandingRequestConutLimit:      url.GetParamInt32(constant.OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, math.MaxInt32),
		requestSuccessiveFailureThreshold: url.GetParamInt32(constant.SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, constant.DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF),
		circuitTrippedTimeoutFactor:       url.GetParamInt32(constant.CIRCUIT_TRIPPED_TIMEOUT_FACTOR_KEY, constant.DEFAULT_CIRCUIT_TRIPPED_TIMEOUT_FACTOR),
	}
}

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

package connection

import (
	"context"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

// TripleHealthChecker implements HealthChecker for Triple protocol (gRPC-based)
// This extends the Issue #1868 solution to Triple protocol
type TripleHealthChecker struct {
	checkInterval time.Duration
	maxIdleTime   time.Duration
}

// NewTripleHealthChecker creates a new Triple protocol health checker
func NewTripleHealthChecker() *TripleHealthChecker {
	return &TripleHealthChecker{
		checkInterval: 30 * time.Second,
		maxIdleTime:   5 * time.Minute,
	}
}

// CheckConnection performs health check on a Triple connection
// Triple protocol benefits from gRPC's built-in connectivity state management
func (thc *TripleHealthChecker) CheckConnection(ctx context.Context, conn Connection) *HealthCheckResult {
	startTime := time.Now()

	result := &HealthCheckResult{
		Healthy:   true,
		CheckTime: startTime,
	}

	//  Basic state check
	state := conn.GetState()
	if state == StateShutdown || state == StateTransientFailure {
		result.Healthy = false
		result.Reason = "connection state is " + state.String()
		result.Duration = time.Since(startTime)
		return result
	}

	//  Availability check using the connection's own IsAvailable method
	// For Triple protocol, this typically checks gRPC connectivity state
	if !conn.IsAvailable() {
		result.Healthy = false
		result.Reason = "connection reports not available"
		result.Duration = time.Since(startTime)
		return result
	}

	// Idle time check
	lastActive := conn.GetLastActive()
	if !lastActive.IsZero() {
		idleTime := time.Since(lastActive)
		if idleTime > thc.maxIdleTime {
			result.Healthy = false
			result.Reason = "connection idle too long: " + idleTime.String()
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// For Triple protocol, we can leverage gRPC's robust connectivity states
	// gRPC has excellent built-in health checking via HTTP/2 PING frames
	// and connectivity state management (IDLE, CONNECTING, READY, TRANSIENT_FAILURE, SHUTDOWN)

	result.Duration = time.Since(startTime)

	if result.Healthy {
		logger.Debugf("Triple connection health check passed for %s (took %v)",
			conn.GetURL().Location, result.Duration)
	} else {
		logger.Debugf("Triple connection health check failed for %s: %s (took %v)",
			conn.GetURL().Location, result.Reason, result.Duration)
	}

	return result
}

// GetProtocol returns the protocol this health checker supports
func (thc *TripleHealthChecker) GetProtocol() string {
	return "triple"
}

// GetCheckInterval returns the recommended check interval
func (thc *TripleHealthChecker) GetCheckInterval() time.Duration {
	return thc.checkInterval
}

// SetCheckInterval sets the health check interval
func (thc *TripleHealthChecker) SetCheckInterval(interval time.Duration) {
	thc.checkInterval = interval
}

// SetMaxIdleTime sets the maximum idle time before considering connection stale
func (thc *TripleHealthChecker) SetMaxIdleTime(maxIdleTime time.Duration) {
	thc.maxIdleTime = maxIdleTime
}

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

// DubboHealthChecker implements HealthChecker for Dubbo protocol
// This addresses Issue #1868 by providing proper connection health detection
// for Dubbo protocol without the circular dependency issues
type DubboHealthChecker struct {
	checkInterval time.Duration
	maxIdleTime   time.Duration
}

// NewDubboHealthChecker creates a new Dubbo protocol health checker
func NewDubboHealthChecker() *DubboHealthChecker {
	return &DubboHealthChecker{
		checkInterval: 30 * time.Second,
		maxIdleTime:   5 * time.Minute,
	}
}

// CheckConnection performs health check on a Dubbo connection
// This method solves the core Issue #1868 problem by checking connection health
// without using the potentially stale connection itself (avoiding circular dependency)
func (dhc *DubboHealthChecker) CheckConnection(ctx context.Context, conn Connection) *HealthCheckResult {
	startTime := time.Now()

	result := &HealthCheckResult{
		Healthy:   true,
		CheckTime: startTime,
	}

	// Basic state check
	state := conn.GetState()
	if state == StateShutdown || state == StateTransientFailure {
		result.Healthy = false
		result.Reason = "connection state is " + state.String()
		result.Duration = time.Since(startTime)
		return result
	}

	//  Availability check using the connection's own IsAvailable method
	// This delegates to the underlying implementation (Getty) which knows
	// how to check session state without network I/O
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
		if idleTime > dhc.maxIdleTime {
			result.Healthy = false
			result.Reason = "connection idle too long: " + idleTime.String()
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// For Dubbo protocol, we trust the Getty session state
	// Getty's internal session management is more reliable than
	// sending network requests which could fail due to the very
	// connection issues we're trying to detect

	result.Duration = time.Since(startTime)

	if result.Healthy {
		logger.Debugf("Dubbo connection health check passed for %s (took %v)",
			conn.GetURL().Location, result.Duration)
	} else {
		logger.Debugf("Dubbo connection health check failed for %s: %s (took %v)",
			conn.GetURL().Location, result.Reason, result.Duration)
	}

	return result
}

// GetProtocol returns the protocol this health checker supports
func (dhc *DubboHealthChecker) GetProtocol() string {
	return "dubbo"
}

// GetCheckInterval returns the recommended check interval
func (dhc *DubboHealthChecker) GetCheckInterval() time.Duration {
	return dhc.checkInterval
}

// SetCheckInterval sets the health check interval
func (dhc *DubboHealthChecker) SetCheckInterval(interval time.Duration) {
	dhc.checkInterval = interval
}

// SetMaxIdleTime sets the maximum idle time before considering connection stale
func (dhc *DubboHealthChecker) SetMaxIdleTime(maxIdleTime time.Duration) {
	dhc.maxIdleTime = maxIdleTime
}

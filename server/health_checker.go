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

package server

import (
	"context"
	"net/http"
	"sync"
)

// HealthChecker defines the interface for customizable health check logic
// Users can implement this interface to define their own health criteria
type HealthChecker interface {
	// CheckLiveness checks if the application is alive
	// Return false only for serious internal errors that require container restart
	CheckLiveness(ctx context.Context) bool

	// CheckReadiness checks if the application is ready to serve traffic
	// Return false if external dependencies are unavailable
	CheckReadiness(ctx context.Context) bool

	// Name returns the identifier of this health checker
	Name() string
}

// HealthCheckResult provides detailed health check information
type HealthCheckResult struct {
	Healthy bool              // Overall health status
	Message string            // Optional descriptive message
	Details map[string]string // Detailed check results
}

// DetailedHealthChecker extends HealthChecker with detailed result information
type DetailedHealthChecker interface {
	HealthChecker

	// CheckLivenessDetailed returns detailed liveness check result
	CheckLivenessDetailed(ctx context.Context) *HealthCheckResult

	// CheckReadinessDetailed returns detailed readiness check result
	CheckReadinessDetailed(ctx context.Context) *HealthCheckResult
}

// HealthCheckHandler allows customizing HTTP response handling
type HealthCheckHandler interface {
	// HandleLivenessCheck processes liveness probe HTTP request
	HandleLivenessCheck(w http.ResponseWriter, r *http.Request, result *HealthCheckResult)

	// HandleReadinessCheck processes readiness probe HTTP request
	HandleReadinessCheck(w http.ResponseWriter, r *http.Request, result *HealthCheckResult)
}

// CompositeHealthChecker combines multiple health checkers
// All checkers must pass for the composite to be healthy
type CompositeHealthChecker struct {
	checkers []HealthChecker
}

// NewCompositeHealthChecker creates a composite health checker
func NewCompositeHealthChecker(checkers ...HealthChecker) *CompositeHealthChecker {
	return &CompositeHealthChecker{checkers: checkers}
}

func (c *CompositeHealthChecker) CheckLiveness(ctx context.Context) bool {
	for _, checker := range c.checkers {
		if !checker.CheckLiveness(ctx) {
			return false
		}
	}
	return len(c.checkers) > 0
}

func (c *CompositeHealthChecker) CheckReadiness(ctx context.Context) bool {
	for _, checker := range c.checkers {
		if !checker.CheckReadiness(ctx) {
			return false
		}
	}
	return len(c.checkers) > 0
}

func (c *CompositeHealthChecker) CheckLivenessDetailed(ctx context.Context) *HealthCheckResult {
	result := &HealthCheckResult{
		Healthy: true,
		Details: make(map[string]string),
	}

	for _, checker := range c.checkers {
		if detailedChecker, ok := checker.(DetailedHealthChecker); ok {
			subResult := detailedChecker.CheckLivenessDetailed(ctx)
			if !subResult.Healthy {
				result.Healthy = false
				result.Message = subResult.Message
			}
			// Merge details
			for k, v := range subResult.Details {
				result.Details[checker.Name()+"_"+k] = v
			}
		} else {
			healthy := checker.CheckLiveness(ctx)
			result.Details[checker.Name()] = map[bool]string{true: "UP", false: "DOWN"}[healthy]
			if !healthy {
				result.Healthy = false
				result.Message = checker.Name() + " liveness check failed"
			}
		}
	}

	return result
}

func (c *CompositeHealthChecker) CheckReadinessDetailed(ctx context.Context) *HealthCheckResult {
	result := &HealthCheckResult{
		Healthy: true,
		Details: make(map[string]string),
	}

	for _, checker := range c.checkers {
		if detailedChecker, ok := checker.(DetailedHealthChecker); ok {
			subResult := detailedChecker.CheckReadinessDetailed(ctx)
			if !subResult.Healthy {
				result.Healthy = false
				result.Message = subResult.Message
			}
			// Merge details
			for k, v := range subResult.Details {
				result.Details[checker.Name()+"_"+k] = v
			}
		} else {
			healthy := checker.CheckReadiness(ctx)
			result.Details[checker.Name()] = map[bool]string{true: "UP", false: "DOWN"}[healthy]
			if !healthy {
				result.Healthy = false
				result.Message = checker.Name() + " readiness check failed"
			}
		}
	}

	return result
}

func (c *CompositeHealthChecker) Name() string {
	return "CompositeHealthChecker"
}

// DefaultHealthChecker provides basic implementation
type DefaultHealthChecker struct{}

func (d *DefaultHealthChecker) CheckLiveness(ctx context.Context) bool {
	// Basic liveness: process is running
	return true
}

func (d *DefaultHealthChecker) CheckReadiness(ctx context.Context) bool {
	// Basic readiness: assume ready
	return true
}

func (d *DefaultHealthChecker) Name() string {
	return "DefaultHealthChecker"
}

// Global health checker registry
var (
	globalHealthChecker HealthChecker      = &DefaultHealthChecker{}
	globalHealthHandler HealthCheckHandler = &DefaultHealthCheckHandler{}
	healthCheckerMu     sync.RWMutex
)

// SetHealthChecker sets the global health checker
func SetHealthChecker(checker HealthChecker) {
	healthCheckerMu.Lock()
	defer healthCheckerMu.Unlock()
	globalHealthChecker = checker
}

// SetHealthCheckHandler sets the global health check handler
func SetHealthCheckHandler(handler HealthCheckHandler) {
	healthCheckerMu.Lock()
	defer healthCheckerMu.Unlock()
	globalHealthHandler = handler
}

// GetHealthChecker gets the current global health checker
func GetHealthChecker() HealthChecker {
	healthCheckerMu.RLock()
	defer healthCheckerMu.RUnlock()
	return globalHealthChecker
}

// GetHealthCheckHandler gets the current global health check handler
func GetHealthCheckHandler() HealthCheckHandler {
	healthCheckerMu.RLock()
	defer healthCheckerMu.RUnlock()
	return globalHealthHandler
}

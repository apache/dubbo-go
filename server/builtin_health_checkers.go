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
	"sync/atomic"
	"time"
)

// DubboHealthChecker checks if dubbo services are properly exported
type DubboHealthChecker struct {
	serverInstance *Server
}

// NewDubboHealthChecker creates a health checker for dubbo services
func NewDubboHealthChecker(server *Server) *DubboHealthChecker {
	return &DubboHealthChecker{serverInstance: server}
}

func (d *DubboHealthChecker) CheckLiveness(ctx context.Context) bool {
	// Liveness: server instance exists (process is alive)
	return d.serverInstance != nil
}

func (d *DubboHealthChecker) CheckReadiness(ctx context.Context) bool {
	// Readiness: services are exported and ready to handle requests
	if d.serverInstance == nil {
		return false
	}

	// Check if any services are registered
	hasServices := false
	d.serverInstance.svcOptsMap.Range(func(key, value interface{}) bool {
		hasServices = true
		return false // Stop iteration after finding first service
	})

	return hasServices
}

func (d *DubboHealthChecker) CheckReadinessDetailed(ctx context.Context) *HealthCheckResult {
	result := &HealthCheckResult{
		Healthy: true,
		Details: make(map[string]string),
	}

	if d.serverInstance == nil {
		result.Healthy = false
		result.Message = "Server instance not initialized"
		result.Details["server"] = "not_initialized"
		return result
	}

	// Count exported services
	serviceCount := 0
	d.serverInstance.svcOptsMap.Range(func(key, value interface{}) bool {
		serviceCount++
		return true
	})

	if serviceCount == 0 {
		result.Healthy = false
		result.Message = "No services exported"
		result.Details["services"] = "none_exported"
	} else {
		result.Details["services"] = "exported"
		result.Details["service_count"] = string(rune(serviceCount))
	}

	return result
}

func (d *DubboHealthChecker) Name() string {
	return "DubboHealthChecker"
}

// GracefulShutdownHealthChecker tracks graceful shutdown state
type GracefulShutdownHealthChecker struct {
	shuttingDown int32
}

// NewGracefulShutdownHealthChecker creates a health checker that tracks shutdown state
func NewGracefulShutdownHealthChecker() *GracefulShutdownHealthChecker {
	return &GracefulShutdownHealthChecker{}
}

func (g *GracefulShutdownHealthChecker) CheckLiveness(ctx context.Context) bool {
	return atomic.LoadInt32(&g.shuttingDown) == 0
}

func (g *GracefulShutdownHealthChecker) CheckReadiness(ctx context.Context) bool {
	return atomic.LoadInt32(&g.shuttingDown) == 0
}

func (g *GracefulShutdownHealthChecker) Name() string {
	return "GracefulShutdownHealthChecker"
}

// MarkShuttingDown marks the service as shutting down
func (g *GracefulShutdownHealthChecker) MarkShuttingDown() {
	atomic.StoreInt32(&g.shuttingDown, 1)
}

// TimeoutHealthChecker wraps another checker with configurable timeout
type TimeoutHealthChecker struct {
	checker HealthChecker
	timeout time.Duration
}

// NewTimeoutHealthChecker creates a timeout-protected health checker
func NewTimeoutHealthChecker(checker HealthChecker, timeout time.Duration) *TimeoutHealthChecker {
	return &TimeoutHealthChecker{
		checker: checker,
		timeout: timeout,
	}
}

func (t *TimeoutHealthChecker) CheckLiveness(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		done <- t.checker.CheckLiveness(ctx)
	}()

	select {
	case result := <-done:
		return result
	case <-ctx.Done():
		return false // Timeout considered unhealthy
	}
}

func (t *TimeoutHealthChecker) CheckReadiness(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		done <- t.checker.CheckReadiness(ctx)
	}()

	select {
	case result := <-done:
		return result
	case <-ctx.Done():
		return false // Timeout considered unhealthy
	}
}

func (t *TimeoutHealthChecker) Name() string {
	return "TimeoutHealthChecker(" + t.checker.Name() + ")"
}

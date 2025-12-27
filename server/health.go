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
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

// HealthCheckServer HTTP health check server for Kubernetes integration
type HealthCheckServer struct {
	url     *common.URL
	server  *http.Server
	checker HealthChecker
	handler HealthCheckHandler
	mu      sync.RWMutex
}

// HealthResponse represents the HTTP response for health check endpoints
type HealthResponse struct {
	Status    string            `json:"status"`            // UP or DOWN
	Timestamp int64             `json:"timestamp"`         // Unix timestamp in milliseconds
	Message   string            `json:"message,omitempty"` // Optional descriptive message
	Details   map[string]string `json:"details,omitempty"` // Detailed check results
}

// DefaultHealthCheckHandler provides standard JSON response handling
type DefaultHealthCheckHandler struct{}

func (d *DefaultHealthCheckHandler) HandleLivenessCheck(w http.ResponseWriter, r *http.Request, result *HealthCheckResult) {
	status := "UP"
	httpStatus := http.StatusOK
	if !result.Healthy {
		status = "DOWN"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now().UnixMilli(),
		Message:   result.Message,
		Details:   result.Details,
	}

	if response.Details == nil {
		response.Details = make(map[string]string)
	}
	response.Details["check"] = "liveness"

	writeJSONResponse(w, response, httpStatus)
}

func (d *DefaultHealthCheckHandler) HandleReadinessCheck(w http.ResponseWriter, r *http.Request, result *HealthCheckResult) {
	status := "UP"
	httpStatus := http.StatusOK
	if !result.Healthy {
		status = "DOWN"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now().UnixMilli(),
		Message:   result.Message,
		Details:   result.Details,
	}

	if response.Details == nil {
		response.Details = make(map[string]string)
	}
	response.Details["check"] = "readiness"

	writeJSONResponse(w, response, httpStatus)
}

// NewHealthCheckServer creates a new health check server
func NewHealthCheckServer(url *common.URL) *HealthCheckServer {
	return &HealthCheckServer{
		url:     url,
		checker: GetHealthChecker(),
		handler: GetHealthCheckHandler(),
	}
}

// SetHealthChecker sets the health checker for this server instance
func (h *HealthCheckServer) SetHealthChecker(checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checker = checker
}

// SetHealthCheckHandler sets the health check handler for this server instance
func (h *HealthCheckServer) SetHealthCheckHandler(handler HealthCheckHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handler = handler
}

// Start starts the health check server if enabled
func (h *HealthCheckServer) Start() {
	if !h.url.GetParamBool(constant.HealthCheckEnabledKey, false) {
		return
	}

	mux := http.NewServeMux()
	port := h.url.GetParam(constant.HealthCheckPortKey, constant.HealthCheckDefaultPort)
	livePath := h.url.GetParam(constant.HealthCheckLivePathKey, constant.HealthCheckDefaultLivePath)
	readyPath := h.url.GetParam(constant.HealthCheckReadyPathKey, constant.HealthCheckDefaultReadyPath)

	mux.HandleFunc(livePath, h.livenessHandler)
	mux.HandleFunc(readyPath, h.readinessHandler)

	h.server = &http.Server{Addr: ":" + port, Handler: mux}

	// Register graceful shutdown callback
	extension.AddCustomShutdownCallback(func() {
		timeoutStr := h.url.GetParam(constant.HealthCheckTimeoutKey, constant.HealthCheckDefaultTimeout)
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			timeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := h.server.Shutdown(ctx); err != nil {
			logger.Errorf("health check server shutdown failed, err: %v", err)
		} else {
			logger.Info("health check server gracefully shutdown success")
		}
	})

	logger.Infof("health check endpoints started - liveness: :%s%s, readiness: :%s%s",
		port, livePath, port, readyPath)

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("health check server error: %v", err)
		}
	}()
}

// livenessHandler handles Kubernetes liveness probe requests
func (h *HealthCheckServer) livenessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	checker := h.checker
	handler := h.handler
	h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var result *HealthCheckResult
	if detailedChecker, ok := checker.(DetailedHealthChecker); ok {
		result = detailedChecker.CheckLivenessDetailed(ctx)
	} else {
		healthy := checker.CheckLiveness(ctx)
		result = &HealthCheckResult{
			Healthy: healthy,
			Details: make(map[string]string),
		}
	}

	handler.HandleLivenessCheck(w, r, result)
}

// readinessHandler handles Kubernetes readiness probe requests
func (h *HealthCheckServer) readinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	checker := h.checker
	handler := h.handler
	h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var result *HealthCheckResult
	if detailedChecker, ok := checker.(DetailedHealthChecker); ok {
		result = detailedChecker.CheckReadinessDetailed(ctx)
	} else {
		healthy := checker.CheckReadiness(ctx)
		result = &HealthCheckResult{
			Healthy: healthy,
			Details: make(map[string]string),
		}
	}

	handler.HandleReadinessCheck(w, r, result)
}

// writeJSONResponse writes JSON response with proper headers
func writeJSONResponse(w http.ResponseWriter, response interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("failed to encode health response: %v", err)
	}
}

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

package triple

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dustin/go-humanize"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"golang.org/x/net/http2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	dubbotls "dubbo.apache.org/dubbo-go/v3/tls"
)

const (
	httpPrefix  string = "http://"
	httpsPrefix string = "https://"
)

// TripleGenericService implements generic invocation for Triple protocol
// It provides a unified interface similar to Dubbo's GenericService
type TripleGenericService struct {
	serviceKey string
}

// NewTripleGenericService creates a new TripleGenericService instance
func NewTripleGenericService(serviceKey string) *TripleGenericService {
	return &TripleGenericService{
		serviceKey: serviceKey,
	}
}

// NewTripleGenericServiceWithClient creates a new TripleGenericService with an existing client
// Note: Currently simplified to just use serviceKey, client parameter is ignored for future compatibility
func NewTripleGenericServiceWithClient(serviceKey string, client any) *TripleGenericService {
	return NewTripleGenericService(serviceKey)
}

// Invoke performs generic invocation with method name, parameter types, and arguments
// This method provides a unified interface similar to Dubbo's GenericService.Invoke
func (tgs *TripleGenericService) Invoke(ctx context.Context, methodName string, types []string, args []any) (any, error) {
	// Validate input parameters
	if methodName == "" {
		return nil, fmt.Errorf("method name cannot be empty")
	}

	// Prepare arguments for generic invocation
	// For Triple protocol generic calls, we follow Dubbo's pattern:
	// args[0] = method name, args[1] = parameter types, args[2] = parameter values
	genericArgs := []any{methodName, types, args}

	// Create RPC invocation with generic method and arguments
	inv := invocation.NewRPCInvocation(constant.Generic, genericArgs, make(map[string]any))

	// Set generic flag to identify this as a generic invocation
	inv.SetAttachment(constant.GenericKey, "true")

	// Set parameter types information if provided
	if len(types) > 0 {
		inv.SetAttachment("parameterTypes", types)
	}

	// Create client manager for the service
	url, urlErr := common.NewURL(tgs.serviceKey, common.WithProtocol("tri"))
	if urlErr != nil {
		return nil, fmt.Errorf("failed to create URL for service %s: %w", tgs.serviceKey, urlErr)
	}

	cm, err := newClientManager(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create client manager for service %s: %w", tgs.serviceKey, err)
	}

	// Perform the actual invocation based on client mode
	if !cm.isIDL {
		// Non-IDL mode: direct parameter passing
		return tgs.invokeNonIDL(ctx, cm, methodName, args)
	} else {
		// IDL mode: protobuf-based invocation
		return tgs.invokeIDL(ctx, cm, inv)
	}
}

// invokeNonIDL handles invocation for non-IDL mode services
func (tgs *TripleGenericService) invokeNonIDL(ctx context.Context, cm *clientManager, methodName string, args []any) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("Triple protocol generic invocation requires at least one argument for non-IDL mode")
	}

	// Get the generalizer manager
	gm := GetTripleGeneralizerManager()

	// Get the appropriate generalizer based on generic type
	// For now, use "true" as default (basic generalizer)
	_ = gm.GetGeneralizer("true")

	// Prepare request parameters - convert generic args to concrete types
	reqParams := make([]any, len(args))
	for i, arg := range args {
		// For generic invocation, args are already in the expected format
		// In a more sophisticated implementation, you might need to determine
		// the target type from method signatures
		reqParams[i] = arg
	}

	// For non-IDL mode, response parameters are typically the same as request parameters
	respParams := make([]any, len(reqParams))
	copy(respParams, reqParams)

	// Execute the call
	err := cm.callUnary(ctx, methodName, reqParams, respParams)
	if err != nil {
		return nil, fmt.Errorf("Triple protocol generic invocation failed for method %s: %w", methodName, err)
	}

	// Return the first response parameter (typical pattern for non-IDL mode)
	if len(respParams) > 0 {
		return respParams[0], nil
	}

	return nil, nil
}

// invokeIDL handles invocation for IDL mode services (protobuf-based)
func (tgs *TripleGenericService) invokeIDL(ctx context.Context, cm *clientManager, inv base.Invocation) (any, error) {
	// For IDL mode, we would typically use protobuf serialization
	// This is a placeholder implementation - in practice, this would involve:
	// 1. Converting generic arguments to protobuf messages
	// 2. Making the actual RPC call
	// 3. Converting protobuf response back to generic format

	// TODO: Implement full IDL mode support
	return nil, fmt.Errorf("IDL mode generic invocation is not yet fully implemented for Triple protocol")
}

// Reference returns the service key
func (tgs *TripleGenericService) Reference() string {
	return tgs.serviceKey
}

// TripleAttachmentManager manages attachments for Triple protocol
type TripleAttachmentManager struct {
	attachments map[string]any
}

func NewTripleAttachmentManager() *TripleAttachmentManager {
	return &TripleAttachmentManager{
		attachments: make(map[string]any),
	}
}

func (tam *TripleAttachmentManager) SetAttachment(key string, value any) {
	tam.attachments[key] = value
}

func (tam *TripleAttachmentManager) GetAttachment(key string) (any, bool) {
	value, exists := tam.attachments[key]
	return value, exists
}

func (tam *TripleAttachmentManager) GetAttachmentString(key string) (string, bool) {
	if value, exists := tam.attachments[key]; exists {
		if str, ok := value.(string); ok {
			return str, true
		}
		// Try to convert to string
		return fmt.Sprintf("%v", value), true
	}
	return "", false
}

func (tam *TripleAttachmentManager) GetAttachmentInt(key string) (int, bool) {
	if value, exists := tam.attachments[key]; exists {
		switch v := value.(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float32:
			return int(v), true
		case float64:
			return int(v), true
		case string:
			if intVal, err := strconv.Atoi(v); err == nil {
				return intVal, true
			}
		}
	}
	return 0, false
}

func (tam *TripleAttachmentManager) GetAttachmentBool(key string) (bool, bool) {
	if value, exists := tam.attachments[key]; exists {
		switch v := value.(type) {
		case bool:
			return v, true
		case string:
			if boolVal, err := strconv.ParseBool(v); err == nil {
				return boolVal, true
			}
		case int, int32, int64:
			// 0 is false, non-zero is true
			return v != 0, true
		}
	}
	return false, false
}

func (tam *TripleAttachmentManager) GetAllAttachments() map[string]any {
	result := make(map[string]any)
	for k, v := range tam.attachments {
		result[k] = v
	}
	return result
}

func (tam *TripleAttachmentManager) RemoveAttachment(key string) {
	delete(tam.attachments, key)
}

func (tam *TripleAttachmentManager) ClearAttachments() {
	tam.attachments = make(map[string]any)
}

func (tam *TripleAttachmentManager) HasAttachment(key string) bool {
	_, exists := tam.attachments[key]
	return exists
}

// SerializeAttachment serializes attachment value to string for HTTP header transmission
func (tam *TripleAttachmentManager) SerializeAttachment(key string) (string, error) {
	if value, exists := tam.attachments[key]; exists {
		return tam.serializeValue(value)
	}
	return "", fmt.Errorf("attachment %s not found", key)
}

// DeserializeAttachment deserializes string back to attachment value
func (tam *TripleAttachmentManager) DeserializeAttachment(key, value string) error {
	deserializedValue, err := tam.deserializeValue(value)
	if err != nil {
		return err
	}
	tam.attachments[key] = deserializedValue
	return nil
}

func (tam *TripleAttachmentManager) serializeValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%g", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case []string:
		// For string arrays, join with comma
		return strings.Join(v, ","), nil
	default:
		// For complex objects, use JSON serialization
		if data, err := json.Marshal(value); err != nil {
			return "", fmt.Errorf("failed to serialize attachment: %w", err)
		} else {
			return string(data), nil
		}
	}
}

func (tam *TripleAttachmentManager) deserializeValue(value string) (any, error) {
	// Try to parse as basic types first
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal, nil
	}
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, nil
	}
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal, nil
	}

	// Try to parse as JSON for complex objects
	var jsonVal any
	if err := json.Unmarshal([]byte(value), &jsonVal); err == nil {
		return jsonVal, nil
	}

	// If all else fails, return as string
	return value, nil
}

// Enhanced TripleGenericService with attachment support
func (tgs *TripleGenericService) InvokeWithAttachments(
	ctx context.Context,
	methodName string,
	types []string,
	args []any,
	attachments map[string]any,
) (any, error) {
	// Validate input parameters
	if methodName == "" {
		return nil, fmt.Errorf("method name cannot be empty")
	}

	// Prepare arguments for generic invocation
	genericArgs := []any{methodName, types, args}

	// Create RPC invocation with generic method and arguments
	inv := invocation.NewRPCInvocation(constant.Generic, genericArgs, make(map[string]any))

	// Set generic flag to identify this as a generic invocation
	inv.SetAttachment(constant.GenericKey, "true")

	// Set parameter types information if provided
	if len(types) > 0 {
		inv.SetAttachment("parameterTypes", types)
	}

	// Set custom attachments
	if attachments != nil {
		for key, value := range attachments {
			inv.SetAttachment(key, value)
		}
	}

	// Create client manager for the service
	url, urlErr := common.NewURL(tgs.serviceKey, common.WithProtocol("tri"))
	if urlErr != nil {
		return nil, fmt.Errorf("failed to create URL for service %s: %w", tgs.serviceKey, urlErr)
	}

	cm, err := newClientManager(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create client manager for service %s: %w", tgs.serviceKey, err)
	}

	// Perform the actual invocation based on client mode
	if !cm.isIDL {
		return tgs.invokeNonIDL(ctx, cm, methodName, args)
	} else {
		return tgs.invokeIDL(ctx, cm, inv)
	}
}

// TripleAsyncManager manages asynchronous invocations
type TripleAsyncManager struct {
	activeCalls map[string]*TripleAsyncCall
	mutex       sync.RWMutex
}

var (
	tripleAsyncManager     *TripleAsyncManager
	tripleAsyncManagerOnce sync.Once
)

func GetTripleAsyncManager() *TripleAsyncManager {
	tripleAsyncManagerOnce.Do(func() {
		tripleAsyncManager = &TripleAsyncManager{
			activeCalls: make(map[string]*TripleAsyncCall),
		}
	})
	return tripleAsyncManager
}

type TripleAsyncCall struct {
	ID          string
	MethodName  string
	StartTime   time.Time
	Timeout     time.Duration
	CancelFunc  context.CancelFunc
	Callback    func(result any, err error)
	Attachments map[string]any
}

// TripleAsyncResult represents the result of an asynchronous call
type TripleAsyncResult struct {
	CallID      string
	MethodName  string
	Result      any
	Error       error
	Duration    time.Duration
	StartTime   time.Time
	EndTime     time.Time
	Attachments map[string]any
}

// InvokeAsync performs asynchronous generic invocation with attachments
func (tgs *TripleGenericService) InvokeAsync(
	ctx context.Context,
	methodName string,
	types []string,
	args []any,
	attachments map[string]any,
	callback func(result any, err error),
) (string, error) {
	return tgs.InvokeAsyncWithTimeout(ctx, methodName, types, args, attachments, callback, 30*time.Second)
}

// InvokeAsyncWithTimeout performs asynchronous generic invocation with custom timeout
func (tgs *TripleGenericService) InvokeAsyncWithTimeout(
	ctx context.Context,
	methodName string,
	types []string,
	args []any,
	attachments map[string]any,
	callback func(result any, err error),
	timeout time.Duration,
) (string, error) {
	// Validate input parameters
	if methodName == "" {
		return "", fmt.Errorf("method name cannot be empty")
	}
	if callback == nil {
		return "", fmt.Errorf("callback function cannot be nil")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second // default timeout
	}

	// Generate unique call ID
	callID := generateCallID()

	// Create async call context with timeout
	asyncCtx, cancelFunc := context.WithTimeout(ctx, timeout)

	// Create async call object
	asyncCall := &TripleAsyncCall{
		ID:          callID,
		MethodName:  methodName,
		StartTime:   time.Now(),
		Timeout:     timeout,
		CancelFunc:  cancelFunc,
		Callback:    callback,
		Attachments: make(map[string]any),
	}

	// Copy attachments
	if attachments != nil {
		for k, v := range attachments {
			asyncCall.Attachments[k] = v
		}
	}

	// Register async call
	asyncManager := GetTripleAsyncManager()
	asyncManager.registerCall(asyncCall)

	// Start async execution
	go func() {
		defer asyncManager.unregisterCall(callID)
		defer cancelFunc()

		// Execute the call synchronously
		result, err := tgs.InvokeWithAttachments(asyncCtx, methodName, types, args, attachments)

		// Check if context was cancelled or timed out
		if asyncCtx.Err() != nil {
			if asyncCtx.Err() == context.DeadlineExceeded {
				err = fmt.Errorf("async call %s timed out after %v", callID, timeout)
			} else {
				err = fmt.Errorf("async call %s was cancelled: %w", callID, asyncCtx.Err())
			}
		}

		// Create result object
		asyncResult := &TripleAsyncResult{
			CallID:      callID,
			MethodName:  methodName,
			Result:      result,
			Error:       err,
			StartTime:   asyncCall.StartTime,
			EndTime:     time.Now(),
			Attachments: asyncCall.Attachments,
		}
		asyncResult.Duration = asyncResult.EndTime.Sub(asyncResult.StartTime)

		// Call the callback function
		asyncCall.Callback(asyncResult.Result, asyncResult.Error)
	}()

	return callID, nil
}

// InvokeAsyncBatch performs batch asynchronous invocations
func (tgs *TripleGenericService) InvokeAsyncBatch(
	ctx context.Context,
	invocations []TripleInvocationRequest,
	callback func(results []TripleAsyncResult),
) ([]string, error) {
	return tgs.InvokeAsyncBatchWithTimeout(ctx, invocations, callback, 30*time.Second)
}

// InvokeAsyncBatchWithTimeout performs batch asynchronous invocations with custom timeout
func (tgs *TripleGenericService) InvokeAsyncBatchWithTimeout(
	ctx context.Context,
	invocations []TripleInvocationRequest,
	callback func(results []TripleAsyncResult),
	timeout time.Duration,
) ([]string, error) {
	if len(invocations) == 0 {
		return nil, fmt.Errorf("no invocations provided")
	}
	if callback == nil {
		return nil, fmt.Errorf("callback function cannot be nil")
	}

	callIDs := make([]string, len(invocations))
	results := make([]TripleAsyncResult, len(invocations))
	resultsReceived := 0
	var resultsMutex sync.Mutex

	// Start all async calls
	for i, req := range invocations {
		localIndex := i
		localReq := req

		// Generate callID first
		currentCallID := generateCallID()
		callIDs[localIndex] = currentCallID

		_, err := tgs.InvokeAsyncWithTimeout(ctx, localReq.MethodName, localReq.Types, localReq.Args, localReq.Attachments,
			func(result any, resultErr error) {
				// Collect result
				resultsMutex.Lock()
				results[localIndex] = TripleAsyncResult{
					CallID:      currentCallID,
					MethodName:  localReq.MethodName,
					Result:      result,
					Error:       resultErr,
					StartTime:   time.Now(),
					EndTime:     time.Now(),
					Attachments: localReq.Attachments,
				}
				results[localIndex].Duration = results[localIndex].EndTime.Sub(results[localIndex].StartTime)
				resultsReceived++
				resultsMutex.Unlock()

				// Call batch callback when all results are received
				resultsMutex.Lock()
				if resultsReceived == len(invocations) {
					callback(results)
				}
				resultsMutex.Unlock()
			}, timeout)

		if err != nil {
			return nil, fmt.Errorf("failed to start async call %d: %w", i, err)
		}
	}

	return callIDs, nil
}

// CancelAsyncCall cancels an asynchronous call
func (tgs *TripleGenericService) CancelAsyncCall(callID string) bool {
	asyncManager := GetTripleAsyncManager()
	asyncCall := asyncManager.getCall(callID)
	if asyncCall != nil {
		asyncCall.CancelFunc()
		asyncManager.unregisterCall(callID)
		return true
	}
	return false
}

// GetAsyncCallStatus gets the status of an asynchronous call
func (tgs *TripleGenericService) GetAsyncCallStatus(callID string) (*TripleAsyncCall, bool) {
	asyncManager := GetTripleAsyncManager()
	asyncCall := asyncManager.getCall(callID)
	if asyncCall != nil {
		return asyncCall, true
	}
	return nil, false
}

// GetActiveAsyncCalls gets all active asynchronous calls
func (tgs *TripleGenericService) GetActiveAsyncCalls() map[string]*TripleAsyncCall {
	asyncManager := GetTripleAsyncManager()
	return asyncManager.getAllActiveCalls()
}

// WaitForAsyncCall waits for an asynchronous call to complete
func (tgs *TripleGenericService) WaitForAsyncCall(callID string, timeout time.Duration) (*TripleAsyncResult, error) {
	asyncManager := GetTripleAsyncManager()

	// Check if call exists
	asyncCall := asyncManager.getCall(callID)
	if asyncCall == nil {
		return nil, fmt.Errorf("async call %s not found", callID)
	}

	// Create channel to wait for completion
	resultChan := make(chan *TripleAsyncResult, 1)

	// Wrap the original callback
	originalCallback := asyncCall.Callback
	asyncCall.Callback = func(result any, err error) {
		resultChan <- &TripleAsyncResult{
			CallID:      callID,
			MethodName:  asyncCall.MethodName,
			Result:      result,
			Error:       err,
			StartTime:   asyncCall.StartTime,
			EndTime:     time.Now(),
			Attachments: asyncCall.Attachments,
		}
		// Call original callback if it exists
		if originalCallback != nil {
			originalCallback(result, err)
		}
	}

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, nil
	case <-time.After(timeout):
		// Cancel the call if timeout
		tgs.CancelAsyncCall(callID)
		return nil, fmt.Errorf("wait for async call %s timed out after %v", callID, timeout)
	}
}

// registerCall registers an async call
func (tam *TripleAsyncManager) registerCall(asyncCall *TripleAsyncCall) {
	tam.mutex.Lock()
	defer tam.mutex.Unlock()
	tam.activeCalls[asyncCall.ID] = asyncCall
}

// unregisterCall unregisters an async call
func (tam *TripleAsyncManager) unregisterCall(callID string) {
	tam.mutex.Lock()
	defer tam.mutex.Unlock()
	delete(tam.activeCalls, callID)
}

// getCall gets an async call by ID
func (tam *TripleAsyncManager) getCall(callID string) *TripleAsyncCall {
	tam.mutex.RLock()
	defer tam.mutex.RUnlock()
	return tam.activeCalls[callID]
}

// GetAllActiveCalls gets all active calls (exported method)
func (tam *TripleAsyncManager) GetAllActiveCalls() map[string]*TripleAsyncCall {
	tam.mutex.RLock()
	defer tam.mutex.RUnlock()

	result := make(map[string]*TripleAsyncCall)
	for k, v := range tam.activeCalls {
		result[k] = v
	}
	return result
}

// getAllActiveCalls gets all active calls (internal method)
func (tam *TripleAsyncManager) getAllActiveCalls() map[string]*TripleAsyncCall {
	return tam.GetAllActiveCalls()
}

// GetActiveCallCount gets the count of active calls
func (tam *TripleAsyncManager) GetActiveCallCount() int {
	tam.mutex.RLock()
	defer tam.mutex.RUnlock()
	return len(tam.activeCalls)
}

// CleanupExpiredCalls cleans up expired async calls
func (tam *TripleAsyncManager) CleanupExpiredCalls() int {
	tam.mutex.Lock()
	defer tam.mutex.Unlock()

	now := time.Now()
	cleanupCount := 0

	for id, asyncCall := range tam.activeCalls {
		if now.After(asyncCall.StartTime.Add(asyncCall.Timeout)) {
			asyncCall.CancelFunc()
			delete(tam.activeCalls, id)
			cleanupCount++
		}
	}

	return cleanupCount
}

// generateCallID generates a unique call ID
func generateCallID() string {
	return fmt.Sprintf("triple-async-%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// BatchInvoke performs batch generic invocations with attachments
func (tgs *TripleGenericService) BatchInvoke(
	ctx context.Context,
	invocations []TripleInvocationRequest,
) ([]TripleInvocationResult, error) {
	if len(invocations) == 0 {
		return nil, fmt.Errorf("no invocations provided")
	}

	results := make([]TripleInvocationResult, len(invocations))

	for i, req := range invocations {
		result, err := tgs.InvokeWithAttachments(ctx, req.MethodName, req.Types, req.Args, req.Attachments)
		results[i] = TripleInvocationResult{
			Result: result,
			Error:  err,
			Index:  i,
		}
	}

	return results, nil
}

// BatchInvokeWithOptions performs batch generic invocations with options
func (tgs *TripleGenericService) BatchInvokeWithOptions(
	ctx context.Context,
	invocations []TripleInvocationRequest,
	options BatchInvokeOptions,
) ([]TripleInvocationResult, error) {
	if len(invocations) == 0 {
		return nil, fmt.Errorf("no invocations provided")
	}

	// Set default max concurrency if not specified
	maxConcurrency := options.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // default concurrency
	}

	results := make([]TripleInvocationResult, len(invocations))
	resultsMutex := sync.Mutex{}

	// Create semaphore for concurrency control
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	// Flag to track if we should fail fast
	var shouldStop bool
	var stopMutex sync.Mutex

	for i, req := range invocations {
		// Check if we should stop due to FailFast
		if options.FailFast {
			stopMutex.Lock()
			if shouldStop {
				stopMutex.Unlock()
				break
			}
			stopMutex.Unlock()
		}

		wg.Add(1)
		go func(index int, request TripleInvocationRequest) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check again if we should stop
			if options.FailFast {
				stopMutex.Lock()
				if shouldStop {
					stopMutex.Unlock()
					return
				}
				stopMutex.Unlock()
			}

			// Perform invocation
			result, err := tgs.InvokeWithAttachments(ctx, request.MethodName, request.Types, request.Args, request.Attachments)

			// Store result
			resultsMutex.Lock()
			results[index] = TripleInvocationResult{
				Result: result,
				Error:  err,
				Index:  index,
			}
			resultsMutex.Unlock()

			// If FailFast is enabled and there's an error, set stop flag
			if options.FailFast && err != nil {
				stopMutex.Lock()
				shouldStop = true
				stopMutex.Unlock()
			}
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

// TripleInvocationRequest represents a single invocation request
type TripleInvocationRequest struct {
	MethodName  string
	Types       []string
	Args        []any
	Attachments map[string]any
}

// TripleInvocationResult represents the result of a single invocation
type TripleInvocationResult struct {
	Result any
	Error  error
	Index  int
}

// BatchInvokeOptions Batch invocation options
type BatchInvokeOptions struct {
	MaxConcurrency int
	FailFast       bool
}

// CreateAttachmentBuilder creates a fluent builder for attachments
func (tgs *TripleGenericService) CreateAttachmentBuilder() *TripleAttachmentBuilder {
	return &TripleAttachmentBuilder{
		attachments: make(map[string]any),
	}
}

// TripleAttachmentBuilder provides fluent API for building attachments
type TripleAttachmentBuilder struct {
	attachments map[string]any
}

func (tab *TripleAttachmentBuilder) SetString(key, value string) *TripleAttachmentBuilder {
	tab.attachments[key] = value
	return tab
}

func (tab *TripleAttachmentBuilder) SetInt(key string, value int) *TripleAttachmentBuilder {
	tab.attachments[key] = value
	return tab
}

func (tab *TripleAttachmentBuilder) SetBool(key string, value bool) *TripleAttachmentBuilder {
	tab.attachments[key] = value
	return tab
}

func (tab *TripleAttachmentBuilder) SetObject(key string, value any) *TripleAttachmentBuilder {
	tab.attachments[key] = value
	return tab
}

func (tab *TripleAttachmentBuilder) SetMap(key string, value map[string]any) *TripleAttachmentBuilder {
	tab.attachments[key] = value
	return tab
}

func (tab *TripleAttachmentBuilder) Build() map[string]any {
	result := make(map[string]any)
	for k, v := range tab.attachments {
		result[k] = v
	}
	return result
}

// convertParamsForIDL converts parameters for IDL mode
func (tgs *TripleGenericService) convertParamsForIDL(params any, types []string) ([]any, error) {
	// Handle different parameter formats
	var paramList []any

	// If params is already a slice
	if reflect.TypeOf(params).Kind() == reflect.Slice {
		paramList = params.([]any)
	} else {
		// Single parameter
		paramList = []any{params}
	}

	// Check if types and params match
	if len(paramList) != len(types) {
		return nil, fmt.Errorf("parameter count mismatch: got %d params but %d types", len(paramList), len(types))
	}

	// Convert each parameter
	result := make([]any, len(paramList))
	for i, param := range paramList {
		converted, err := convertSingleArgForSerialization(param, types[i])
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter %d: %w", i, err)
		}
		result[i] = converted
	}

	return result, nil
}

// convertSingleArgForSerialization converts a single argument based on target type
func convertSingleArgForSerialization(arg any, targetType string) (any, error) {
	if arg == nil {
		return nil, nil
	}

	// Handle basic type conversions
	switch targetType {
	case "string", "java.lang.String", "String":
		return fmt.Sprintf("%v", arg), nil

	case "int", "int32", "java.lang.Integer", "Integer":
		switch v := arg.(type) {
		case int:
			return int32(v), nil
		case int32:
			return v, nil
		case int64:
			return int32(v), nil
		case float64:
			return int32(v), nil
		case string:
			val, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("cannot convert '%s' to int32: %w", v, err)
			}
			return int32(val), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to int32", arg)
		}

	case "int64", "long", "java.lang.Long", "Long":
		switch v := arg.(type) {
		case int:
			return int64(v), nil
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		case string:
			val, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert '%s' to int64: %w", v, err)
			}
			return val, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to int64", arg)
		}

	case "float32", "float", "java.lang.Float", "Float":
		switch v := arg.(type) {
		case float32:
			return v, nil
		case float64:
			return float32(v), nil
		case int:
			return float32(v), nil
		case int32:
			return float32(v), nil
		case int64:
			return float32(v), nil
		case string:
			val, err := strconv.ParseFloat(v, 32)
			if err != nil {
				return nil, fmt.Errorf("cannot convert '%s' to float32: %w", v, err)
			}
			return float32(val), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to float32", arg)
		}

	case "float64", "double", "java.lang.Double", "Double":
		val, err := convertToFloat64(arg)
		if err != nil {
			return nil, err
		}
		return val, nil

	case "bool", "boolean", "java.lang.Boolean", "Boolean":
		switch v := arg.(type) {
		case bool:
			return v, nil
		case string:
			val, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("cannot convert '%s' to bool: %w", v, err)
			}
			return val, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to bool", arg)
		}

	case "map", "Map", "java.util.Map", "object", "Object":
		// For map/object types, return as-is or convert to map
		if reflect.TypeOf(arg).Kind() == reflect.Map {
			return arg, nil
		}
		// Try to convert struct to map
		return objToTripleMap(arg), nil

	default:
		// For unknown types, return as-is
		return arg, nil
	}
}

// convertToFloat64 converts various types to float64
func convertToFloat64(arg any) (float64, error) {
	if arg == nil {
		return 0, nil
	}

	switch v := arg.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to float64: %w", v, err)
		}
		return val, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", arg)
	}
}

// convertToInt32 converts various types to int32
func convertToInt32(arg any) (int32, error) {
	if arg == nil {
		return 0, nil
	}

	switch v := arg.(type) {
	case int32:
		return v, nil
	case int:
		return int32(v), nil
	case int64:
		return int32(v), nil
	case float64:
		return int32(v), nil
	case float32:
		return int32(v), nil
	case string:
		val, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to int32: %w", v, err)
		}
		return int32(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int32", arg)
	}
}

// Convenience methods for common attachment patterns
func (tgs *TripleGenericService) WithTimeout(timeoutMs int) *TripleAttachmentBuilder {
	return tgs.CreateAttachmentBuilder().SetInt(constant.TimeoutKey, timeoutMs)
}

func (tgs *TripleGenericService) WithUserId(userId string) *TripleAttachmentBuilder {
	return tgs.CreateAttachmentBuilder().SetString("userId", userId)
}

func (tgs *TripleGenericService) WithTraceId(traceId string) *TripleAttachmentBuilder {
	return tgs.CreateAttachmentBuilder().SetString("traceId", traceId)
}

func (tgs *TripleGenericService) WithCustomAttachments(attachments map[string]any) *TripleAttachmentBuilder {
	builder := tgs.CreateAttachmentBuilder()
	for k, v := range attachments {
		builder.attachments[k] = v
	}
	return builder
}

// TripleGeneralizer defines the interface for data type conversion in Triple protocol
type TripleGeneralizer interface {
	// Generalize converts an object to a general format (like Map)
	Generalize(obj any) (any, error)

	// Realize converts a general format back to a specific type
	Realize(obj any, typ reflect.Type) (any, error)

	// GetType returns the type name of the object
	GetType(obj any) (string, error)
}

// MapTripleGeneralizer implements TripleGeneralizer for Map-based conversion
type MapTripleGeneralizer struct{}

func (g *MapTripleGeneralizer) Generalize(obj any) (any, error) {
	return objToTripleMap(obj), nil
}

func (g *MapTripleGeneralizer) Realize(obj any, typ reflect.Type) (any, error) {
	// Simple map to struct conversion
	// In practice, you might want to use a more robust solution like mapstructure
	if data, ok := obj.(map[string]any); ok {
		// Basic map to struct conversion
		result := reflect.New(typ).Elem()
		for key, value := range data {
			field := result.FieldByName(key)
			if field.IsValid() && field.CanSet() {
				if val := reflect.ValueOf(value); val.Type().AssignableTo(field.Type()) {
					field.Set(val)
				}
			}
		}
		return result.Interface(), nil
	}
	return obj, nil
}

func (g *MapTripleGeneralizer) GetType(obj any) (string, error) {
	if obj == nil {
		return "java.lang.Object", nil
	}
	return reflect.TypeOf(obj).String(), nil
}

// JSONTripleGeneralizer implements TripleGeneralizer for JSON-based conversion
type JSONTripleGeneralizer struct{}

func (g *JSONTripleGeneralizer) Generalize(obj any) (any, error) {
	// Convert to JSON string for simplicity
	// In practice, you might want to return structured data
	if data, err := json.Marshal(obj); err != nil {
		return nil, err
	} else {
		return string(data), nil
	}
}

func (g *JSONTripleGeneralizer) Realize(obj any, typ reflect.Type) (any, error) {
	if str, ok := obj.(string); ok {
		newobj := reflect.New(typ).Interface()
		if err := json.Unmarshal([]byte(str), newobj); err != nil {
			return nil, err
		}
		return reflect.ValueOf(newobj).Elem().Interface(), nil
	}
	return obj, nil
}

func (g *JSONTripleGeneralizer) GetType(obj any) (string, error) {
	if obj == nil {
		return "java.lang.Object", nil
	}
	return "java.lang.String", nil // JSON format is represented as string
}

// ProtobufTripleGeneralizer implements TripleGeneralizer for Protobuf conversion
type ProtobufTripleGeneralizer struct{}

func (g *ProtobufTripleGeneralizer) Generalize(obj any) (any, error) {
	// For Triple protocol, protobuf is the native format
	// Return as-is for protobuf messages
	return obj, nil
}

func (g *ProtobufTripleGeneralizer) Realize(obj any, typ reflect.Type) (any, error) {
	// For protobuf, the object should already be in the correct format
	if reflect.TypeOf(obj).AssignableTo(typ) {
		return obj, nil
	}
	return nil, fmt.Errorf("cannot convert %T to %s", obj, typ.String())
}

func (g *ProtobufTripleGeneralizer) GetType(obj any) (string, error) {
	if obj == nil {
		return "java.lang.Object", nil
	}
	return reflect.TypeOf(obj).String(), nil
}

// BasicTripleGeneralizer implements TripleGeneralizer for basic types
type BasicTripleGeneralizer struct{}

func (g *BasicTripleGeneralizer) Generalize(obj any) (any, error) {
	// Basic types (int, string, bool, etc.) are returned as-is
	return obj, nil
}

func (g *BasicTripleGeneralizer) Realize(obj any, typ reflect.Type) (any, error) {
	val := reflect.ValueOf(obj)
	if val.Type().AssignableTo(typ) {
		return obj, nil
	}

	// Try type conversion for basic types
	targetVal := reflect.New(typ).Elem()
	if val.CanConvert(typ) {
		targetVal.Set(val.Convert(typ))
		return targetVal.Interface(), nil
	}

	return nil, fmt.Errorf("cannot convert %T to %s", obj, typ.String())
}

func (g *BasicTripleGeneralizer) GetType(obj any) (string, error) {
	if obj == nil {
		return "java.lang.Object", nil
	}

	switch reflect.TypeOf(obj).Kind() {
	case reflect.String:
		return "java.lang.String", nil
	case reflect.Int, reflect.Int32:
		return "java.lang.Integer", nil
	case reflect.Int64:
		return "java.lang.Long", nil
	case reflect.Float32:
		return "java.lang.Float", nil
	case reflect.Float64:
		return "java.lang.Double", nil
	case reflect.Bool:
		return "java.lang.Boolean", nil
	default:
		return reflect.TypeOf(obj).String(), nil
	}
}

// TripleGeneralizerManager manages different generalizers
type TripleGeneralizerManager struct {
	generalizers map[string]TripleGeneralizer
}

var (
	tripleGeneralizerManager     *TripleGeneralizerManager
	tripleGeneralizerManagerOnce sync.Once
)

func GetTripleGeneralizerManager() *TripleGeneralizerManager {
	tripleGeneralizerManagerOnce.Do(func() {
		tripleGeneralizerManager = &TripleGeneralizerManager{
			generalizers: make(map[string]TripleGeneralizer),
		}
		tripleGeneralizerManager.initGeneralizers()
	})
	return tripleGeneralizerManager
}

func (tgm *TripleGeneralizerManager) initGeneralizers() {
	tgm.generalizers["map"] = &MapTripleGeneralizer{}
	tgm.generalizers["json"] = &JSONTripleGeneralizer{}
	tgm.generalizers["protobuf"] = &ProtobufTripleGeneralizer{}
	tgm.generalizers["basic"] = &BasicTripleGeneralizer{}
	// Default generalizer
	tgm.generalizers["true"] = &BasicTripleGeneralizer{}
}

func (tgm *TripleGeneralizerManager) GetGeneralizer(genericType string) TripleGeneralizer {
	if g, ok := tgm.generalizers[genericType]; ok {
		return g
	}
	// Return default generalizer
	return tgm.generalizers["true"]
}

// objToTripleMap converts an object to a map for Triple protocol
func objToTripleMap(obj any) any {
	if obj == nil {
		return obj
	}

	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)

	// Dereference pointers
	for typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		result := make(map[string]any)
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			fieldVal := val.Field(i)

			// Skip unexported fields
			if !fieldVal.CanInterface() {
				continue
			}

			// Use field name as key, recursively convert field value
			fieldName := field.Name
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				fieldName = jsonTag
			}

			result[fieldName] = objToTripleMap(fieldVal.Interface())
		}
		return result

	case reflect.Slice, reflect.Array:
		var result []any
		for i := 0; i < val.Len(); i++ {
			result = append(result, objToTripleMap(val.Index(i).Interface()))
		}
		return result

	case reflect.Map:
		result := make(map[string]any)
		for _, key := range val.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			result[keyStr] = objToTripleMap(val.MapIndex(key).Interface())
		}
		return result

	default:
		return obj
	}
}

// clientManager wraps triple clients and is responsible for find concrete triple client to invoke
// callUnary, callClientStream, callServerStream, callBidiStream.
// A Reference has a clientManager.
type clientManager struct {
	isIDL bool
	// triple_protocol clients, key is method name
	triClients map[string]*tri.Client
}

// TODO: code a triple client between clientManager and triple_protocol client
// TODO: write a NewClient for triple client

func (cm *clientManager) getClient(method string) (*tri.Client, error) {
	triClient, ok := cm.triClients[method]
	if !ok {
		return nil, fmt.Errorf("missing triple client for method: %s", method)
	}
	return triClient, nil
}

func (cm *clientManager) callUnary(ctx context.Context, method string, req, resp any) error {
	triClient, err := cm.getClient(method)
	if err != nil {
		return err
	}
	triReq := tri.NewRequest(req)
	triResp := tri.NewResponse(resp)
	if err := triClient.CallUnary(ctx, triReq, triResp); err != nil {
		return err
	}
	return nil
}

func (cm *clientManager) callClientStream(ctx context.Context, method string) (any, error) {
	triClient, err := cm.getClient(method)
	if err != nil {
		return nil, err
	}
	stream, err := triClient.CallClientStream(ctx)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (cm *clientManager) callServerStream(ctx context.Context, method string, req any) (any, error) {
	triClient, err := cm.getClient(method)
	if err != nil {
		return nil, err
	}
	triReq := tri.NewRequest(req)
	stream, err := triClient.CallServerStream(ctx, triReq)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (cm *clientManager) callBidiStream(ctx context.Context, method string) (any, error) {
	triClient, err := cm.getClient(method)
	if err != nil {
		return nil, err
	}
	stream, err := triClient.CallBidiStream(ctx)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (cm *clientManager) close() error {
	// There is no need to release resources right now.
	// But we leave this function here for future use.
	return nil
}

// newClientManager extracts configurations from url and builds clientManager
func newClientManager(url *common.URL) (*clientManager, error) {
	var cliOpts []tri.ClientOption
	var isIDL bool

	// set serialization
	serialization := url.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	switch serialization {
	case constant.ProtobufSerialization:
		isIDL = true
	case constant.JSONSerialization:
		isIDL = true
		cliOpts = append(cliOpts, tri.WithProtoJSON())
	case constant.Hessian2Serialization:
		cliOpts = append(cliOpts, tri.WithHessian2())
	case constant.MsgpackSerialization:
		cliOpts = append(cliOpts, tri.WithMsgPack())
	default:
		panic(fmt.Sprintf("Unsupported serialization: %s", serialization))
	}

	// set timeout
	timeout := url.GetParamDuration(constant.TimeoutKey, "")
	cliOpts = append(cliOpts, tri.WithTimeout(timeout))

	// set service group and version
	group := url.GetParam(constant.GroupKey, "")
	version := url.GetParam(constant.VersionKey, "")
	cliOpts = append(cliOpts, tri.WithGroup(group), tri.WithVersion(version))

	// todo(DMwangnima): support opentracing

	// handle tls
	var (
		tlsFlag bool
		tlsConf *global.TLSConfig
		cfg     *tls.Config
		err     error
	)

	tlsConfRaw, ok := url.GetAttribute(constant.TLSConfigKey)
	if ok {
		tlsConf, ok = tlsConfRaw.(*global.TLSConfig)
		if !ok {
			return nil, errors.New("TRIPLE clientManager initialized the TLSConfig configuration failed")
		}
	}
	if dubbotls.IsClientTLSValid(tlsConf) {
		cfg, err = dubbotls.GetClientTlSConfig(tlsConf)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			logger.Infof("TRIPLE clientManager initialized the TLSConfig configuration")
			tlsFlag = true
		}
	}

	var tripleConf *global.TripleConfig

	tripleConfRaw, ok := url.GetAttribute(constant.TripleConfigKey)
	if ok {
		tripleConf = tripleConfRaw.(*global.TripleConfig)
	}

	// handle keepalive options
	cliKeepAliveOpts, keepAliveInterval, keepAliveTimeout, genKeepAliveOptsErr := genKeepAliveOptions(url, tripleConf)
	if genKeepAliveOptsErr != nil {
		logger.Errorf("genKeepAliveOpts err: %v", genKeepAliveOptsErr)
		return nil, genKeepAliveOptsErr
	}
	cliOpts = append(cliOpts, cliKeepAliveOpts...)

	// handle http transport of triple protocol
	var transport http.RoundTripper

	var callProtocol string
	if tripleConf != nil && tripleConf.Http3 != nil && tripleConf.Http3.Enable {
		callProtocol = constant.CallHTTP2AndHTTP3
	} else {
		// HTTP default type is HTTP/2.
		callProtocol = constant.CallHTTP2
	}

	switch callProtocol {
	// This case might be for backward compatibility,
	// it's not useful for the Triple protocol, HTTP/1 lacks trailer functionality.
	// Triple protocol only supports HTTP/2 and HTTP/3.
	case constant.CallHTTP:
		transport = &http.Transport{
			TLSClientConfig: cfg,
		}
		cliOpts = append(cliOpts, tri.WithTriple())
	case constant.CallHTTP2:
		// TODO: Enrich the http2 transport config for triple protocol.
		if tlsFlag {
			transport = &http2.Transport{
				TLSClientConfig: cfg,
				ReadIdleTimeout: keepAliveInterval,
				PingTimeout:     keepAliveTimeout,
			}
		} else {
			transport = &http2.Transport{
				DialTLSContext: func(_ context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
				AllowHTTP:       true,
				ReadIdleTimeout: keepAliveInterval,
				PingTimeout:     keepAliveTimeout,
			}
		}
	case constant.CallHTTP3:
		if !tlsFlag {
			return nil, fmt.Errorf("TRIPLE http3 client must have TLS config, but TLS config is nil")
		}

		// TODO: Enrich the http3 transport config for triple protocol.
		transport = &http3.Transport{
			TLSClientConfig: cfg,
			QUICConfig: &quic.Config{
				// ref: https://quic-go.net/docs/quic/connection/#keeping-a-connection-alive
				KeepAlivePeriod: keepAliveInterval,
				// ref: https://quic-go.net/docs/quic/connection/#idle-timeout
				MaxIdleTimeout: keepAliveTimeout,
			},
		}

		logger.Infof("Triple http3 client transport init successfully")
	case constant.CallHTTP2AndHTTP3:
		if !tlsFlag {
			return nil, fmt.Errorf("TRIPLE HTTP/2 and HTTP/3 client must have TLS config, but TLS config is nil")
		}

		// Create a dual transport that can handle both HTTP/2 and HTTP/3
		transport = newDualTransport(cfg, keepAliveInterval, keepAliveTimeout)
		logger.Infof("Triple HTTP/2 and HTTP/3 client transport init successfully")
	default:
		return nil, fmt.Errorf("unsupported http protocol: %s", callProtocol)
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	var baseTriURL string
	baseTriURL = strings.TrimPrefix(url.Location, httpPrefix)
	baseTriURL = strings.TrimPrefix(baseTriURL, httpsPrefix)
	if tlsFlag {
		baseTriURL = httpsPrefix + baseTriURL
	} else {
		baseTriURL = httpPrefix + baseTriURL
	}

	triClients := make(map[string]*tri.Client)

	if len(url.Methods) != 0 {
		for _, method := range url.Methods {
			triURL, err := joinPath(baseTriURL, url.Interface(), method)
			if err != nil {
				return nil, fmt.Errorf("JoinPath failed for base %s, interface %s, method %s", baseTriURL, url.Interface(), method)
			}
			triClient := tri.NewClient(httpClient, triURL, cliOpts...)
			triClients[method] = triClient
		}
	} else {
		// This branch is for the non-IDL mode, where we pass in the service solely
		// for the purpose of using reflection to obtain all methods of the service.
		// There might be potential for optimization in this area later on.
		service, ok := url.GetAttribute(constant.RpcServiceKey)
		if !ok {
			return nil, fmt.Errorf("triple clientmanager can't get methods")
		}

		serviceType := reflect.TypeOf(service)
		for i := range serviceType.NumMethod() {
			methodName := serviceType.Method(i).Name
			triURL, err := joinPath(baseTriURL, url.Interface(), methodName)
			if err != nil {
				return nil, fmt.Errorf("JoinPath failed for base %s, interface %s, method %s", baseTriURL, url.Interface(), methodName)
			}
			triClient := tri.NewClient(httpClient, triURL, cliOpts...)
			triClients[methodName] = triClient
		}
	}

	return &clientManager{
		isIDL:      isIDL,
		triClients: triClients,
	}, nil
}

func genKeepAliveOptions(url *common.URL, tripleConf *global.TripleConfig) ([]tri.ClientOption, time.Duration, time.Duration, error) {
	var cliKeepAliveOpts []tri.ClientOption

	// set max send and recv msg size
	maxCallRecvMsgSize := constant.DefaultMaxCallRecvMsgSize
	if recvMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallRecvMsgSize, "")); err == nil && recvMsgSize > 0 {
		maxCallRecvMsgSize = int(recvMsgSize)
	}
	cliKeepAliveOpts = append(cliKeepAliveOpts, tri.WithReadMaxBytes(maxCallRecvMsgSize))
	maxCallSendMsgSize := constant.DefaultMaxCallSendMsgSize
	if sendMsgSize, err := humanize.ParseBytes(url.GetParam(constant.MaxCallSendMsgSize, "")); err == nil && sendMsgSize > 0 {
		maxCallSendMsgSize = int(sendMsgSize)
	}
	cliKeepAliveOpts = append(cliKeepAliveOpts, tri.WithSendMaxBytes(maxCallSendMsgSize))

	// set keepalive interval and keepalive timeout
	// Deprecated：use tripleconfig
	// TODO: remove KeepAliveInterval and KeepAliveInterval in version 4.0.0
	keepAliveInterval := url.GetParamDuration(constant.KeepAliveInterval, constant.DefaultKeepAliveInterval)
	keepAliveTimeout := url.GetParamDuration(constant.KeepAliveTimeout, constant.DefaultKeepAliveTimeout)

	if tripleConf == nil {
		return cliKeepAliveOpts, keepAliveInterval, keepAliveTimeout, nil
	}

	var parseErr error

	if tripleConf.KeepAliveInterval != "" {
		keepAliveInterval, parseErr = time.ParseDuration(tripleConf.KeepAliveInterval)
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
	}
	if tripleConf.KeepAliveTimeout != "" {
		keepAliveTimeout, parseErr = time.ParseDuration(tripleConf.KeepAliveTimeout)
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
	}

	return cliKeepAliveOpts, keepAliveInterval, keepAliveTimeout, nil
}

// dualTransport is a transport that can handle both HTTP/2 and HTTP/3
// It uses HTTP Alternative Services (Alt-Svc) for protocol negotiation
type dualTransport struct {
	http2Transport *http2.Transport
	http3Transport *http3.Transport
	// Cache for alternative services to avoid repeated lookups
	altSvcCache *tri.AltSvcCache
}

// newDualTransport creates a new dual transport that supports both HTTP/2 and HTTP/3
func newDualTransport(tlsConfig *tls.Config, keepAliveInterval, keepAliveTimeout time.Duration) http.RoundTripper {
	http2Transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
		ReadIdleTimeout: keepAliveInterval,
		PingTimeout:     keepAliveTimeout,
	}

	http3Transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig: &quic.Config{
			KeepAlivePeriod: keepAliveInterval,
			MaxIdleTimeout:  keepAliveTimeout,
		},
	}

	return &dualTransport{
		http2Transport: http2Transport,
		http3Transport: http3Transport,
		altSvcCache:    tri.NewAltSvcCache(),
	}
}

// RoundTrip implements http.RoundTripper interface with HTTP Alternative Services support
func (dt *dualTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if we have cached alternative service information
	cachedAltSvc := dt.altSvcCache.Get(req.URL.Host)

	// If we have valid cached alt-svc info and it's for HTTP/3, try HTTP/3 first
	// Check if the cached information is still valid (not expired)
	if cachedAltSvc != nil && cachedAltSvc.Protocol == "h3" {
		logger.Debugf("Using cached HTTP/3 alternative service for %s", req.URL.String())
		resp, err := dt.http3Transport.RoundTrip(req)
		if err == nil {
			// Update alt-svc cache from response headers
			dt.altSvcCache.UpdateFromHeaders(req.URL.Host, resp.Header)
			return resp, nil
		}
		logger.Debugf("Cached HTTP/3 request failed to %s, falling back to HTTP/2: %v", req.URL.String(), err)
	}

	// Start with HTTP/2 to get alternative service information
	logger.Debugf("Making initial HTTP/2 request to %s to discover alternative services", req.URL.String())
	resp, err := dt.http2Transport.RoundTrip(req)
	if err != nil {
		logger.Errorf("HTTP/2 request failed to %s: %v", req.URL.String(), err)
		return nil, err
	}

	// Check for alternative services in the response
	dt.altSvcCache.UpdateFromHeaders(req.URL.Host, resp.Header)

	// If the response indicates HTTP/3 is available, try HTTP/3 for future requests
	if altSvc := dt.altSvcCache.Get(req.URL.Host); altSvc != nil && altSvc.Protocol == "h3" {
		logger.Debugf("Server %s supports HTTP/3, will use HTTP/3 for future requests", req.URL.Host)
	}

	return resp, nil
}

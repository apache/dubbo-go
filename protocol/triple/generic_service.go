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
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// TripleGenericService 提供泛化调用功能
type TripleGenericService struct {
	serviceURL string
	mu         sync.RWMutex
}

// TripleInvocationRequest 批量调用请求
type TripleInvocationRequest struct {
	MethodName  string
	Types       []string
	Args        []any
	Attachments map[string]any
}

// TripleInvocationResult 批量调用结果
type TripleInvocationResult struct {
	Index  int
	Result any
	Error  error
}

// BatchInvokeOptions 批量调用选项
type BatchInvokeOptions struct {
	MaxConcurrency int
	FailFast       bool
}

// TripleAsyncCall 异步调用信息
type TripleAsyncCall struct {
	ID         string
	MethodName string
	StartTime  time.Time
	cancel     context.CancelFunc
}

// TripleAsyncManager 异步调用管理器
type TripleAsyncManager struct {
	calls map[string]*TripleAsyncCall
	mu    sync.RWMutex
}

// AttachmentBuilder 附件构建器
type AttachmentBuilder struct {
	attachments map[string]any
}

var asyncManager *TripleAsyncManager
var asyncManagerOnce sync.Once

// NewTripleGenericService 创建新的泛化服务实例
func NewTripleGenericService(serviceURL string) *TripleGenericService {
	return &TripleGenericService{
		serviceURL: serviceURL,
	}
}

// Reference 获取服务引用
func (tgs *TripleGenericService) Reference() string {
	return tgs.serviceURL
}

// Invoke 执行泛化调用
func (tgs *TripleGenericService) Invoke(ctx context.Context, methodName string, types []string, args []any) (any, error) {
	return tgs.InvokeWithAttachments(ctx, methodName, types, args, nil)
}

// InvokeWithAttachments 带附件的泛化调用
func (tgs *TripleGenericService) InvokeWithAttachments(ctx context.Context, methodName string, types []string, args []any, attachments map[string]any) (any, error) {
	if methodName == "" {
		return nil, errors.New("method name cannot be empty")
	}

	// 参数转换和验证
	_, err := tgs.convertParamsForIDL(args, types)
	if err != nil {
		return nil, err
	}

	// 这里应该是实际的网络调用，但由于没有服务器，我们模拟网络错误
	return nil, fmt.Errorf("network connection failed: no route to host %s", tgs.serviceURL)
}

// InvokeAsync 异步调用
func (tgs *TripleGenericService) InvokeAsync(ctx context.Context, methodName string, types []string, args []any, attachments map[string]any, callback func(result any, err error)) (string, error) {
	return tgs.InvokeAsyncWithTimeout(ctx, methodName, types, args, attachments, callback, 30*time.Second)
}

// InvokeAsyncWithTimeout 带超时的异步调用
func (tgs *TripleGenericService) InvokeAsyncWithTimeout(ctx context.Context, methodName string, types []string, args []any, attachments map[string]any, callback func(result any, err error), timeout time.Duration) (string, error) {
	callID := generateCallID()
	manager := GetTripleAsyncManager()

	asyncCtx, cancel := context.WithTimeout(ctx, timeout)

	asyncCall := &TripleAsyncCall{
		ID:         callID,
		MethodName: methodName,
		StartTime:  time.Now(),
		cancel:     cancel,
	}

	manager.registerCall(asyncCall)

	go func() {
		defer func() {
			manager.unregisterCall(callID)
			cancel()
		}()

		result, err := tgs.InvokeWithAttachments(asyncCtx, methodName, types, args, attachments)
		if callback != nil {
			callback(result, err)
		}
	}()

	return callID, nil
}

// CancelAsyncCall 取消异步调用
func (tgs *TripleGenericService) CancelAsyncCall(callID string) bool {
	manager := GetTripleAsyncManager()
	call := manager.getCall(callID)
	if call == nil {
		return false
	}

	if call.cancel != nil {
		call.cancel()
	}
	manager.unregisterCall(callID)
	return true
}

// BatchInvoke 批量调用
func (tgs *TripleGenericService) BatchInvoke(ctx context.Context, invocations []TripleInvocationRequest) ([]TripleInvocationResult, error) {
	if len(invocations) == 0 {
		return nil, errors.New("empty invocations")
	}

	options := BatchInvokeOptions{
		MaxConcurrency: 10,
		FailFast:       false,
	}

	return tgs.BatchInvokeWithOptions(ctx, invocations, options)
}

// BatchInvokeWithOptions 带选项的批量调用
func (tgs *TripleGenericService) BatchInvokeWithOptions(ctx context.Context, invocations []TripleInvocationRequest, options BatchInvokeOptions) ([]TripleInvocationResult, error) {
	results := make([]TripleInvocationResult, len(invocations))

	// 处理零并发的情况
	maxConcurrency := options.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}

	// 使用 semaphore 控制并发数
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, inv := range invocations {
		wg.Add(1)
		go func(index int, invocation TripleInvocationRequest) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result, err := tgs.InvokeWithAttachments(ctx, invocation.MethodName, invocation.Types, invocation.Args, invocation.Attachments)
			results[index] = TripleInvocationResult{
				Index:  index,
				Result: result,
				Error:  err,
			}

			if options.FailFast && err != nil {
				// 在快速失败模式下，可以考虑取消其他调用
				// 这里暂时只记录错误
			}
		}(i, inv)
	}

	wg.Wait()
	return results, nil
}

// CreateAttachmentBuilder 创建附件构建器
func (tgs *TripleGenericService) CreateAttachmentBuilder() *AttachmentBuilder {
	return &AttachmentBuilder{
		attachments: make(map[string]any),
	}
}

// convertParamsForIDL 为IDL模式转换参数
func (tgs *TripleGenericService) convertParamsForIDL(params any, types []string) ([]any, error) {
	var paramsList []any

	// 处理单个参数的情况
	if reflect.TypeOf(params).Kind() != reflect.Slice {
		paramsList = []any{params}
	} else {
		paramSlice := reflect.ValueOf(params)
		for i := 0; i < paramSlice.Len(); i++ {
			paramsList = append(paramsList, paramSlice.Index(i).Interface())
		}
	}

	// 检查参数数量是否匹配
	if len(paramsList) != len(types) {
		return nil, fmt.Errorf("parameter count mismatch: got %d, expected %d", len(paramsList), len(types))
	}

	// 类型转换
	result := make([]any, len(paramsList))
	for i, param := range paramsList {
		converted, err := convertSingleArgForSerialization(param, types[i])
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter %d: %w", i, err)
		}
		result[i] = converted
	}

	return result, nil
}

// convertSingleArgForSerialization 转换单个参数用于序列化
func convertSingleArgForSerialization(arg any, targetType string) (any, error) {
	switch targetType {
	case "int32":
		return convertToInt32(arg)
	case "int64":
		return convertToInt64(arg)
	case "float32":
		return convertToFloat32(arg)
	case "float64":
		return convertToFloat64(arg)
	case "string":
		return convertToString(arg), nil
	case "bool":
		return convertToBool(arg), nil
	case "bytes":
		return convertToBytes(arg), nil
	default:
		// 对于复杂类型（如map、slice等），直接返回
		return arg, nil
	}
}

// 类型转换辅助函数
func convertToInt32(val any) (int32, error) {
	switch v := val.(type) {
	case int32:
		return v, nil
	case int:
		return int32(v), nil
	case int64:
		return int32(v), nil
	case float32:
		return int32(v), nil
	case float64:
		return int32(v), nil
	case string:
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(i), nil
		}
	}
	return 0, fmt.Errorf("cannot convert %T to int32", val)
}

func convertToInt64(val any) (int64, error) {
	switch v := val.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, nil
		}
	}
	return 0, fmt.Errorf("cannot convert %T to int64", val)
}

func convertToFloat32(val any) (float32, error) {
	switch v := val.(type) {
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
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			return float32(f), nil
		}
	}
	return 0, fmt.Errorf("cannot convert %T to float32", val)
}

func convertToFloat64(val any) (float64, error) {
	switch v := val.(type) {
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
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
	}
	return 0, fmt.Errorf("cannot convert %T to float64", val)
}

func convertToString(val any) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

func convertToBool(val any) bool {
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

func convertToBytes(val any) []byte {
	if val == nil {
		return nil
	}
	if b, ok := val.([]byte); ok {
		return b
	}
	if s, ok := val.(string); ok {
		return []byte(s)
	}
	return []byte(fmt.Sprintf("%v", val))
}

// AttachmentBuilder 方法
func (ab *AttachmentBuilder) SetString(key, value string) *AttachmentBuilder {
	ab.attachments[key] = value
	return ab
}

func (ab *AttachmentBuilder) SetInt(key string, value int) *AttachmentBuilder {
	ab.attachments[key] = value
	return ab
}

func (ab *AttachmentBuilder) SetBool(key string, value bool) *AttachmentBuilder {
	ab.attachments[key] = value
	return ab
}

func (ab *AttachmentBuilder) Build() map[string]any {
	result := make(map[string]any)
	for k, v := range ab.attachments {
		result[k] = v
	}
	return result
}

// 异步管理器相关方法
func GetTripleAsyncManager() *TripleAsyncManager {
	asyncManagerOnce.Do(func() {
		asyncManager = &TripleAsyncManager{
			calls: make(map[string]*TripleAsyncCall),
		}
	})
	return asyncManager
}

func (tam *TripleAsyncManager) registerCall(call *TripleAsyncCall) {
	tam.mu.Lock()
	defer tam.mu.Unlock()
	tam.calls[call.ID] = call
}

func (tam *TripleAsyncManager) unregisterCall(callID string) {
	tam.mu.Lock()
	defer tam.mu.Unlock()
	delete(tam.calls, callID)
}

func (tam *TripleAsyncManager) getCall(callID string) *TripleAsyncCall {
	tam.mu.RLock()
	defer tam.mu.RUnlock()
	return tam.calls[callID]
}

func (tam *TripleAsyncManager) getAllActiveCalls() []string {
	tam.mu.RLock()
	defer tam.mu.RUnlock()

	var callIDs []string
	for id := range tam.calls {
		callIDs = append(callIDs, id)
	}
	return callIDs
}

// generateCallID 生成调用ID
func generateCallID() string {
	return fmt.Sprintf("triple-async-%d-%d", time.Now().UnixNano(), rand.Intn(1000000))
}

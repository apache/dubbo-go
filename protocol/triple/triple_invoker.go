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
	"reflect"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

var triAttachmentKeys = []string{
	constant.InterfaceKey, constant.TokenKey, constant.TimeoutKey,
}

// Triple heartbeat constants
const (
	DefaultHeartbeatInterval = 30 * time.Second // 默认心跳间隔
	DefaultHeartbeatTimeout  = 5 * time.Second  // 默认心跳超时
	HeartbeatMethod          = "$heartbeat"     // 心跳方法名
	HeartbeatMaxRetries      = 3                // 最大重试次数
)

// TripleMetricsEvent represents a metrics event for Triple protocol
type TripleMetricsEvent struct {
	Name       string
	Invoker    base.Invoker
	Invocation base.Invocation
	Result     result.Result
	CostTime   time.Duration
}

// Type returns the type of the event
func (tme *TripleMetricsEvent) Type() string {
	return constant.MetricsRpc
}

// NewTripleMetricsEvent creates a new TripleMetricsEvent
func NewTripleMetricsEvent(name string, invoker base.Invoker, invocation base.Invocation) *TripleMetricsEvent {
	return &TripleMetricsEvent{
		Name:       name,
		Invoker:    invoker,
		Invocation: invocation,
	}
}

// WithResult sets the result for the metrics event
func (tme *TripleMetricsEvent) WithResult(result result.Result) *TripleMetricsEvent {
	tme.Result = result
	return tme
}

// WithCostTime sets the cost time for the metrics event
func (tme *TripleMetricsEvent) WithCostTime(costTime time.Duration) *TripleMetricsEvent {
	tme.CostTime = costTime
	return tme
}

type TripleInvoker struct {
	base.BaseInvoker
	quitOnce          sync.Once
	clientGuard       *sync.RWMutex
	clientManager     *clientManager
	heartbeatTicker   *time.Ticker
	heartbeatStop     chan struct{}
	heartbeatEnabled  bool
	lastHeartbeatTime time.Time
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
}

func (ti *TripleInvoker) setClientManager(cm *clientManager) {
	ti.clientGuard.Lock()
	defer ti.clientGuard.Unlock()

	ti.clientManager = cm
}

func (ti *TripleInvoker) getClientManager() *clientManager {
	ti.clientGuard.RLock()
	defer ti.clientGuard.RUnlock()

	return ti.clientManager
}

// Invoke is used to call client-side method.
func (ti *TripleInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	startTime := time.Now()

	// Send BeforeInvoke metrics event
	metrics.Publish(NewTripleMetricsEvent("before_invoke", ti, invocation))

	// Debug logging removed for cleaner test output
	var result result.RPCResult

	if !ti.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("TripleInvoker is destroyed")
		result.SetError(base.ErrDestroyedInvoker)

		// Send AfterInvoke metrics event
		costTime := time.Since(startTime)
		metrics.Publish(NewTripleMetricsEvent("after_invoke", ti, invocation).WithResult(&result).WithCostTime(costTime))

		return &result
	}

	ti.clientGuard.RLock()
	defer ti.clientGuard.RUnlock()

	if ti.clientManager == nil {
		result.SetError(base.ErrClientClosed)

		// Send AfterInvoke metrics event
		costTime := time.Since(startTime)
		metrics.Publish(NewTripleMetricsEvent("after_invoke", ti, invocation).WithResult(&result).WithCostTime(costTime))

		return &result
	}

	callType, inRaw, in, method, err := parseInvocation(ctx, ti.GetURL(), invocation)
	if err != nil {
		result.SetError(err)

		// Send AfterInvoke metrics event
		costTime := time.Since(startTime)
		metrics.Publish(NewTripleMetricsEvent("after_invoke", ti, invocation).WithResult(&result).WithCostTime(costTime))

		return &result
	}

	ctx, err = mergeAttachmentToOutgoing(ctx, invocation)
	if err != nil {
		result.SetError(err)

		// Send AfterInvoke metrics event
		costTime := time.Since(startTime)
		metrics.Publish(NewTripleMetricsEvent("after_invoke", ti, invocation).WithResult(&result).WithCostTime(costTime))

		return &result
	}

	inRawLen := len(inRaw)

	generic, ok := invocation.GetAttachment(constant.GenericKey)

	logger.Warnf("in len: %v", len(in))

	reqParams := make([]any, 0, len(in))
	if ok && generic == "true" {
		for i, v := range in {
			logger.Warnf("i: %v", i)
			reqParams = append(reqParams, v.Interface())
		}
	}

	logger.Warnf("reqParams len: %v", len(reqParams))

	if !ti.clientManager.isIDL {
		switch callType {
		case constant.CallUnary:
			// todo(DMwangnima): consider inRawLen == 0
			if inRawLen != 0 {
				if err := ti.clientManager.callUnary(ctx, method, inRaw[0:inRawLen-1], inRaw[inRawLen-1]); err != nil {
					result.SetError(err)
				}
			} else {
				if err := ti.clientManager.callUnary(ctx, method, reqParams, reqParams); err != nil {
					result.SetError(err)
				}
			}
		default:
			panic("Triple only supports Unary Invocation for Non-IDL mode")
		}
		return &result
	}
	switch callType {
	case constant.CallUnary:
		if len(inRaw) != 2 {
			panic(fmt.Sprintf("Wrong parameter Values number for CallUnary, want 2, but got %d", inRawLen))
		}
		if err := ti.clientManager.callUnary(ctx, method, inRaw[0], inRaw[1]); err != nil {
			result.SetError(err)
		}
	case constant.CallClientStream:
		stream, err := ti.clientManager.callClientStream(ctx, method)
		if err != nil {
			result.SetError(err)
		}
		result.SetResult(stream)
	case constant.CallServerStream:
		if inRawLen != 1 {
			panic(fmt.Sprintf("Wrong parameter Values number for CallServerStream, want 1, but got %d", inRawLen))
		}
		stream, err := ti.clientManager.callServerStream(ctx, method, inRaw[0])
		if err != nil {
			result.Err = err
		}
		result.SetResult(stream)
	case constant.CallBidiStream:
		stream, err := ti.clientManager.callBidiStream(ctx, method)
		if err != nil {
			result.Err = err
		}
		result.SetResult(stream)
	default:
		panic(fmt.Sprintf("Unsupported CallType: %s", callType))
	}

	if header, ok := tri.FromIncomingContext(ctx); ok {
		for v, k := range header {
			if tri.IsReservedHeader(v) {
				continue
			}
			result.AddAttachment(v, k)
		}
	}

	// Send AfterInvoke metrics event
	costTime := time.Since(startTime)
	metrics.Publish(NewTripleMetricsEvent("after_invoke", ti, invocation).WithResult(&result).WithCostTime(costTime))

	return &result
}

func mergeAttachmentToOutgoing(ctx context.Context, inv base.Invocation) (context.Context, error) {
	// Todo(finalt) Temporarily solve the problem that the timeout time is not valid
	if timeout, ok := inv.GetAttachment(constant.TimeoutKey); ok {
		ctx = context.WithValue(ctx, tri.TimeoutKey{}, timeout)
	}
	for key, valRaw := range inv.Attachments() {
		if str, ok := valRaw.(string); ok {
			ctx = tri.AppendToOutgoingContext(ctx, key, str)
			continue
		}
		if strs, ok := valRaw.([]string); ok {
			for _, str := range strs {
				ctx = tri.AppendToOutgoingContext(ctx, key, str)
			}
			continue
		}
		return ctx, fmt.Errorf("triple attachments value with key = %s is invalid, which should be string or []string", key)
	}
	return ctx, nil
}

// parseInvocation retrieves information from invocation.
// it returns ctx, callType, inRaw, method, error
func parseInvocation(ctx context.Context, url *common.URL, invocation base.Invocation) (string, []any, []reflect.Value, string, error) {
	callTypeRaw, ok := invocation.GetAttribute(constant.CallTypeKey)
	if !ok {
		return "", nil, nil, "", errors.New("miss CallType in invocation to invoke TripleInvoker")
	}
	callType, ok := callTypeRaw.(string)
	if !ok {
		return "", nil, nil, "", fmt.Errorf("CallType should be string, but got %v", callTypeRaw)
	}
	// please refer to methods of client.Client or code generated by new triple for the usage of inRaw and inRawLen
	// e.g. Client.CallUnary(... req, resp []interface, ...)
	// inRaw represents req and resp
	inRaw := invocation.ParameterRawValues()
	in := invocation.ParameterValues()
	method := invocation.MethodName()
	if method == "" {
		return "", nil, nil, "", errors.New("miss MethodName in invocation to invoke TripleInvoker")
	}

	parseAttachments(ctx, url, invocation)

	return callType, inRaw, in, method, nil
}

// parseAttachments retrieves attachments from users passed-in and URL, then injects them into ctx
func parseAttachments(ctx context.Context, url *common.URL, invocation base.Invocation) {
	// retrieve users passed-in attachment
	attaRaw := ctx.Value(constant.AttachmentKey)
	if attaRaw != nil {
		if userAtta, ok := attaRaw.(map[string]any); ok {
			for key, val := range userAtta {
				invocation.SetAttachment(key, val)
			}
		}
	}
	// set pre-defined attachments
	for _, key := range triAttachmentKeys {
		if val := url.GetParam(key, ""); len(val) > 0 {
			invocation.SetAttachment(key, val)
		}
	}
}

// IsAvailable get available status
func (ti *TripleInvoker) IsAvailable() bool {
	if ti.getClientManager() != nil {
		return ti.BaseInvoker.IsAvailable()
	}

	return false
}

// IsHeartbeatEnabled checks if heartbeat is enabled
func (ti *TripleInvoker) IsHeartbeatEnabled() bool {
	return ti.heartbeatEnabled
}

// EnableHeartbeat enables heartbeat for the invoker
func (ti *TripleInvoker) EnableHeartbeat() {
	if ti.heartbeatEnabled {
		return
	}

	ti.heartbeatEnabled = true
	ti.heartbeatInterval = DefaultHeartbeatInterval
	ti.heartbeatTimeout = DefaultHeartbeatTimeout
	ti.heartbeatStop = make(chan struct{})
	ti.lastHeartbeatTime = time.Now()

	// Check URL for custom heartbeat configuration
	url := ti.GetURL()
	if intervalStr := url.GetParam("heartbeat.interval", ""); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			ti.heartbeatInterval = interval
		}
	}
	if timeoutStr := url.GetParam("heartbeat.timeout", ""); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			ti.heartbeatTimeout = timeout
		}
	}

	ti.heartbeatTicker = time.NewTicker(ti.heartbeatInterval)
	go ti.heartbeatLoop()

	logger.Infof("Triple heartbeat enabled for invoker %s, interval: %v, timeout: %v",
		ti.GetURL().ServiceKey(), ti.heartbeatInterval, ti.heartbeatTimeout)
}

// DisableHeartbeat disables heartbeat for the invoker
func (ti *TripleInvoker) DisableHeartbeat() {
	if !ti.heartbeatEnabled {
		return
	}

	ti.heartbeatEnabled = false
	if ti.heartbeatTicker != nil {
		ti.heartbeatTicker.Stop()
		ti.heartbeatTicker = nil
	}
	if ti.heartbeatStop != nil {
		close(ti.heartbeatStop)
		ti.heartbeatStop = nil
	}

	logger.Infof("Triple heartbeat disabled for invoker %s", ti.GetURL().ServiceKey())
}

// heartbeatLoop runs the heartbeat loop
func (ti *TripleInvoker) heartbeatLoop() {
	for {
		select {
		case <-ti.heartbeatTicker.C:
			ti.sendHeartbeat()
		case <-ti.heartbeatStop:
			logger.Infof("Triple heartbeat loop stopped for invoker %s", ti.GetURL().ServiceKey())
			return
		}
	}
}

// sendHeartbeat sends a heartbeat request
func (ti *TripleInvoker) sendHeartbeat() {
	if !ti.IsAvailable() {
		logger.Warnf("Triple invoker %s is not available, skipping heartbeat", ti.GetURL().ServiceKey())
		return
	}

	// Create heartbeat invocation
	heartbeatInvocation := invocation.NewRPCInvocation(HeartbeatMethod, []any{}, make(map[string]any))
	heartbeatInvocation.SetAttachment("heartbeat", "true")
	heartbeatInvocation.SetAttachment("timestamp", time.Now().Unix())

	// Set timeout for heartbeat
	ctx, cancel := context.WithTimeout(context.Background(), ti.heartbeatTimeout)
	defer cancel()

	logger.Debugf("Sending heartbeat to %s", ti.GetURL().ServiceKey())

	startTime := time.Now()
	result := ti.Invoke(ctx, heartbeatInvocation)
	costTime := time.Since(startTime)

	if result.Error() != nil {
		logger.Warnf("Triple heartbeat failed for %s, error: %v, cost: %v",
			ti.GetURL().ServiceKey(), result.Error(), costTime)

		// Send heartbeat failure metrics event
		metrics.Publish(NewTripleMetricsEvent("heartbeat_failed", ti, heartbeatInvocation).
			WithResult(result).
			WithCostTime(costTime))

		// Mark as unavailable if multiple failures
		// In a real implementation, you might want to implement a failure counter
	} else {
		ti.lastHeartbeatTime = time.Now()
		logger.Debugf("Triple heartbeat success for %s, cost: %v", ti.GetURL().ServiceKey(), costTime)

		// Send heartbeat success metrics event
		metrics.Publish(NewTripleMetricsEvent("heartbeat_success", ti, heartbeatInvocation).
			WithResult(result).
			WithCostTime(costTime))
	}
}

// GetLastHeartbeatTime returns the last heartbeat time
func (ti *TripleInvoker) GetLastHeartbeatTime() time.Time {
	return ti.lastHeartbeatTime
}

// GetHeartbeatInterval returns the heartbeat interval
func (ti *TripleInvoker) GetHeartbeatInterval() time.Duration {
	return ti.heartbeatInterval
}

// GetHeartbeatTimeout returns the heartbeat timeout
func (ti *TripleInvoker) GetHeartbeatTimeout() time.Duration {
	return ti.heartbeatTimeout
}

// IsHeartbeatHealthy checks if the heartbeat is healthy
func (ti *TripleInvoker) IsHeartbeatHealthy() bool {
	if !ti.heartbeatEnabled {
		return true // If heartbeat is disabled, consider it healthy
	}

	timeSinceLastHeartbeat := time.Since(ti.lastHeartbeatTime)
	maxAllowedTime := ti.heartbeatInterval + ti.heartbeatTimeout

	return timeSinceLastHeartbeat <= maxAllowedTime
}

// IsDestroyed get destroyed status
func (ti *TripleInvoker) IsDestroyed() bool {
	if ti.getClientManager() != nil {
		return ti.BaseInvoker.IsDestroyed()
	}

	return false
}

// Destroy will destroy Triple's invoker and client, so it is only called once
func (ti *TripleInvoker) Destroy() {
	ti.quitOnce.Do(func() {
		// Stop heartbeat first
		ti.DisableHeartbeat()

		ti.BaseInvoker.Destroy()
		if cm := ti.getClientManager(); cm != nil {
			ti.setClientManager(nil)
			// todo:// find a better way to destroy these resources
			cm.close()
		}
	})
}

func NewTripleInvoker(url *common.URL) (*TripleInvoker, error) {
	cm, err := newClientManager(url)
	if err != nil {
		return nil, err
	}

	invoker := &TripleInvoker{
		BaseInvoker:       *base.NewBaseInvoker(url),
		quitOnce:          sync.Once{},
		clientGuard:       &sync.RWMutex{},
		clientManager:     cm,
		heartbeatEnabled:  false,
		heartbeatInterval: DefaultHeartbeatInterval,
		heartbeatTimeout:  DefaultHeartbeatTimeout,
		lastHeartbeatTime: time.Now(),
	}

	// Check if heartbeat is enabled by URL parameter
	if heartbeatEnabled := url.GetParamBool("heartbeat.enabled", false); heartbeatEnabled {
		invoker.EnableHeartbeat()
	}

	return invoker, nil
}

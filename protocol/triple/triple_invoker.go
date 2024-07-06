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
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

var triAttachmentKeys = []string{
	constant.InterfaceKey, constant.TokenKey, constant.TimeoutKey,
}

type TripleInvoker struct {
	protocol.BaseInvoker
	quitOnce      sync.Once
	clientGuard   *sync.RWMutex
	clientManager *clientManager
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
func (ti *TripleInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var result protocol.RPCResult

	if !ti.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("TripleInvoker is destroyed")
		result.SetError(protocol.ErrDestroyedInvoker)
		return &result
	}

	ti.clientGuard.RLock()
	defer ti.clientGuard.RUnlock()

	if ti.clientManager == nil {
		result.SetError(protocol.ErrClientClosed)
		return &result
	}

	callType, inRaw, method, err := parseInvocation(ctx, ti.GetURL(), invocation)
	if err != nil {
		result.SetError(err)
		return &result
	}

	ctx, err = mergeAttachmentToOutgoing(ctx, invocation)
	if err != nil {
		result.SetError(err)
		return &result
	}

	inRawLen := len(inRaw)

	if !ti.clientManager.isIDL {
		switch callType {
		case constant.CallUnary:
			// todo(DMwangnima): consider inRawLen == 0
			if err := ti.clientManager.callUnary(ctx, method, inRaw[0:inRawLen-1], inRaw[inRawLen-1]); err != nil {
				result.SetError(err)
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

	return &result
}

func mergeAttachmentToOutgoing(ctx context.Context, inv protocol.Invocation) (context.Context, error) {
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
func parseInvocation(ctx context.Context, url *common.URL, invocation protocol.Invocation) (string, []interface{}, string, error) {
	callTypeRaw, ok := invocation.GetAttribute(constant.CallTypeKey)
	if !ok {
		return "", nil, "", errors.New("miss CallType in invocation to invoke TripleInvoker")
	}
	callType, ok := callTypeRaw.(string)
	if !ok {
		return "", nil, "", fmt.Errorf("CallType should be string, but got %v", callTypeRaw)
	}
	// please refer to methods of client.Client or code generated by new triple for the usage of inRaw and inRawLen
	// e.g. Client.CallUnary(... req, resp []interface, ...)
	// inRaw represents req and resp
	inRaw := invocation.ParameterRawValues()
	method := invocation.MethodName()
	if method == "" {
		return "", nil, "", errors.New("miss MethodName in invocation to invoke TripleInvoker")
	}

	parseAttachments(ctx, url, invocation)

	return callType, inRaw, method, nil
}

// parseAttachments retrieves attachments from users passed-in and URL, then injects them into ctx
func parseAttachments(ctx context.Context, url *common.URL, invocation protocol.Invocation) {
	// retrieve users passed-in attachment
	attaRaw := ctx.Value(constant.AttachmentKey)
	if attaRaw != nil {
		if userAtta, ok := attaRaw.(map[string]interface{}); ok {
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
	return &TripleInvoker{
		BaseInvoker:   *protocol.NewBaseInvoker(url),
		quitOnce:      sync.Once{},
		clientGuard:   &sync.RWMutex{},
		clientManager: cm,
	}, nil
}

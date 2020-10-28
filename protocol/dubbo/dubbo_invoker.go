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

package dubbo

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/opentracing/opentracing-go"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
)

var (
	// ErrNoReply
	ErrNoReply = perrors.New("request need @response")
	// ErrDestroyedInvoker
	ErrDestroyedInvoker = perrors.New("request Destroyed invoker")
)

var (
	attachmentKey = []string{constant.INTERFACE_KEY, constant.GROUP_KEY, constant.TOKEN_KEY, constant.TIMEOUT_KEY,
		constant.VERSION_KEY}
)

// DubboInvoker is implement of protocol.Invoker. A dubboInvoker refer to one service and ip.
type DubboInvoker struct {
	protocol.BaseInvoker
	// the exchange layer, it is focus on network communication.
	client   *remoting.ExchangeClient
	quitOnce sync.Once
	// timeout for service(interface) level.
	timeout time.Duration
	// Used to record the number of requests. -1 represent this DubboInvoker is destroyed
	reqNum int64
}

// NewDubboInvoker constructor
func NewDubboInvoker(url common.URL, client *remoting.ExchangeClient) *DubboInvoker {
	requestTimeout := config.GetConsumerConfig().RequestTimeout

	requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
		requestTimeout = t
	}
	return &DubboInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
		reqNum:      0,
		timeout:     requestTimeout,
	}
}

// Invoke call remoting.
func (di *DubboInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var (
		err    error
		result protocol.RPCResult
	)
	if di.reqNum < 0 {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("this dubboInvoker is destroyed")
		result.Err = ErrDestroyedInvoker
		return &result
	}
	atomic.AddInt64(&(di.reqNum), 1)
	defer atomic.AddInt64(&(di.reqNum), -1)

	inv := invocation.(*invocation_impl.RPCInvocation)
	// init param
	inv.SetAttachments(constant.PATH_KEY, di.GetUrl().GetParam(constant.INTERFACE_KEY, ""))
	for _, k := range attachmentKey {
		if v := di.GetUrl().GetParam(k, ""); len(v) > 0 {
			inv.SetAttachments(k, v)
		}
	}

	// put the ctx into attachment
	di.appendCtx(ctx, inv)

	url := di.GetUrl()
	// default hessian2 serialization, compatible
	if url.GetParam(constant.SERIALIZATION_KEY, "") == "" {
		url.SetParam(constant.SERIALIZATION_KEY, constant.HESSIAN2_SERIALIZATION)
	}
	// async
	async, err := strconv.ParseBool(inv.AttachmentsByKey(constant.ASYNC_KEY, "false"))
	if err != nil {
		logger.Errorf("ParseBool - error: %v", err)
		async = false
	}
	//response := NewResponse(inv.Reply(), nil)
	rest := &protocol.RPCResult{}
	timeout := di.getTimeout(inv)
	if async {
		if callBack, ok := inv.CallBack().(func(response common.CallbackResponse)); ok {
			//result.Err = di.client.AsyncCall(NewRequest(url.Location, url, inv.MethodName(), inv.Arguments(), inv.Attachments()), callBack, response)
			result.Err = di.client.AsyncRequest(&invocation, url, timeout, callBack, rest)
		} else {
			result.Err = di.client.Send(&invocation, url, timeout)
		}
	} else {
		if inv.Reply() == nil {
			result.Err = ErrNoReply
		} else {
			result.Err = di.client.Request(&invocation, url, timeout, rest)
		}
	}
	if result.Err == nil {
		result.Rest = inv.Reply()
		result.Attrs = rest.Attrs
	}
	logger.Debugf("result.Err: %v, result.Rest: %v", result.Err, result.Rest)

	return &result
}

// get timeout including methodConfig
func (di *DubboInvoker) getTimeout(invocation *invocation_impl.RPCInvocation) time.Duration {
	var timeout = di.GetUrl().GetParam(strings.Join([]string{constant.METHOD_KEYS, invocation.MethodName(), constant.TIMEOUT_KEY}, "."), "")
	if len(timeout) != 0 {
		if t, err := time.ParseDuration(timeout); err == nil {
			// config timeout into attachment
			invocation.SetAttachments(constant.TIMEOUT_KEY, strconv.Itoa(int(t.Milliseconds())))
			return t
		}
	}
	// set timeout into invocation at method level
	invocation.SetAttachments(constant.TIMEOUT_KEY, strconv.Itoa(int(di.timeout.Milliseconds())))
	return di.timeout
}

func (di *DubboInvoker) IsAvailable() bool {
	return di.client.IsAvailable()
}

// Destroy destroy dubbo client invoker.
func (di *DubboInvoker) Destroy() {
	di.quitOnce.Do(func() {
		for {
			if di.reqNum == 0 {
				di.reqNum = -1
				logger.Infof("dubboInvoker is destroyed,url:{%s}", di.GetUrl().Key())
				di.BaseInvoker.Destroy()
				if di.client != nil {
					di.client.Close()
					di.client = nil
				}
				break
			}
			logger.Warnf("DubboInvoker is to be destroyed, wait {%v} req end,url:{%s}", di.reqNum, di.GetUrl().Key())
			time.Sleep(1 * time.Second)
		}

	})
}

// Finally, I made the decision that I don't provide a general way to transfer the whole context
// because it could be misused. If the context contains to many key-value pairs, the performance will be much lower.
func (di *DubboInvoker) appendCtx(ctx context.Context, inv *invocation_impl.RPCInvocation) {
	// inject opentracing ctx
	currentSpan := opentracing.SpanFromContext(ctx)
	if currentSpan != nil {
		err := injectTraceCtx(currentSpan, inv)
		if err != nil {
			logger.Errorf("Could not inject the span context into attachments: %v", err)
		}
	}
}

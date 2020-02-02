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
	"sync"
)

import (
	"github.com/opentracing/opentracing-go"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
)

var (
	// ErrNoReply ...
	ErrNoReply = perrors.New("request need @response")
)

var (
	attachmentKey = []string{constant.INTERFACE_KEY, constant.GROUP_KEY, constant.TOKEN_KEY, constant.TIMEOUT_KEY}
)

// DubboInvoker ...
type DubboInvoker struct {
	protocol.BaseInvoker
	client   *Client
	quitOnce sync.Once
}

// NewDubboInvoker ...
func NewDubboInvoker(url common.URL, client *Client) *DubboInvoker {
	return &DubboInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
	}
}

// Invoke ...
func (di *DubboInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var (
		err    error
		result protocol.RPCResult
	)

	inv := invocation.(*invocation_impl.RPCInvocation)
	for _, k := range attachmentKey {
		if v := di.GetUrl().GetParam(k, ""); len(v) > 0 {
			inv.SetAttachments(k, v)
		}
	}

	// put the ctx into attachment
	di.appendCtx(ctx, inv)

	url := di.GetUrl()
	// async
	async, err := strconv.ParseBool(inv.AttachmentsByKey(constant.ASYNC_KEY, "false"))
	if err != nil {
		logger.Errorf("ParseBool - error: %v", err)
		async = false
	}
	response := NewResponse(inv.Reply(), nil)
	if async {
		if callBack, ok := inv.CallBack().(func(response common.CallbackResponse)); ok {
			result.Err = di.client.AsyncCall(NewRequest(url.Location, url, inv.MethodName(), inv.Arguments(), inv.Attachments()), callBack, response)
		} else {
			result.Err = di.client.CallOneway(NewRequest(url.Location, url, inv.MethodName(), inv.Arguments(), inv.Attachments()))
		}
	} else {
		if inv.Reply() == nil {
			result.Err = ErrNoReply
		} else {
			result.Err = di.client.Call(NewRequest(url.Location, url, inv.MethodName(), inv.Arguments(), inv.Attachments()), response)
		}
	}
	if result.Err == nil {
		result.Rest = inv.Reply()
		result.Attrs = response.atta
	}
	logger.Debugf("result.Err: %v, result.Rest: %v", result.Err, result.Rest)

	return &result
}

// Destroy ...
func (di *DubboInvoker) Destroy() {
	di.quitOnce.Do(func() {
		di.BaseInvoker.Destroy()

		if di.client != nil {
			di.client.Close()
		}
	})
}

// Finally, I made the decision that I don't provide a general way to transfer the whole context
// because it could be misused. If the context contains to many key-value pairs, the performance will be much lower.
func (di *DubboInvoker) appendCtx(ctx context.Context, inv *invocation_impl.RPCInvocation) {
	// inject opentracing ctx
	currentSpan := opentracing.SpanFromContext(ctx)
	if currentSpan != nil {
		carrier := opentracing.TextMapCarrier(inv.Attachments())
		err := opentracing.GlobalTracer().Inject(currentSpan.Context(), opentracing.TextMap, carrier)
		if err != nil {
			logger.Errorf("Could not inject the span context into attachments: %v", err)
		}
	}
}

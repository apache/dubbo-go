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

package jsonrpc

import (
	"context"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
)

// JsonrpcInvoker is JSON RPC invoker
type JsonrpcInvoker struct {
	protocol.BaseInvoker
	client *HTTPClient
}

// NewJsonrpcInvoker creates JSON RPC invoker with @url and @client
func NewJsonrpcInvoker(url *common.URL, client *HTTPClient) *JsonrpcInvoker {
	return &JsonrpcInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
	}
}

// Invoke the JSON RPC invocation and return result.
func (ji *JsonrpcInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {

	var (
		result protocol.RPCResult
	)

	inv := invocation.(*invocation_impl.RPCInvocation)
	url := ji.GetUrl()
	req := ji.client.NewRequest(url, inv.MethodName(), inv.Arguments())
	ctxNew := context.WithValue(ctx, constant.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": url.Path,
		"X-Method":   inv.MethodName(),
	})
	result.Err = ji.client.Call(ctxNew, url, req, inv.Reply())
	if result.Err == nil {
		result.Rest = inv.Reply()
	}
	logger.Debugf("result.Err: %v, result.Rest: %v", result.Err, result.Rest)

	return &result
}

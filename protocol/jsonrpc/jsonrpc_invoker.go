// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonrpc

import (
	"context"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	invocation_impl "github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
)

type JsonrpcInvoker struct {
	protocol.BaseInvoker
	client *HTTPClient
}

func NewJsonrpcInvoker(url common.URL, client *HTTPClient) *JsonrpcInvoker {
	return &JsonrpcInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
	}
}

func (ji *JsonrpcInvoker) Invoke(invocation protocol.Invocation) protocol.Result {

	var (
		result protocol.RPCResult
	)

	inv := invocation.(*invocation_impl.RPCInvocation)
	url := ji.GetUrl()
	req := ji.client.NewRequest(url, inv.MethodName(), inv.Arguments())
	ctx := context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": url.Path,
		"X-Method":   inv.MethodName(),
	})
	result.Err = ji.client.Call(ctx, url, req, inv.Reply())
	if result.Err == nil {
		result.Rest = inv.Reply()
	}
	log.Debug("result.Err: %v, result.Rest: %v", result.Err, result.Rest)

	return &result
}

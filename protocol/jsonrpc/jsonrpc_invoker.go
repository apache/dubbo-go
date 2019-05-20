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

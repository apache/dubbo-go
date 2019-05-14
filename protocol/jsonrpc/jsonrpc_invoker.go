package jsonrpc

import (
	"context"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
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
	if err := ji.client.Call(ctx, url, req, inv.Reply()); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		result.Err = err
	} else {
		log.Debug("result: %v", inv.Reply())
		result.Rest = inv.Reply()
	}

	return &result
}

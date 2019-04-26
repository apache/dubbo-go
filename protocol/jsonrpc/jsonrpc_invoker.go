package jsonrpc

import (
	"context"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
	"github.com/dubbo/dubbo-go/public"
)

type JsonrpcInvoker struct {
	protocol.BaseInvoker
	client *HTTPClient
}

func NewJsonrpcInvoker(url config.IURL, client *HTTPClient) *JsonrpcInvoker {
	return &JsonrpcInvoker{
		BaseInvoker: protocol.NewBaseInvoker(url),
		client:      client,
	}
}

func (ji *JsonrpcInvoker) Invoke(invocation protocol.Invocation) protocol.Result {

	var (
		result protocol.RPCResult
	)

	inv := invocation.(*protocol.RPCInvocation)
	url := inv.Invoker().GetUrl().(*config.URL)

	req := ji.client.NewRequest(*url, inv.MethodName(), inv.Arguments())
	ctx := context.WithValue(context.Background(), public.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": url.Service,
		"X-Method":   inv.MethodName(),
	})
	if err := ji.client.Call(ctx, *url, req, inv.Reply()); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		result.Err = err
	} else {
		result.Rest = inv.Reply()
	}

	return &result
}

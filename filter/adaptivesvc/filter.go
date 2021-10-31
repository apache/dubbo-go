package adaptivesvc

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// Filter for adaptive service on server side.
type Filter struct {}

func (f *Filter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	panic("implement me")
}

func (f *Filter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker,
	invocation protocol.Invocation) protocol.Result {
	panic("implement me")
}


package consumer

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"sync"
)

type consumer struct {
	invokerMap  sync.Map
	defCallOpts *CallOptions
}

func (con *consumer) Call(ctx context.Context, opts ...CallOption) error {
	// get a default CallOptions
	// apply CallOption
	options := con.defCallOpts
	for _, opt := range opts {
		opt(options)
	}
	// todo:// think about a identifier to refer invoker

	var invoker protocol.Invoker
	val, ok := con.invokerMap.Load("")
	if !ok {
		// create a new invoker and set it to invokerMap
	}
	invoker, ok = val.(protocol.Invoker)
	if !ok {
		panic("wrong type")
	}
	inv, err := callOptions2Invocation(options)
	if err != nil {
		return err
	}
	// todo: move timeout into context or invocation
	res := invoker.Invoke(ctx, inv)
	// todo: process result and error

}

func callOptions2Invocation(opts *CallOptions) (protocol.Invocation, error) {
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(""),
		invocation.WithArguments(nil),
		invocation.WithCallBack(nil),
		invocation.WithParameterValues(nil),
	)
	return inv, nil
}

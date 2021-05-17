package cluster_impl

import (
	"context"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

// InterceptorInvoker mocks cluster interceptor as an invoker
type InterceptorInvoker struct {
	next        protocol.Invoker
	interceptor cluster.Interceptor
}

// GetURL is used to get url from InterceptorInvoker
func (i *InterceptorInvoker) GetURL() *common.URL {
	return i.next.GetURL()
}

// IsAvailable is used to get available status
func (i *InterceptorInvoker) IsAvailable() bool {
	return i.next.IsAvailable()
}

// Invoke is used to call service method by invocation
func (i *InterceptorInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	return i.interceptor.Invoke(ctx, i.next, invocation)
}

// Destroy will destroy invoker
func (i *InterceptorInvoker) Destroy() {
	i.next.Destroy()
}

func buildInterceptorChain(invoker protocol.Invoker, builtins ...cluster.Interceptor) protocol.Invoker {
	// The order of interceptors is from left to right, so loading from right to left
	next := invoker
	interceptors := extension.GetClusterInterceptors()
	if len(interceptors) != 0 {
		for i := len(interceptors) - 1; i >= 0; i-- {
			v := &InterceptorInvoker{next: next, interceptor: interceptors[i]}
			next = v
		}
	}

	if builtins != nil && len(builtins) > 0 {
		for i := len(builtins) - 1; i >= 0; i-- {
			v := &InterceptorInvoker{next: next, interceptor: builtins[i]}
			next = v
		}
	}

	return next
}

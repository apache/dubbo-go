package auth

import (
	"context"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

// This filter is used to verify the correctness of the signature on provider side
type ProviderAuthFilter struct {
}

func init() {
	extension.SetFilter(constant.PROVIDER_AUTH_FILTER, getProviderAuthFilter)
}

func (paf *ProviderAuthFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking providerAuth filter.")
	url := invoker.GetUrl()

	err := doAuthWork(&url, func(authenticator filter.Authenticator) error {
		return authenticator.Authenticate(invocation, &url)
	})
	if err != nil {
		logger.Infof("auth the request: %v occur exception, cause: %s", invocation, err.Error())
		return &protocol.RPCResult{
			Err: err,
		}
	}

	return invoker.Invoke(ctx, invocation)
}

func (paf *ProviderAuthFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}
func getProviderAuthFilter() filter.Filter {
	return &ProviderAuthFilter{}
}

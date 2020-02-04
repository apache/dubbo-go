package auth

import (
	"context"
	"fmt"
)
import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

// This filter is working for signing the request on consumer side
type ConsumerSignFilter struct {
}

func init() {
	extension.SetFilter(constant.CONSUMER_SIGN_FILTER, getConsumerSignFilter)
}

func (csf *ConsumerSignFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking ConsumerSign filter.")
	url := invoker.GetUrl()

	err := doAuthWork(&url, func(authenticator filter.Authenticator) error {
		return authenticator.Sign(invocation, &url)
	})
	if err != nil {
		panic(fmt.Sprintf("Sign for invocation %s # %s failed", url.ServiceKey(), invocation.MethodName()))

	}
	return invoker.Invoke(ctx, invocation)
}

func (csf *ConsumerSignFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}
func getConsumerSignFilter() filter.Filter {
	return &ConsumerSignFilter{}
}

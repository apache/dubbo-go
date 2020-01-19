package auth

import (
	"fmt"
)
import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

type ConsumerSignFilter struct {
}

func init() {
	extension.SetFilter(constant.CONSUMER_SIGN_FILTER, getConsumerSignFilter)
}

func (csf *ConsumerSignFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking ConsumerSign filter.")
	url := invoker.GetUrl()

	err := doAuthWork(&url, func(authenticator filter.Authenticator) error {
		return authenticator.Sign(invocation, &url)
	})
	if err != nil {
		panic(fmt.Sprintf("Sign for invocation %s # %s failed", url.ServiceKey(), invocation.MethodName()))

	}
	return invoker.Invoke(invocation)
}

func (csf *ConsumerSignFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}
func getConsumerSignFilter() filter.Filter {
	return &ConsumerSignFilter{}
}

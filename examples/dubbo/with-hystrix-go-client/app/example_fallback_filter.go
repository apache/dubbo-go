package main

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/filter/impl"
	"github.com/apache/dubbo-go/protocol"
	"time"
)

const (
	EXAMPLE_FALLBACK_FILTER = "example_fallback"
)

//This is an example filter that handles result from hystrix filter
//Define your filters and write your service downgrade strategy like this
//TODO: Maybe a base fallback filter can be offered
type ExampleFallbackFilter struct {
}

func GetExampleFallbackFilter() filter.Filter {
	return &ExampleFallbackFilter{}
}

//The name should be the same as in your config
//Put the filter in front of hystrix filter
func init() {
	extension.SetFilter(EXAMPLE_FALLBACK_FILTER, GetExampleFallbackFilter)
}

func (ff *ExampleFallbackFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return invoker.Invoke(invocation)

}
func (ff *ExampleFallbackFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if err := result.Error(); err != nil {
		hystrixError, ok := err.(*impl.HystrixFilterError)
		if ok {
			if hystrixError.FailByHystrix() {
				logger.Debugf("[Example fallback filter]%s get error caused by %s", invocation.MethodName(), hystrixError.Error())
				//Handle the error caused by Hystrix, including circuit breaking, concurrency limit and timeout
				//The detailed error source can be got through hystrixError.Error()
				//In this example we return a mock result under this circumstance
				res := User{
					"MockID",
					"MockName",
					55,
					time.Now(),
					Gender(MAN),
				}
				*(invocation.Reply().(*User)) = res
				result.SetResult(&res)
				result.SetError(nil)
			}
		}
	}
	//If the error is not caused by hystrix, the result doesn't come from hystrix filter or there's no error,
	//we just return it here.
	return result
}

package main

import (
	"github.com/apache/dubbo-go/filter/impl"
	"github.com/apache/dubbo-go/protocol"
)
import "github.com/afex/hystrix-go/hystrix"

const EXAMPLE_FALLBACK_NAME = "exampleFallback"


type ExampleHystrixFallback struct {
}

//Example for hystrix fallback function
//The 4 parameters:
//The error that caused the fallback;
//The invoker that the filter receives;
//The invocation that should be invoked;
//The copy of circuit breaker for this invocation, for getting its status
func(f *ExampleHystrixFallback) FallbackFunc (err error, invoker protocol.Invoker, invocation protocol.Invocation, cb hystrix.CircuitBreaker) protocol.Result {
	result := &protocol.RPCResult{}
	if cb.IsOpen() {
		result.SetError(nil)
		result.SetResult("Mock OK")
	} else {
		result.SetError(err)
	}
	return result
}


//Add the fallback function to the map
//The name MUST be the same as in your config file
func init() {
	impl.SetHystrixFallback(EXAMPLE_FALLBACK_NAME, &ExampleHystrixFallback{})
}

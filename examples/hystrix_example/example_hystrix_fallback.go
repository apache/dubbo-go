package hystrix_example

import (
	"github.com/apache/dubbo-go/filter/impl"
	"github.com/apache/dubbo-go/protocol"
)
import "github.com/afex/hystrix-go/hystrix"

const EXAMPLE_FALLBACK_NAME = "exampleFallback"

//Example for hystrix fallback function
//The 3 parameters:
//The error that caused the fallback;
//The result to return in the invoker;
//The copy of circuit breaker for this invocation, for getting its status
func ExampleHystrixFallback(err error, result protocol.Result, cb hystrix.CircuitBreaker) {
	if cb.IsOpen() {
		result.SetError(nil)
		result.SetResult("Mock OK")
	} else {
		result.SetError(err)
	}
}

//Add the fallback function to the map
//The name MUST be the same as in your config file
func init() {
	impl.SetHystrixFallback(EXAMPLE_FALLBACK_NAME, ExampleHystrixFallback)
}

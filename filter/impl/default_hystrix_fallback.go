package impl

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/apache/dubbo-go/protocol"
)

func defalutHystrixFallback(err error, invoker protocol.Invoker, invocation protocol.Invocation, cb hystrix.CircuitBreaker) protocol.Result {
	//By default, return nil value and the error occurred
	res := &protocol.RPCResult{}
	res.SetError(err)
	return res
}

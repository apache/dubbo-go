package impl

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/apache/dubbo-go/protocol"
)

func defalutHystrixFallback(err error, res protocol.Result, cb hystrix.CircuitBreaker) {
	//By default, return nil value and the error occurred
	res = &protocol.RPCResult{}
	res.SetError(err)
}

package impl

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/filter"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

const ECHO = "echo"

func init() {
	extension.SetFilter(ECHO, GetFilter)
}

// RPCService need a Echo method in consumer, if you want to use EchoFilter
// eg:
//		Echo func(ctx context.Context, args []interface{}, rsp *Xxx) error
type EchoFilter struct {
}

func (ef *EchoFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	log.Info("invoking echo filter.")
	log.Debug("%v,%v", invocation.MethodName(), len(invocation.Arguments()))
	if invocation.MethodName() == constant.ECHO && len(invocation.Arguments()) == 1 {
		return &protocol.RPCResult{
			Rest: invocation.Arguments()[0],
		}
	}
	return invoker.Invoke(invocation)
}

func (ef *EchoFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func GetFilter() filter.Filter {
	return &EchoFilter{}
}

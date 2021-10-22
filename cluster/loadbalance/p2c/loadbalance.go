package p2c

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type loadBalance struct {
	
}

func (l loadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	panic("implement me")
}


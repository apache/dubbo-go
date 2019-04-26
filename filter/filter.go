package filter

import (
	"github.com/dubbo/dubbo-go/protocol"
)

// Extension - Protocol
type Filter interface {
	Invoke(protocol.Invoker, protocol.Invocation) protocol.Result
	OnResponse(protocol.Result, protocol.Invoker, protocol.Invocation) protocol.Result
}

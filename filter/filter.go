package filter

import (
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

// Extension - Filter
type Filter interface {
	Invoke(protocol.Invoker, protocol.Invocation) protocol.Result
	OnResponse(protocol.Result, protocol.Invoker, protocol.Invocation) protocol.Result
}

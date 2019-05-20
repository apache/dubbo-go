package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

// Extension - LoadBalance
type LoadBalance interface {
	Select([]protocol.Invoker, protocol.Invocation) protocol.Invoker
}

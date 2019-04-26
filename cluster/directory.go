package cluster

import (
	"github.com/dubbo/dubbo-go/common"
	"github.com/dubbo/dubbo-go/protocol"
)

// Extension - Directory
type Directory interface {
	common.Node
	List(invocation protocol.Invocation) []protocol.Invoker
}

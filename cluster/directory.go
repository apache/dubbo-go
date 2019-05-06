package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

// Extension - Directory
type Directory interface {
	common.Node
	List(invocation protocol.Invocation) []protocol.Invoker
}

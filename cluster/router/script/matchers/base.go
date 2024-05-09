package matchers

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type ScriptInstance interface {
	Route([]protocol.Invoker, *common.URL, protocol.Invocation) []protocol.Invoker
}

type ScriptManager interface {
	GetInstance(scriptHash string) (ScriptInstance, error)
}

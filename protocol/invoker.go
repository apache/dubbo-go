package protocol

import "github.com/dubbo/dubbo-go/common"

// Extension - Invoker
type Invoker interface {
	common.Node
	Invoke(Invocation) Result
}

package script

import "dubbo.apache.org/dubbo-go/v3/protocol"

type ScriptMatcher interface {
	DoMatch([]protocol.Invoker, protocol.Invocation) []protocol.Invoker
}

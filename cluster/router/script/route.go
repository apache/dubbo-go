package script

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"sync"
)

type ScriptRouter struct {
	mu      sync.RWMutex
	key     string
	enabled bool
}

func NewScriptPriorityRouter() *ScriptRouter {
	return nil
}
func (s ScriptRouter) Process(event *config_center.ConfigChangeEvent) {
	//TODO implement me
	panic("implement me")
}

func (s ScriptRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	//TODO implement me
	panic("implement me")
}

func (s ScriptRouter) URL() *common.URL {
	return nil
}

func (s ScriptRouter) Priority() int64 {
	return 0
}

func (s ScriptRouter) Notify(invokers []protocol.Invoker) {
	//TODO implement me
	panic("implement me")
}

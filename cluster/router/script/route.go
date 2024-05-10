package script

import (
	ins "dubbo.apache.org/dubbo-go/v3/cluster/router/script/instance"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
	"strings"
	"sync"
)

type ScriptRouter struct {
	mu         sync.RWMutex
	scriptType string
	key        string // key to application - name
	enabled    bool   // enabled
	rawScript  string
}

func NewScriptRouter() *ScriptRouter {
	applicationName := config.GetApplicationConfig().Name
	a := &ScriptRouter{
		key:     applicationName,
		enabled: false,
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration != nil {
		dynamicConfiguration.AddListener(strings.Join([]string{applicationName, constant.ScriptRouterRuleSuffix}, ""), a)
	}
	return a
}
func (s *ScriptRouter) Process(event *config_center.ConfigChangeEvent) {
	//TODO implement me
	panic("implement me")
}

func (s *ScriptRouter) runScript(scriptType, rawScript string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error) {
	in, err := ins.GetInstance(scriptType)
	if err != nil {
		return nil, err
	}
	return in.RunScript(rawScript, invokers, invocation)
}

func (s *ScriptRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if invokers == nil || len(invokers) == 0 {
		return []protocol.Invoker{}
	}
	if s.enabled == false {
		return invokers
	}
	res, err := s.runScript(s.scriptType, s.rawScript, invokers, invocation)
	if err != nil {
		logger.Warnf("ScriptRouter.Route error: %v", err)
		return []protocol.Invoker{}
	}
	return res
}

func (s *ScriptRouter) URL() *common.URL {
	return nil
}

func (s *ScriptRouter) Priority() int64 {
	return 0
}

func (s *ScriptRouter) Notify(invokers []protocol.Invoker) {
	//TODO implement me
	panic("implement me")
}

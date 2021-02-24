package uniform

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
)

type DubboRouterRule struct {
	uniformRules []*UniformRule
}

func newDubboRouterRule(dubboRoutes []*config.DubboRoute, destinationMap map[string]map[string]string) (*DubboRouterRule, error) {
	uniformRules := make([]*UniformRule, 0)
	for _, v := range dubboRoutes {
		uniformRule, err := newUniformRule(v, destinationMap)
		if err != nil {
			return nil, err
		}
		uniformRules = append(uniformRules, uniformRule)
	}

	return &DubboRouterRule{
		uniformRules: uniformRules,
	}, nil
}

func (drr *DubboRouterRule) route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	resultInvokers := make([]protocol.Invoker, 0)
	for _, v := range drr.uniformRules {
		if resultInvokers = v.route(invokers, url, invocation); len(resultInvokers) == 0 {
			continue
		}
		return resultInvokers
	}
	return resultInvokers
}

package uniform

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/protocol"
	"gopkg.in/yaml.v2"
	"io"
	"strings"
)

type RouterChain struct {
	routers []*UniformRouter
	notify  chan struct{}
}

func NewUniformRouterChain(virtualServiceConfig, destinationRuleConfig []byte, notify chan struct{}) (router.PriorityRouter, error) {
	uniformRouters, err := parseFromConfigToRouters(virtualServiceConfig, destinationRuleConfig, notify)
	if err != nil {
		return nil, err
	}
	r := &RouterChain{
		routers: uniformRouters,
		notify:  notify,
	}
	return r, nil
}

func (r *RouterChain) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	// todo is this none copy cause problem?
	for _, v := range r.routers {
		invokers = v.Route(invokers, url, invocation)
	}
	return invokers
}

func (r *RouterChain) Process(event *config_center.ConfigChangeEvent) {
	// todo deal with router change

}

// Pool separates healthy invokers from others.
func (r *RouterChain) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	rb := make(router.AddrPool, 8)
	//rb[uniformSelected] = roaring.NewBitmap()
	//for i, invoker := range invokers {
	//	if r.checker.IsConnHealthy(invoker) {
	//		rb[connHealthy].Add(uint32(i))
	//	}
	//}
	return rb, nil
}

// ShouldPool will always return true to make sure healthy check constantly.
func (r *RouterChain) ShouldPool() bool {
	return true
}

// Name get name of ConnCheckerRouter
func (r *RouterChain) Name() string {
	return name
}

// Priority get Router priority level
func (r *RouterChain) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *RouterChain) URL() *common.URL {
	return nil
}

func parseFromConfigToRouters(virtualServiceConfig, destinationRuleConfig []byte, notify chan struct{}) ([]*UniformRouter, error) {
	// todo parse config byte
	var virtualServiceConfigList []*config.VirtualServiceConfig
	destRuleConfigsMap := make(map[string]map[string]map[string]string)

	vsDecoder := yaml.NewDecoder(strings.NewReader(string(virtualServiceConfig)))
	drDecoder := yaml.NewDecoder(strings.NewReader(string(destinationRuleConfig)))
	for {
		virtualServiceCfg := &config.VirtualServiceConfig{}
		err := vsDecoder.Decode(virtualServiceCfg)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("parseFromConfigTo virtual service err = ", err)
			return nil, err
		}
		virtualServiceConfigList = append(virtualServiceConfigList, virtualServiceCfg)
	}

	for {
		destRuleCfg := &config.DestinationRuleConfig{}
		err := drDecoder.Decode(destRuleCfg)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("parseFromConfigTo destination rule err = ", err)
			return nil, err
		}
		destRuleCfgMap := make(map[string]map[string]string)
		for _, v := range destRuleCfg.Spec.SubSets {
			destRuleCfgMap[v.Name] = v.Labels
		}
		destRuleConfigsMap[destRuleCfg.Spec.Host] = destRuleCfgMap
	}

	routers := make([]*UniformRouter, 0)

	for _, v := range virtualServiceConfigList {
		tempSerivceNeedsDescMap := make(map[string]map[string]string)
		for _, host := range v.Spec.Hosts {
			targetDestMap := destRuleConfigsMap[host]

			// copy to new Map
			mapCombine(tempSerivceNeedsDescMap, targetDestMap)
		}

		newRule, err := newDubboRouterRule(v.Spec.Dubbo, tempSerivceNeedsDescMap)
		if err != nil {
			logger.Error("Parse config to uniform rule err = ", err)
			return nil, err
		}
		rtr, err := NewUniformRouter(newRule, notify)
		if err != nil {
			logger.Error("new uniform router err = ", err)
			return nil, err
		}
		routers = append(routers, rtr)
	}
	return routers, nil
}

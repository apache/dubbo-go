package uniform

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/uniform/k8s_api"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/remoting"
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"strings"
)

type RouterChain struct {
	routers                    []*UniformRouter
	virtualServiceConfigBytes  []byte
	destinationRuleConfigBytes []byte
	notify                     chan struct{}
}

func NewUniformRouterChain(virtualServiceConfig, destinationRuleConfig []byte, notify chan struct{}) (router.PriorityRouter, error) {
	uniformRouters := make([]*UniformRouter, 0)
	var err error
	fromFileConfig := true
	uniformRouters, err = parseFromConfigToRouters(virtualServiceConfig, destinationRuleConfig, notify)
	if err != nil {
		fromFileConfig = false
		logger.Info("parse router config form local file failed")
	}
	r := &RouterChain{
		virtualServiceConfigBytes:  virtualServiceConfig,
		destinationRuleConfigBytes: destinationRuleConfig,
		routers:                    uniformRouters,
		notify:                     notify,
	}
	if err := k8s_api.SetK8sEventListener(r); err != nil {
		logger.Info("try listen K8s router config failed")
		if !fromFileConfig {
			return nil, perrors.New("No config file from both local file and k8s")
		}
	}
	return r, nil
}

func (r *RouterChain) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	for _, v := range r.routers {
		invokers = v.Route(invokers, url, invocation)
	}
	return invokers
}

func (r *RouterChain) Process(event *config_center.ConfigChangeEvent) {
	fmt.Printf("on processed event = %+v\n", *event)
	if event.ConfigType == remoting.EventTypeAdd || event.ConfigType == remoting.EventTypeUpdate {
		fmt.Println("event type add or update ")
		switch event.Key {
		case k8s_api.VirtualServiceEventKey:
			fmt.Println("virtul service event key")
			newVSValue, ok := event.Value.(*config.VirtualServiceConfig)
			if !ok {
				logger.Error("event.Value assertion error")
				return
			}

			newVSJsonValue, ok := newVSValue.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			if !ok {
				logger.Error("newVSValue.ObjectMeta.Annotations has no key named kubectl.kubernetes.io/last-applied-configuration")
				return
			}
			fmt.Println("json file = ", newVSJsonValue)
			newVirtualServiceConfig := &config.VirtualServiceConfig{}
			if err := json.Unmarshal([]byte(newVSJsonValue), newVirtualServiceConfig); err != nil {
				logger.Error("on process json data unmarshal error = ", err)
				return
			}
			newVirtualServiceConfig.YamlAPIVersion = newVirtualServiceConfig.APIVersion
			newVirtualServiceConfig.YamlKind = newVirtualServiceConfig.Kind
			newVirtualServiceConfig.MetaData.Name = newVirtualServiceConfig.ObjectMeta.Name
			fmt.Printf("get event after asseration = %+v\n", newVirtualServiceConfig)
			data, err := yaml.Marshal(newVirtualServiceConfig)
			if err != nil {
				logger.Error("Process change of virtual service: event.Value marshal error:", err)
				return
			}
			r.routers, err = parseFromConfigToRouters(data, r.destinationRuleConfigBytes, r.notify)
			if err != nil {
				logger.Error("Process change of virtual service: parseFromConfigToRouters:", err)
				return
			}
		case k8s_api.DestinationRuleEventKey:
			fmt.Println("dest rule event key")
			newDRValue, ok := event.Value.(*config.DestinationRuleConfig)
			if !ok {
				logger.Error("event.Value assertion error")
				return
			}

			newDRJsonValue, ok := newDRValue.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			if !ok {
				logger.Error("newVSValue.ObjectMeta.Annotations has no key named kubectl.kubernetes.io/last-applied-configuration")
				return
			}
			fmt.Println("json file = ", newDRJsonValue)
			newDestRuleConfig := &config.DestinationRuleConfig{}
			if err := json.Unmarshal([]byte(newDRJsonValue), newDestRuleConfig); err != nil {
				logger.Error("on process json data unmarshal error = ", err)
				return
			}
			newDestRuleConfig.YamlAPIVersion = newDestRuleConfig.APIVersion
			newDestRuleConfig.YamlKind = newDestRuleConfig.Kind
			newDestRuleConfig.MetaData.Name = newDestRuleConfig.ObjectMeta.Name
			fmt.Printf("get event after asseration = %+v\n", newDestRuleConfig)
			data, err := yaml.Marshal(newDestRuleConfig)
			if err != nil {
				logger.Error("Process change of dest rule: event.Value marshal error:", err)
				return
			}
			r.routers, err = parseFromConfigToRouters(r.virtualServiceConfigBytes, data, r.notify)
			if err != nil {
				logger.Error("Process change of dest rule: parseFromConfigToRouters:", err)
				return
			}
		default:
			logger.Error("unknow unsupported event key:", event.Key)
		}
	}

	if event.ConfigType == remoting.EventTypeDel {

	}
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

// parseFromConfigToRouters file -> routers
func parseFromConfigToRouters(virtualServiceConfig, destinationRuleConfig []byte, notify chan struct{}) ([]*UniformRouter, error) {
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
		data, err := json.Marshal(v.Spec.Dubbo)
		if err != nil {
			logger.Error("v.Spec.Dubbo unmarshal error = ", err)
		}
		fmt.Printf("===v.Spec.Dubbo = %s\n", string(data))
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
	logger.Debug("parsed successed! with router size = ", len(routers))
	return routers, nil
}

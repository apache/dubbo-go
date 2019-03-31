package plugins

import (
	"github.com/dubbo/dubbo-go/client/selector"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
)

var PluggableRegistries = map[string]func(...registry.RegistryOption) (registry.Registry, error){
	"zookeeper": zookeeper.NewZkRegistry,
}

var PluggableLoadbalance = map[string]func() selector.Selector{
	"round_robin": selector.NewRoundRobinSelector,
	"random":      selector.NewRandomSelector,
}

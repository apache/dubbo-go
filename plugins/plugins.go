package plugins

import (
	"github.com/dubbo/dubbo-go/client/loadBalance"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
)

var PluggableRegistries = map[string]func(...registry.OptionInf) (registry.Registry,error){
	"zookeeper":zookeeper.NewZkRegistry,
}

var PluggableLoadbalance = map[string]func()loadBalance.Selector{
	"round_robin":loadBalance.NewRoundRobinSelector,
	"random":loadBalance.NewRandomSelector,
}

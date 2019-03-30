package plugins

import (
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
)

var PluggableRegistries = map[string]func(...registry.OptionInf) (registry.Registry,error){
	"zookeeper":zookeeper.NewZkRegistry,
}


package plugins

import (
	"github.com/dubbo/dubbo-go/client/selector"
	"github.com/dubbo/dubbo-go/registry"
)

var PluggableRegistries = map[string]func(...registry.RegistryOption) (registry.Registry, error){}

var PluggableLoadbalance = map[string]func() selector.Selector{
	"round_robin": selector.NewRoundRobinSelector,
	"random":      selector.NewRandomSelector,
}

// service configuration plugins , related to SeviceConfig for consumer paramters / ProviderSeviceConfig for provider parameters /

// TODO:ServiceEven & ServiceURL subscribed by consumer from provider's listener shoud abstract to interface
var PluggableServiceConfig = map[string]func() registry.ServiceConfig{
	"default": registry.NewDefaultServiceConfig,
}
var PluggableProviderServiceConfig = map[string]func() registry.ProviderServiceConfig{
	"default": registry.NewDefaultProviderServiceConfig,
}

var PluggableServiceURL = map[string]func(string) (registry.ServiceURL, error){
	"default": registry.NewDefaultServiceURL,
}

var defaultServiceConfig = registry.NewDefaultServiceConfig
var defaultProviderServiceConfig = registry.NewDefaultProviderServiceConfig

var defaultServiceURL = registry.NewDefaultServiceURL

func SetDefaultServiceConfig(s string) {
	defaultServiceConfig = PluggableServiceConfig[s]
}
func DefaultServiceConfig() func() registry.ServiceConfig {
	return defaultServiceConfig
}

func SetDefaultProviderServiceConfig(s string) {
	defaultProviderServiceConfig = PluggableProviderServiceConfig[s]
}
func DefaultProviderServiceConfig() func() registry.ProviderServiceConfig {
	return defaultProviderServiceConfig
}

func SetDefaultServiceURL(s string) {
	defaultServiceURL = PluggableServiceURL[s]
}
func DefaultServiceURL() func(string) (registry.ServiceURL, error) {
	return defaultServiceURL
}

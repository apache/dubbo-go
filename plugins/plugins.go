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

// TODO:ServiceEven & ConfigURL subscribed by consumer from provider's listener shoud abstract to interface
var PluggableServiceConfig = map[string]func() registry.ReferenceConfig{
	"default": registry.NewServiceConfig,
}
var PluggableProviderServiceConfig = map[string]func() registry.ProviderServiceConfig{
	"default": registry.NewDefaultProviderServiceConfig,
}

var PluggableServiceURL = map[string]func(string) (config.ConfigURL, error){
	"default": registry.NewDefaultServiceURL,
}

var defaultServiceConfig = registry.NewServiceConfig
var defaultProviderServiceConfig = registry.NewDefaultProviderServiceConfig

var defaultServiceURL = registry.NewDefaultServiceURL

func SetDefaultServiceConfig(s string) {
	defaultServiceConfig = PluggableServiceConfig[s]
}
func DefaultServiceConfig() func() registry.ReferenceConfig {
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
func DefaultServiceURL() func(string) (config.ConfigURL, error) {
	return defaultServiceURL
}

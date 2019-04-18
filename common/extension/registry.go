package extension

import (
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/registry"
)

var (
	registrys       map[string]func(config.RegisterConfig) registry.Registry
	registryConfigs map[string]func(string, map[string]string) config.RegisterConfig
)

/*
it must excute first
*/
func init() {
	// init map
	registrys = make(map[string]func(config.RegisterConfig) registry.Registry)
	registryConfigs = make(map[string]func(string, map[string]string) config.RegisterConfig)
}

func SetRegistry(name string, v func(config.RegisterConfig) registry.Registry) {
	registrys[name] = v
}

func SetRegistryConfig(name string, v func(string, map[string]string) config.RegisterConfig) {
	registryConfigs[name] = v
}

func GetRegistryExtension(name string, config config.RegisterConfig) registry.Registry {
	return registrys[name](config)
}

func GetRegistryConfigExtension(name string, src map[string]string) config.RegisterConfig {
	return registryConfigs[name](name, src)
}

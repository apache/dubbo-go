package extension

import (
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/registry"
)

var (
	registrys map[string]func(config *config.RegistryURL) (registry.Registry, error)
)

/*
it must excute first
*/
func init() {
	// init map
	registrys = make(map[string]func(config *config.RegistryURL) (registry.Registry, error))
}

func SetRegistry(name string, v func(config *config.RegistryURL) (registry.Registry, error)) {
	registrys[name] = v
}

func GetRegistryExtension(name string, config *config.RegistryURL) (registry.Registry, error) {
	return registrys[name](config)

}

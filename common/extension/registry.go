package extension

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

var (
	registrys map[string]func(config *common.URL) (registry.Registry, error)
)

/*
it must excute first
*/
func init() {
	// init map
	registrys = make(map[string]func(config *common.URL) (registry.Registry, error))
}

func SetRegistry(name string, v func(config *common.URL) (registry.Registry, error)) {
	registrys[name] = v
}

func GetRegistryExtension(name string, config *common.URL) (registry.Registry, error) {
	return registrys[name](config)

}

package extension

import (
	"github.com/dubbo/dubbo-go/registry"
)

var (
	registrys       map[string]func(...registry.RegistryOption) registry.Registry
)

/*
it must excute first
*/
func init() {
	// init map
	registrys = make(map[string]func(...registry.RegistryOption) registry.Registry)
}

func SetRegistry(name string, v func(...registry.RegistryOption) registry.Registry) {
	registrys[name] = v
}



func GetRegistryExtension(name string, options registry.RegistryOption) registry.Registry {
	return registrys[name](options)
}

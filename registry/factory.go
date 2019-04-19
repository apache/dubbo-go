package registry

import "github.com/dubbo/dubbo-go/config"

type RegistryFactory interface {
	GetRegistry(url config.ConfigURL) Registry
}

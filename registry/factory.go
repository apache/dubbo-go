package registry

import "github.com/dubbo/go-for-apache-dubbo/config"

type RegistryFactory interface {
	GetRegistry(url config.URL) Registry
}

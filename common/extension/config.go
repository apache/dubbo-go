package extension

import (
	"github.com/dubbo/dubbo-go/config"
)

var (
	serviceConfig map[string]func() config.ServiceConfig
	url           map[string]func(string) config.URL
)

func init() {
	// init map
	serviceConfig = make(map[string]func() config.ServiceConfig)
	url = make(map[string]func(string) config.URL)
}

func SetServiceConfig(name string, v func() config.ServiceConfig) {
	serviceConfig[name] = v
}

func SetURL(name string, v func(string) config.URL) {
	url[name] = v
}

func GetServiceConfigExtension(name string) config.ServiceConfig {
	if name == "" {
		name = "default"
	}
	return serviceConfig[name]()
}

func GetDefaultServiceConfigExtension() config.ServiceConfig {
	return serviceConfig["default"]()
}

func GetURLExtension(name string, urlString string) config.URL {
	if name == "" {
		name = "default"
	}
	return url[name](urlString)
}
func GetDefaultURLExtension(urlString string) config.URL {
	return url["default"](urlString)
}

package support

import (
	"context"
	"github.com/dubbo/dubbo-go/common/constant"
	"net/url"
)
import (
	log "github.com/AlexStocks/log4go"
)
import "github.com/dubbo/dubbo-go/config"

type RegistryConfig struct {
	Id         string `required:"true" yaml:"id"  json:"id,omitempty"`
	TimeoutStr string `yaml:"timeout" default:"5s" json:"timeout,omitempty"` // unit: second
	Type       string `required:"true" yaml:"type"  json:"type,omitempty"`
	Group      string `yaml:"group" json:"group,omitempty"`
	//for registry
	Address string `yaml:"address" json:"address,omitempty"`
}

func loadRegistries(registriesIds []ConfigRegistry, registries []RegistryConfig, roleType config.RoleType) []*config.URL {
	var urls []*config.URL
	for _, registry := range registriesIds {
		for _, registryConf := range registries {
			if string(registry) == registryConf.Id {
				url, err := config.NewURL(context.TODO(), registryConf.Address, config.WithParams(registryConf.getUrlMap(roleType)), config.WithProtocol(constant.REGISTRY_PROTOCOL))
				if err != nil {
					log.Error("The registry id:%s url is invalid ,and will skip the registry, error: %#v", registryConf.Id, err)
				} else {
					urls = append(urls, &url)
				}

			}
		}

	}
	return urls
}

func (regconfig *RegistryConfig) getUrlMap(roleType config.RoleType) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, regconfig.Group)
	urlMap.Set(constant.ROLE_KEY, roleType.String())
	urlMap.Set(constant.REGISTRY_KEY, regconfig.Type)
	urlMap.Set(constant.REGISTRY_TIMEOUT_KEY, regconfig.TimeoutStr)
	return urlMap
}

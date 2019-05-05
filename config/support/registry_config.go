package support

import (
	"context"
	"github.com/dubbo/dubbo-go/common/constant"
	"net/url"
	"strconv"
)
import (
	log "github.com/AlexStocks/log4go"
)
import "github.com/dubbo/dubbo-go/config"

type RegistryConfig struct {
	Id         string `required:"true" yaml:"id"  json:"id,omitempty"`
	Type       string `required:"true" yaml:"type"  json:"type,omitempty"`
	TimeoutStr string `yaml:"timeout" default:"5s" json:"timeout,omitempty"` // unit: second
	Group      string `yaml:"group" json:"group,omitempty"`
	//for registry
	Address  string `yaml:"address" json:"address,omitempty"`
	Username string `yaml:"username" json:"address,omitempty"`
	Password string `yaml:"password" json:"address,omitempty"`
}

func loadRegistries(registriesIds []ConfigRegistry, registries []RegistryConfig, roleType config.RoleType) []*config.URL {
	var urls []*config.URL
	for _, registry := range registriesIds {
		for _, registryConf := range registries {
			if string(registry) == registryConf.Id {

				url, err := config.NewURL(context.TODO(), constant.REGISTRY_PROTOCOL+"://"+registryConf.Address, config.WithParams(registryConf.getUrlMap(roleType)),
					config.WithUsername(registryConf.Username), config.WithPassword(registryConf.Password),
				)

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
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(int(roleType)))
	urlMap.Set(constant.REGISTRY_KEY, regconfig.Type)
	urlMap.Set(constant.REGISTRY_TIMEOUT_KEY, regconfig.TimeoutStr)
	return urlMap
}

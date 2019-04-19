package config

import (
	log "github.com/AlexStocks/log4go"
)

import "github.com/dubbo/dubbo-go/common/extension"

var refprotocol = extension.GetRefProtocol()

type ReferenceConfig struct {
	Interface  string                    `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries []referenceConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	URLs       []ConfigURL               `yaml:"-"`
}
type referenceConfigRegistry struct {
	string
}

func NewReferenceConfig() *ReferenceConfig {
	return &ReferenceConfig{}
}

func (refconfig *ReferenceConfig) CreateProxy() {
	//首先是user specified URL, could be peer-to-peer address, or register center's address.

	//其次是assemble URL from register center's configuration模式
	urls := refconfig.loadRegistries()

	if len(urls) == 1 {
		refprotocol.Export()
	} else {

	}
}

func (refconfig *ReferenceConfig) loadRegistries() []ConfigURL {
	var urls []ConfigURL
	for _, registry := range refconfig.Registries {
		for _, registryConf := range consumerConfig.Registries {
			if registry.string == registryConf.Id {
				url, err := NewConfigURL(registryConf.Address)
				if err != nil {
					log.Error("The registry id:%s url is invalid ,and will skip the registry", registryConf.Id)
				} else {
					urls = append(urls, *url)
				}

			}
		}

	}
	return urls
}

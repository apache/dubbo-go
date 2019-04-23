package config

import (
	"context"
	log "github.com/AlexStocks/log4go"
)

import "github.com/dubbo/dubbo-go/common/extension"

var refprotocol = extension.GetRefProtocol()

type ReferenceConfig struct {
	context    context.Context
	Interface  string                    `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries []referenceConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster    string                    `default:"failover" yaml:"cluster"  json:"cluster,omitempty"`
	Methods    []method                  `yaml:"methods"  json:"methods,omitempty"`
	URLs       []URL                     `yaml:"-"`
}
type referenceConfigRegistry struct {
	string
}

type method struct {
	name    string `yaml:"name"  json:"name,omitempty"`
	retries int    `yaml:"retries"  json:"retries,omitempty"`
}

func NewReferenceConfig(ctx context.Context) *ReferenceConfig {
	return &ReferenceConfig{context: ctx}
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

func (refconfig *ReferenceConfig) loadRegistries() []URL {
	var urls []URL
	for _, registry := range refconfig.Registries {
		for _, registryConf := range consumerConfig.Registries {
			if registry.string == registryConf.Id {
				url, err := NewURL(registryConf.Address)
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

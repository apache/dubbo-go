// Copyright 2016-2019 hxmhlt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"net/url"
	"strconv"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
)

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

func loadRegistries(registriesIds []ConfigRegistry, registries []RegistryConfig, roleType common.RoleType) []*common.URL {
	var urls []*common.URL
	for _, registry := range registriesIds {
		for _, registryConf := range registries {
			if string(registry) == registryConf.Id {

				url, err := common.NewURL(context.TODO(), constant.REGISTRY_PROTOCOL+"://"+registryConf.Address, common.WithParams(registryConf.getUrlMap(roleType)),
					common.WithUsername(registryConf.Username), common.WithPassword(registryConf.Password),
				)

				if err != nil {
					logger.Errorf("The registry id:%s url is invalid ,and will skip the registry, error: %#v", registryConf.Id, err)
				} else {
					urls = append(urls, &url)
				}

			}
		}

	}
	return urls
}

func (regconfig *RegistryConfig) getUrlMap(roleType common.RoleType) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, regconfig.Group)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(int(roleType)))
	urlMap.Set(constant.REGISTRY_KEY, regconfig.Type)
	urlMap.Set(constant.REGISTRY_TIMEOUT_KEY, regconfig.TimeoutStr)
	return urlMap
}

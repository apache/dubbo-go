/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"context"
	"net/url"
	"strconv"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

type RegistryConfig struct {
	Protocol string `required:"true" yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	//I changed "type" to "protocol" ,the same as "protocol" field in java class RegistryConfig
	TimeoutStr string `yaml:"timeout" default:"5s" json:"timeout,omitempty" property:"timeout"` // unit: second
	Group      string `yaml:"group" json:"group,omitempty" property:"group"`
	//for registry
	Address  string `yaml:"address" json:"address,omitempty" property:"address"`
	Username string `yaml:"username" json:"address,omitempty" property:"username"`
	Password string `yaml:"password" json:"address,omitempty"  property:"password"`
}

func (*RegistryConfig) Prefix() string {
	return constant.RegistryConfigPrefix
}

func loadRegistries(registries map[string]*RegistryConfig, roleType common.RoleType) []*common.URL {
	var urls []*common.URL
	for k, registryConf := range registries {

		url, err := common.NewURL(
			context.TODO(),
			constant.REGISTRY_PROTOCOL+"://"+registryConf.Address,
			common.WithParams(registryConf.getUrlMap(roleType)),
			common.WithUsername(registryConf.Username),
			common.WithPassword(registryConf.Password),
		)

		if err != nil {
			logger.Errorf("The registry id:%s url is invalid ,and will skip the registry, error: %#v", k, err)
		} else {
			urls = append(urls, &url)
		}

	}

	return urls
}

func (regconfig *RegistryConfig) getUrlMap(roleType common.RoleType) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, regconfig.Group)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(int(roleType)))
	urlMap.Set(constant.REGISTRY_KEY, regconfig.Protocol)
	urlMap.Set(constant.REGISTRY_TIMEOUT_KEY, regconfig.TimeoutStr)

	return urlMap
}

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
	"strings"
)

import (
	"github.com/creasty/defaults"
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
	Address  string            `yaml:"address" json:"address,omitempty" property:"address"`
	Username string            `yaml:"username" json:"username,omitempty" property:"username"`
	Password string            `yaml:"password" json:"password,omitempty"  property:"password"`
	Params   map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
}

func (c *RegistryConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain RegistryConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

func (*RegistryConfig) Prefix() string {
	return constant.RegistryConfigPrefix + "|" + constant.SingleRegistryConfigPrefix
}

func loadRegistries(targetRegistries string, registries map[string]*RegistryConfig, roleType common.RoleType) []*common.URL {
	var urls []*common.URL
	trSlice := strings.Split(targetRegistries, ",")

	for k, registryConf := range registries {
		target := false

		// if user not config targetRegistries,default load all
		// Notice:in func "func Split(s, sep string) []string"  comment : if s does not contain sep and sep is not empty, SplitAfter returns a slice of length 1 whose only element is s.
		// So we have to add the condition when targetRegistries string is not set (it will be "" when not set)
		if len(trSlice) == 0 || (len(trSlice) == 1 && trSlice[0] == "") {
			target = true
		} else {
			// else if user config targetRegistries
			for _, tr := range trSlice {
				if tr == k {
					target = true
					break
				}
			}
		}

		if target {
			var (
				url common.URL
				err error
			)

			addresses := strings.Split(registryConf.Address, ",")
			address := addresses[0]
			address = traslateRegistryConf(address, registryConf)
			url, err = common.NewURL(
				context.Background(),
				constant.REGISTRY_PROTOCOL+"://"+address,
				common.WithParams(registryConf.getUrlMap(roleType)),
				common.WithUsername(registryConf.Username),
				common.WithPassword(registryConf.Password),
				common.WithLocation(registryConf.Address),
			)

			if err != nil {
				logger.Errorf("The registry id:%s url is invalid , error: %#v", k, err)
				panic(err)
			} else {
				urls = append(urls, &url)
			}
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
	for k, v := range regconfig.Params {
		urlMap.Set(k, v)
	}
	return urlMap
}

func traslateRegistryConf(address string, registryConf *RegistryConfig) string {
	if strings.Contains(address, "://") {
		translatedUrl, err := url.Parse(address)
		if err != nil {
			logger.Errorf("The registry  url is invalid , error: %#v", err)
			panic(err)
		}
		address = translatedUrl.Host
		registryConf.Protocol = translatedUrl.Scheme
		registryConf.Address = strings.Replace(registryConf.Address, translatedUrl.Scheme+"://", "", -1)
	}
	return address
}

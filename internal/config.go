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

// Package internal contains dubbo-go-internal code, to avoid polluting
// the top-level dubbo-go package.  It must not import any dubbo-go symbols
// except internal symbols to avoid circular dependencies.
package internal

import (
	"net/url"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

func LoadRegistries(registryIds []string, registries map[string]*global.RegistryConfig, roleType common.RoleType) []*common.URL {
	var registryURLs []*common.URL

	for k, registryConf := range registries {
		target := false

		if len(registryIds) == 0 || (len(registryIds) == 1 && registryIds[0] == "") {
			target = true
		} else {
			for _, tr := range registryIds {
				if tr == k {
					target = true
					break
				}
			}
		}

		if target {
			urls, err := toURLs(registryConf, roleType)
			if err != nil {
				logger.Errorf("The registry id: %s url is invalid, error: %#v", k, err)
				continue
			}

			for _, u := range urls {
				if u == nil {
					continue
				}

				clonedURL := u.Clone()
				clonedURL.AddParam(constant.RegistryIdKey, k)
				registryURLs = append(registryURLs, clonedURL)
			}
		}
	}

	return registryURLs
}

func toURLs(registriesConfig *global.RegistryConfig, roleType common.RoleType) ([]*common.URL, error) {
	address, err := translateRegistryAddress(registriesConfig)
	if err != nil {
		return nil, err
	}
	var urls []*common.URL
	var registryURL *common.URL

	if address == "" || address == constant.NotAvailable {
		logger.Infof("Empty or N/A registry address found, the process will work with no registry enabled " +
			"which means that the address of this instance will not be registered and not able to be found by other consumer instances.")
		return urls, nil
	}
	switch registriesConfig.RegistryType {
	case constant.RegistryTypeService:
		// service discovery protocol
		if registryURL, err = createNewURL(registriesConfig, constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	case constant.RegistryTypeInterface:
		if registryURL, err = createNewURL(registriesConfig, constant.RegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	case constant.RegistryTypeAll:
		if registryURL, err = createNewURL(registriesConfig, constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
		if registryURL, err = createNewURL(registriesConfig, constant.RegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	default:
		if registryURL, err = createNewURL(registriesConfig, constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	}
	return urls, err
}

func createNewURL(registriesConfig *global.RegistryConfig, protocol string, address string, roleType common.RoleType) (*common.URL, error) {
	return common.NewURL(protocol+"://"+address,
		common.WithParams(getUrlMap(registriesConfig, roleType)),
		common.WithParamsValue(constant.RegistrySimplifiedKey, strconv.FormatBool(registriesConfig.Simplified)),
		common.WithParamsValue(constant.RegistryKey, registriesConfig.Protocol),
		common.WithParamsValue(constant.RegistryNamespaceKey, registriesConfig.Namespace),
		common.WithParamsValue(constant.RegistryTimeoutKey, registriesConfig.Timeout),
		common.WithUsername(registriesConfig.Username),
		common.WithPassword(registriesConfig.Password),
		common.WithLocation(registriesConfig.Address),
	)
}

func translateRegistryAddress(registriesConfig *global.RegistryConfig) (string, error) {
	addr := registriesConfig.Address
	if strings.Contains(addr, "://") {
		u, err := url.Parse(addr)
		if err != nil {
			return "", err
		}
		return u.Host + u.Path, nil
	}
	return addr, nil
}

func getUrlMap(registriesConfig *global.RegistryConfig, roleType common.RoleType) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.RegistryGroupKey, registriesConfig.Group)
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(int(roleType)))
	urlMap.Set(constant.RegistryKey, registriesConfig.Protocol)
	urlMap.Set(constant.RegistryTimeoutKey, registriesConfig.Timeout)
	urlMap.Set(constant.RegistryKey+"."+constant.RegistryLabelKey, strconv.FormatBool(true))
	urlMap.Set(constant.RegistryKey+"."+constant.PreferredKey, strconv.FormatBool(registriesConfig.Preferred))
	urlMap.Set(constant.RegistryKey+"."+constant.RegistryZoneKey, registriesConfig.Zone)
	urlMap.Set(constant.RegistryKey+"."+constant.WeightKey, strconv.FormatInt(registriesConfig.Weight, 10))
	urlMap.Set(constant.RegistryTTLKey, registriesConfig.TTL)
	urlMap.Set(constant.ClientNameKey, clientNameID(registriesConfig.Protocol, registriesConfig.Address))
	urlMap.Set(constant.RegistryTypeKey, registriesConfig.RegistryType)

	for k, v := range registriesConfig.Params {
		urlMap.Set(k, v)
	}
	return urlMap
}

func clientNameID(protocol, address string) string {
	return strings.Join([]string{constant.RegistryConfigPrefix, protocol, address}, "-")
}

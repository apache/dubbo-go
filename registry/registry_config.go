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

package registry

import (
	"net/url"
	"strconv"
	"strings"
)

import (
	"github.com/creasty/defaults"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
)

// RegistryConfig is the configuration of the registry center
type RegistryConfig struct {
	Protocol          string            `validate:"required" yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	Timeout           string            `default:"5s" validate:"required" yaml:"timeout" json:"timeout,omitempty" property:"timeout"` // unit: second
	Group             string            `yaml:"group" json:"group,omitempty" property:"group"`
	Namespace         string            `yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	TTL               string            `default:"15m" yaml:"ttl" json:"ttl,omitempty" property:"ttl"` // unit: minute
	Address           string            `validate:"required" yaml:"address" json:"address,omitempty" property:"address"`
	Username          string            `yaml:"username" json:"username,omitempty" property:"username"`
	Password          string            `yaml:"password" json:"password,omitempty"  property:"password"`
	Simplified        bool              `yaml:"simplified" json:"simplified,omitempty"  property:"simplified"`
	Preferred         bool              `yaml:"preferred" json:"preferred,omitempty" property:"preferred"` // Always use this registry first if set to true, useful when subscribe to multiple registriesConfig
	Zone              string            `yaml:"zone" json:"zone,omitempty" property:"zone"`                // The region where the registry belongs, usually used to isolate traffics
	Weight            int64             `yaml:"weight" json:"weight,omitempty" property:"weight"`          // Affects traffic distribution among registriesConfig, useful when subscribe to multiple registriesConfig Take effect only when no preferred registry is specified.
	Params            map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
	RegistryType      string            `yaml:"registry-type"`
	UseAsMetaReport   bool              `default:"true" yaml:"use-as-meta-report" json:"use-as-meta-report,omitempty" property:"use-as-meta-report"`
	UseAsConfigCenter bool              `default:"true" yaml:"use-as-config-center" json:"use-as-config-center,omitempty" property:"use-as-config-center"`
}

// Prefix dubbo.registries
func (RegistryConfig) Prefix() string {
	return constant.RegistryConfigPrefix
}

func (c *RegistryConfig) Init() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return c.startRegistryConfig()
}

func (c *RegistryConfig) getUrlMap(roleType common.RoleType) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.RegistryGroupKey, c.Group)
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(int(roleType)))
	urlMap.Set(constant.RegistryKey, c.Protocol)
	urlMap.Set(constant.RegistryTimeoutKey, c.Timeout)
	// multi registry invoker weight label for load balance
	urlMap.Set(constant.RegistryKey+"."+constant.RegistryLabelKey, strconv.FormatBool(true))
	urlMap.Set(constant.RegistryKey+"."+constant.PreferredKey, strconv.FormatBool(c.Preferred))
	urlMap.Set(constant.RegistryKey+"."+constant.RegistryZoneKey, c.Zone)
	urlMap.Set(constant.RegistryKey+"."+constant.WeightKey, strconv.FormatInt(c.Weight, 10))
	urlMap.Set(constant.RegistryTTLKey, c.TTL)
	urlMap.Set(constant.ClientNameKey, ClientNameID(c, c.Protocol, c.Address))

	for k, v := range c.Params {
		urlMap.Set(k, v)
	}
	return urlMap
}

func (c *RegistryConfig) startRegistryConfig() error {
	c.translateRegistryAddress()
	if c.UseAsMetaReport && commonCfg.IsValid(c.Address) {
		if tmpUrl, err := c.toMetadataReportUrl(); err == nil {
			instance.SetMetadataReportInstanceByReg(tmpUrl)
		} else {
			return perrors.Wrap(err, "Start RegistryConfig failed.")
		}
	}
	return commonCfg.Verify(c)
}

// toMetadataReportUrl translate the registry configuration to the metadata reporting url
func (c *RegistryConfig) toMetadataReportUrl() (*common.URL, error) {
	res, err := common.NewURL(c.Address,
		common.WithLocation(c.Address),
		common.WithProtocol(c.Protocol),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithParamsValue(constant.TimeoutKey, c.Timeout),
		common.WithParamsValue(constant.ClientNameKey, ClientNameID(c, c.Protocol, c.Address)),
		common.WithParamsValue(constant.MetadataReportGroupKey, c.Group),
		common.WithParamsValue(constant.MetadataReportNamespaceKey, c.Namespace),
	)
	if err != nil || len(res.Protocol) == 0 {
		return nil, perrors.New("Invalid Registry Config.")
	}
	return res, nil
}

// translateRegistryAddress translate registry address
//
//	eg:address=nacos://127.0.0.1:8848 will return 127.0.0.1:8848 and protocol will set nacos
func (c *RegistryConfig) translateRegistryAddress() string {
	if strings.Contains(c.Address, "://") {
		u, err := url.Parse(c.Address)
		if err != nil {
			logger.Errorf("The registry url is invalid, error: %#v", err)
			panic(err)
		}
		c.Protocol = u.Scheme
		c.Address = strings.Join([]string{u.Host, u.Path}, "")
	}
	return c.Address
}

func (c *RegistryConfig) GetInstance(roleType common.RoleType) (Registry, error) {
	u, err := c.toURL(roleType)
	if err != nil {
		return nil, err
	}
	// if the protocol == registry, set protocol the registry value in url.params
	if u.Protocol == constant.RegistryProtocol {
		u.Protocol = u.GetParam(constant.RegistryKey, "")
	}
	return extension.GetRegistry(u.Protocol, u)
}

func (c *RegistryConfig) toURL(roleType common.RoleType) (*common.URL, error) {
	address := c.translateRegistryAddress()
	var registryURLProtocol string
	if c.RegistryType == constant.RegistryTypeService {
		// service discovery protocol
		registryURLProtocol = constant.ServiceRegistryProtocol
	} else if c.RegistryType == constant.RegistryTypeInterface {
		registryURLProtocol = constant.RegistryProtocol
	} else {
		registryURLProtocol = constant.ServiceRegistryProtocol
	}
	return common.NewURL(registryURLProtocol+"://"+address,
		common.WithParams(c.getUrlMap(roleType)),
		common.WithParamsValue(constant.RegistrySimplifiedKey, strconv.FormatBool(c.Simplified)),
		common.WithParamsValue(constant.RegistryKey, c.Protocol),
		common.WithParamsValue(constant.RegistryNamespaceKey, c.Namespace),
		common.WithParamsValue(constant.RegistryTimeoutKey, c.Timeout),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
	)
}

func (c *RegistryConfig) ToURLs(roleType common.RoleType) ([]*common.URL, error) {
	address := c.translateRegistryAddress()
	var urls []*common.URL
	var err error
	var registryURL *common.URL

	if !commonCfg.IsValid(c.Address) {
		logger.Infof("Empty or N/A registry address found, the process will work with no registry enabled " +
			"which means that the address of this instance will not be registered and not able to be found by other consumer instances.")
		return urls, nil
	}

	if c.RegistryType == constant.RegistryTypeService {
		// service discovery protocol
		if registryURL, err = c.createNewURL(constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	} else if c.RegistryType == constant.RegistryTypeInterface {
		if registryURL, err = c.createNewURL(constant.RegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	} else if c.RegistryType == constant.RegistryTypeAll {
		if registryURL, err = c.createNewURL(constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
		if registryURL, err = c.createNewURL(constant.RegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	} else {
		if registryURL, err = c.createNewURL(constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	}
	return urls, err
}

func LoadRegistries(registryIds []string, registries map[string]*RegistryConfig, roleType common.RoleType) []*common.URL {
	var registryURLs []*common.URL
	//trSlice := strings.Split(targetRegistries, ",")

	for k, registryConf := range registries {
		target := false

		// if user not config targetRegistries, default load all
		// Notice: in func "func Split(s, sep string) []string" comment:
		// if s does not contain sep and sep is not empty, SplitAfter returns
		// a slice of length 1 whose only element is s. So we have to add the
		// condition when targetRegistries string is not set (it will be "" when not set)
		if len(registryIds) == 0 || (len(registryIds) == 1 && registryIds[0] == "") {
			target = true
		} else {
			// else if user config targetRegistries
			for _, tr := range registryIds {
				if tr == k {
					target = true
					break
				}
			}
		}

		if target {
			if urls, err := registryConf.ToURLs(roleType); err != nil {
				logger.Errorf("The registry id: %s url is invalid, error: %#v", k, err)
				panic(err)
			} else {
				registryURLs = append(registryURLs, urls...)
			}
		}
	}

	return registryURLs
}

// ClientNameID unique identifier id for client
func ClientNameID(config *RegistryConfig, protocol, address string) string {
	return strings.Join([]string{config.Prefix(), protocol, address}, "-")
}

func (c *RegistryConfig) createNewURL(protocol string, address string, roleType common.RoleType) (*common.URL, error) {
	return common.NewURL(protocol+"://"+address,
		common.WithParams(c.getUrlMap(roleType)),
		common.WithParamsValue(constant.RegistrySimplifiedKey, strconv.FormatBool(c.Simplified)),
		common.WithParamsValue(constant.RegistryKey, c.Protocol),
		common.WithParamsValue(constant.RegistryNamespaceKey, c.Namespace),
		common.WithParamsValue(constant.RegistryTimeoutKey, c.Timeout),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
	)
}

// DynamicUpdateProperties update registry
func (c *RegistryConfig) DynamicUpdateProperties(updateRegistryConfig *RegistryConfig) {
	// if nacos's registry timeout not equal local root config's registry timeout , update.
	if updateRegistryConfig != nil && updateRegistryConfig.Timeout != c.Timeout {
		c.Timeout = updateRegistryConfig.Timeout
		logger.Infof("RegistryConfigs Timeout was dynamically updated, new value:%v", c.Timeout)
	}
}

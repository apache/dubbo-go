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
	"net/url"
	"strconv"
	"strings"
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// RegistryConfig is the configuration of the registry center
type RegistryConfig struct {
	Protocol  string `validate:"required" yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	Timeout   string `default:"5s" validate:"required" yaml:"timeout" json:"timeout,omitempty" property:"timeout"` // unit: second
	Group     string `yaml:"group" json:"group,omitempty" property:"group"`
	Namespace string `yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	TTL       string `default:"10s" yaml:"ttl" json:"ttl,omitempty" property:"ttl"` // unit: minute
	// for registry
	Address    string `validate:"required" yaml:"address" json:"address,omitempty" property:"address"`
	Username   string `yaml:"username" json:"username,omitempty" property:"username"`
	Password   string `yaml:"password" json:"password,omitempty"  property:"password"`
	Simplified bool   `yaml:"simplified" json:"simplified,omitempty"  property:"simplified"`
	// Always use this registry first if set to true, useful when subscribe to multiple registriesConfig
	Preferred bool `yaml:"preferred" json:"preferred,omitempty" property:"preferred"`
	// The region where the registry belongs, usually used to isolate traffics
	Zone string `yaml:"zone" json:"zone,omitempty" property:"zone"`
	// Affects traffic distribution among registriesConfig,
	// useful when subscribe to multiple registriesConfig Take effect only when no preferred registry is specified.
	Weight       int64             `yaml:"weight" json:"weight,omitempty" property:"weight"`
	Params       map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
	RegistryType string            `yaml:"registry-type"`
}

// Prefix dubbo.registries
func (RegistryConfig) Prefix() string {
	return constant.RegistryConfigPrefix
}

func (c *RegistryConfig) check() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	c.translateRegistryAddress()
	return verify(c)
}

// initRegistryConfig init registry config
func initRegistryConfig(rc *RootConfig) error {
	registries := rc.Registries
	for key, reg := range registries {
		if err := reg.check(); err != nil {
			return err
		}
		registries[key] = reg
	}
	return nil
}

func (c *RegistryConfig) getUrlMap(roleType common.RoleType) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, c.Group)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(int(roleType)))
	urlMap.Set(constant.REGISTRY_KEY, c.Protocol)
	urlMap.Set(constant.REGISTRY_TIMEOUT_KEY, c.Timeout)
	// multi registry invoker weight label for load balance
	urlMap.Set(constant.REGISTRY_KEY+"."+constant.REGISTRY_LABEL_KEY, strconv.FormatBool(true))
	urlMap.Set(constant.REGISTRY_KEY+"."+constant.PREFERRED_KEY, strconv.FormatBool(c.Preferred))
	urlMap.Set(constant.REGISTRY_KEY+"."+constant.ZONE_KEY, c.Zone)
	urlMap.Set(constant.REGISTRY_KEY+"."+constant.WEIGHT_KEY, strconv.FormatInt(c.Weight, 10))
	urlMap.Set(constant.REGISTRY_TTL_KEY, c.TTL)
	for k, v := range c.Params {
		urlMap.Set(k, v)
	}
	return urlMap
}

//translateRegistryAddress translate registry address
//  eg:address=nacos://127.0.0.1:8848 will return 127.0.0.1:8848 and protocol will set nacos
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

func (c *RegistryConfig) GetInstance(roleType common.RoleType) (registry.Registry, error) {
	u, err := c.toURL(roleType)
	if err != nil {
		return nil, err
	}
	// if the protocol == registry, set protocol the registry value in url.params
	if u.Protocol == constant.REGISTRY_PROTOCOL {
		u.Protocol = u.GetParam(constant.REGISTRY_KEY, "")
	}
	return extension.GetRegistry(u.Protocol, u)
}

func (c *RegistryConfig) toURL(roleType common.RoleType) (*common.URL, error) {
	address := c.translateRegistryAddress()
	var registryURLProtocol string
	if c.RegistryType == "service" {
		// service discovery protocol
		registryURLProtocol = constant.SERVICE_REGISTRY_PROTOCOL
	} else {
		registryURLProtocol = constant.REGISTRY_PROTOCOL
	}
	return common.NewURL(registryURLProtocol+"://"+address,
		common.WithParams(c.getUrlMap(roleType)),
		common.WithParamsValue(constant.SIMPLIFIED_KEY, strconv.FormatBool(c.Simplified)),
		common.WithParamsValue(constant.REGISTRY_KEY, c.Protocol),
		common.WithParamsValue(constant.GROUP_KEY, c.Group),
		common.WithParamsValue(constant.NAMESPACE_KEY, c.Namespace),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
	)
}

///////////////////////////////////// registry config api
const (
	// defaultZKAddr is the default registry address of zookeeper
	defaultZKAddr = "127.0.0.1:2181"

	// defaultNacosAddr is the default registry address of nacos
	defaultNacosAddr = "127.0.0.1:8848"

	// defaultRegistryTimeout is the default registry timeout
	defaultRegistryTimeout = "3s"
)

type RegistryConfigOpt func(config *RegistryConfig) *RegistryConfig

// NewRegistryConfigWithProtocolDefaultPort New default registry config
// the input @protocol can only be:
// "zookeeper" with default addr "127.0.0.1:2181"
// "nacos" with default addr "127.0.0.1:8848"
func NewRegistryConfigWithProtocolDefaultPort(protocol string) *RegistryConfig {
	switch protocol {
	case "zookeeper":
		return &RegistryConfig{
			Protocol: protocol,
			Address:  defaultZKAddr,
			Timeout:  defaultRegistryTimeout,
		}
	case "nacos":
		return &RegistryConfig{
			Protocol: protocol,
			Address:  defaultNacosAddr,
			Timeout:  defaultRegistryTimeout,
		}
	default:
		return &RegistryConfig{
			Protocol: protocol,
		}
	}
}

// NewRegistryConfig creates New RegistryConfig with @opts
func NewRegistryConfig(opts ...RegistryConfigOpt) *RegistryConfig {
	newRegistryConfig := NewRegistryConfigWithProtocolDefaultPort("")
	for _, v := range opts {
		newRegistryConfig = v(newRegistryConfig)
	}
	return newRegistryConfig
}

// WithRegistryProtocol returns RegistryConfigOpt with given @regProtocol name
func WithRegistryProtocol(regProtocol string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Protocol = regProtocol
		return config
	}
}

// WithRegistryAddress returns RegistryConfigOpt with given @addr registry address
func WithRegistryAddress(addr string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Address = addr
		return config
	}
}

// WithRegistryTimeOut returns RegistryConfigOpt with given @timeout registry config
func WithRegistryTimeOut(timeout string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Timeout = timeout
		return config
	}
}

// WithRegistryGroup returns RegistryConfigOpt with given @group registry group
func WithRegistryGroup(group string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Group = group
		return config
	}
}

// WithRegistryTTL returns RegistryConfigOpt with given @ttl registry ttl
func WithRegistryTTL(ttl string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.TTL = ttl
		return config
	}
}

// WithRegistryUserName returns RegistryConfigOpt with given @userName registry userName
func WithRegistryUserName(userName string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Username = userName
		return config
	}
}

// WithRegistryPassword returns RegistryConfigOpt with given @psw registry password
func WithRegistryPassword(psw string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Password = psw
		return config
	}
}

// WithRegistrySimplified returns RegistryConfigOpt with given @simplified registry simplified flag
func WithRegistrySimplified(simplified bool) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Simplified = simplified
		return config
	}
}

// WithRegistryPreferred returns RegistryConfig with given @preferred registry preferred flag
func WithRegistryPreferred(preferred bool) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Preferred = preferred
		return config
	}
}

// WithRegistryWeight returns RegistryConfigOpt with given @weight registry weight flag
func WithRegistryWeight(weight int64) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Weight = weight
		return config
	}
}

// WithRegistryParams returns RegistryConfigOpt with given registry @params
func WithRegistryParams(params map[string]string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Params = params
		return config
	}
}

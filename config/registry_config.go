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

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// RegistryConfig is the configuration of the registry center
type RegistryConfig struct {
	Protocol     string            `validate:"required" yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	Timeout      string            `default:"5s" validate:"required" yaml:"timeout" json:"timeout,omitempty" property:"timeout"` // unit: second
	Group        string            `yaml:"group" json:"group,omitempty" property:"group"`
	Namespace    string            `yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	TTL          string            `default:"10s" yaml:"ttl" json:"ttl,omitempty" property:"ttl"` // unit: minute
	Address      string            `validate:"required" yaml:"address" json:"address,omitempty" property:"address"`
	Username     string            `yaml:"username" json:"username,omitempty" property:"username"`
	Password     string            `yaml:"password" json:"password,omitempty"  property:"password"`
	Simplified   bool              `yaml:"simplified" json:"simplified,omitempty"  property:"simplified"`
	Preferred    bool              `yaml:"preferred" json:"preferred,omitempty" property:"preferred"` // Always use this registry first if set to true, useful when subscribe to multiple registriesConfig
	Zone         string            `yaml:"zone" json:"zone,omitempty" property:"zone"`                // The region where the registry belongs, usually used to isolate traffics
	Weight       int64             `yaml:"weight" json:"weight,omitempty" property:"weight"`          // Affects traffic distribution among registriesConfig, useful when subscribe to multiple registriesConfig Take effect only when no preferred registry is specified.
	Params       map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
	RegistryType string            `yaml:"registry-type"`
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
	urlMap.Set(constant.ClientNameKey, clientNameID(c, c.Protocol, c.Address))

	for k, v := range c.Params {
		urlMap.Set(k, v)
	}
	return urlMap
}

func (c *RegistryConfig) startRegistryConfig() error {
	c.translateRegistryAddress()
	if GetApplicationConfig().MetadataType == constant.DefaultMetadataStorageType && c.RegistryType == constant.ServiceKey {
		if tmpUrl, err := c.toMetadataReportUrl(); err == nil {
			instance.SetMetadataReportInstanceByReg(tmpUrl)
		} else {
			return perrors.Wrap(err, "Start RegistryConfig failed.")
		}
	}
	return verify(c)
}

// toMetadataReportUrl translate the registry configuration to the metadata reporting url
func (c *RegistryConfig) toMetadataReportUrl() (*common.URL, error) {
	res, err := common.NewURL(c.Address,
		common.WithLocation(c.Address),
		common.WithProtocol(c.Protocol),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithParamsValue(constant.TimeoutKey, c.Timeout),
		common.WithParamsValue(constant.ClientNameKey, clientNameID(c, c.Protocol, c.Address)),
		common.WithParamsValue(constant.MetadataReportGroupKey, c.Group),
		common.WithParamsValue(constant.MetadataReportNamespaceKey, c.Namespace),
	)
	if err != nil || len(res.Protocol) == 0 {
		return nil, perrors.New("Invalid Registry Config.")
	}
	return res, nil
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
	if u.Protocol == constant.RegistryProtocol {
		u.Protocol = u.GetParam(constant.RegistryKey, "")
	}
	return extension.GetRegistry(u.Protocol, u)
}

func (c *RegistryConfig) toURL(roleType common.RoleType) (*common.URL, error) {
	address := c.translateRegistryAddress()
	var registryURLProtocol string
	if c.RegistryType == "service" {
		// service discovery protocol
		registryURLProtocol = constant.ServiceRegistryProtocol
	} else {
		registryURLProtocol = constant.RegistryProtocol
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

const (
	defaultZKAddr          = "127.0.0.1:2181" // default registry address of zookeeper
	defaultNacosAddr       = "127.0.0.1:8848" // the default registry address of nacos
	defaultRegistryTimeout = "3s"             // the default registry timeout
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

func NewRegistryConfigBuilder() *RegistryConfigBuilder {
	return &RegistryConfigBuilder{
		registryConfig: &RegistryConfig{},
	}
}

type RegistryConfigBuilder struct {
	registryConfig *RegistryConfig
}

func (rcb *RegistryConfigBuilder) SetProtocol(protocol string) *RegistryConfigBuilder {
	rcb.registryConfig.Protocol = protocol
	return rcb
}

func (rcb *RegistryConfigBuilder) SetTimeout(timeout string) *RegistryConfigBuilder {
	rcb.registryConfig.Timeout = timeout
	return rcb
}

func (rcb *RegistryConfigBuilder) SetGroup(group string) *RegistryConfigBuilder {
	rcb.registryConfig.Group = group
	return rcb
}

func (rcb *RegistryConfigBuilder) SetNamespace(namespace string) *RegistryConfigBuilder {
	rcb.registryConfig.Namespace = namespace
	return rcb
}

func (rcb *RegistryConfigBuilder) SetTTL(ttl string) *RegistryConfigBuilder {
	rcb.registryConfig.TTL = ttl
	return rcb
}

func (rcb *RegistryConfigBuilder) SetAddress(address string) *RegistryConfigBuilder {
	rcb.registryConfig.Address = address
	return rcb
}

func (rcb *RegistryConfigBuilder) SetUsername(username string) *RegistryConfigBuilder {
	rcb.registryConfig.Username = username
	return rcb
}

func (rcb *RegistryConfigBuilder) SetPassword(password string) *RegistryConfigBuilder {
	rcb.registryConfig.Password = password
	return rcb
}

func (rcb *RegistryConfigBuilder) SetSimplified(simplified bool) *RegistryConfigBuilder {
	rcb.registryConfig.Simplified = simplified
	return rcb
}

func (rcb *RegistryConfigBuilder) SetPreferred(preferred bool) *RegistryConfigBuilder {
	rcb.registryConfig.Preferred = preferred
	return rcb
}

func (rcb *RegistryConfigBuilder) SetZone(zone string) *RegistryConfigBuilder {
	rcb.registryConfig.Zone = zone
	return rcb
}

func (rcb *RegistryConfigBuilder) SetWeight(weight int64) *RegistryConfigBuilder {
	rcb.registryConfig.Weight = weight
	return rcb
}

func (rcb *RegistryConfigBuilder) SetParams(params map[string]string) *RegistryConfigBuilder {
	rcb.registryConfig.Params = params
	return rcb
}

func (rcb *RegistryConfigBuilder) AddParam(key, value string) *RegistryConfigBuilder {
	if rcb.registryConfig.Params == nil {
		rcb.registryConfig.Params = make(map[string]string)
	}
	rcb.registryConfig.Params[key] = value
	return rcb
}

func (rcb *RegistryConfigBuilder) SetRegistryType(registryType string) *RegistryConfigBuilder {
	rcb.registryConfig.RegistryType = registryType
	return rcb
}

func (rcb *RegistryConfigBuilder) Build() *RegistryConfig {
	if err := rcb.registryConfig.Init(); err != nil {
		panic(err)
	}
	return rcb.registryConfig
}

// DynamicUpdateProperties update registry
func (c *RegistryConfig) DynamicUpdateProperties(updateRegistryConfig *RegistryConfig) {
	// if nacos's registry timeout not equal local root config's registry timeout , update.
	if updateRegistryConfig != nil && updateRegistryConfig.Timeout != c.Timeout {
		c.Timeout = updateRegistryConfig.Timeout
		logger.Infof("RegistryConfigs Timeout was dynamically updated, new value:%v", c.Timeout)
	}
}

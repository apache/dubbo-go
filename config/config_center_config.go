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
	"strings"
)

import (
	"github.com/creasty/defaults"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

// CenterConfig is configuration for config center
//
// ConfigCenter also introduced concepts of namespace and group to better manage Key-Value pairs by group,
// those configs are already built-in in many professional third-party configuration centers.
// In most cases, namespace is used to isolate different tenants, while group is used to divide the key set from one tenant into groups.
//
// CenterConfig has currently supported Zookeeper, Nacos, Etcd, Consul, Apollo
type CenterConfig struct {
	Protocol string `validate:"required" yaml:"protocol"  json:"protocol,omitempty"`
	Address  string `validate:"required" yaml:"address" json:"address,omitempty"`
	DataId   string `validate:"required" yaml:"data-id" json:"data-id,omitempty"`
	Cluster  string `yaml:"cluster" json:"cluster,omitempty"`
	Group    string `default:"dubbo" yaml:"group" json:"group,omitempty"`
	Username string `yaml:"username" json:"username,omitempty"`
	Password string `yaml:"password" json:"password,omitempty"`
	// Deprecated
	LogDir string `yaml:"log-dir" json:"log-dir,omitempty"`
	// Deprecated
	ConfigFile string `default:"dubbo.properties" yaml:"config-file"  json:"config-file,omitempty"`
	Namespace  string `default:"dubbo" yaml:"namespace"  json:"namespace,omitempty"`
	// Deprecated
	AppConfigFile string `default:"dubbo.properties" yaml:"app-config-file"  json:"app-config-file,omitempty"`
	// Deprecated
	AppID   string `default:"dubbo" yaml:"app-id"  json:"app-id,omitempty"`
	Timeout string `default:"10s" yaml:"timeout"  json:"timeout,omitempty"`
	// Deprecated
	RemoteRef string            `required:"false"  yaml:"remote-ref"  json:"remote-ref,omitempty"`
	Params    map[string]string `yaml:"params"  json:"parameters,omitempty"`
}

// Prefix dubbo.config-center
func (CenterConfig) Prefix() string {
	return constant.ConfigCenterPrefix
}

func (c *CenterConfig) check() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	c.translateConfigAddress()
	return verify(c)
}

func (c *CenterConfig) Init(rc *RootConfig) error {
	if c == nil {
		return nil
	}
	if err := c.check(); err != nil {
		return err
	}
	return startConfigCenter(rc)
}

// GetUrlMap gets url map from ConfigCenterConfig
func (c *CenterConfig) GetUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.CONFIG_NAMESPACE_KEY, c.Namespace)
	urlMap.Set(constant.CONFIG_GROUP_KEY, c.Group)
	urlMap.Set(constant.CONFIG_CLUSTER_KEY, c.Cluster)
	urlMap.Set(constant.CONFIG_APP_ID_KEY, c.AppID)
	urlMap.Set(constant.CONFIG_LOG_DIR_KEY, c.LogDir)
	urlMap.Set(constant.CONFIG_USERNAME_KEY, c.Username)
	urlMap.Set(constant.CONFIG_PASSWORD_KEY, c.Password)
	urlMap.Set(constant.CONFIG_TIMEOUT_KEY, c.Timeout)

	for key, val := range c.Params {
		urlMap.Set(key, val)
	}
	return urlMap
}

//translateConfigAddress translate config address
//  eg:address=nacos://127.0.0.1:8848 will return 127.0.0.1:8848 and protocol will set nacos
func (c *CenterConfig) translateConfigAddress() string {
	if strings.Contains(c.Address, "://") {
		translatedUrl, err := url.Parse(c.Address)
		if err != nil {
			logger.Errorf("The config address:%s is invalid, error: %#v", c.Address, err)
			panic(err)
		}
		c.Protocol = translatedUrl.Scheme
		c.Address = strings.Replace(c.Address, translatedUrl.Scheme+"://", "", -1)
	}
	return c.Address
}

// toURL will compatible with baseConfig.ShutdownConfig.Address and baseConfig.ShutdownConfig.RemoteRef before 1.6.0
// After 1.6.0 will not compatible, only baseConfig.ShutdownConfig.RemoteRef
func (c *CenterConfig) toURL() (*common.URL, error) {
	return common.NewURL(c.Address,
		common.WithProtocol(c.Protocol),
		common.WithParams(c.GetUrlMap()))
}

// startConfigCenter will start the config center.
// it will prepare the environment
func startConfigCenter(rc *RootConfig) error {
	cc := rc.ConfigCenter
	configCenterUrl, err := cc.toURL()
	if err != nil {
		return err
	}
	strConf, err := cc.prepareEnvironment(configCenterUrl)
	if err != nil {
		return errors.WithMessagef(err, "start config center error!")
	}

	koan := koanf.New(".")
	if err = koan.Load(rawbytes.Provider([]byte(strConf)), yaml.Parser()); err != nil {
		return err
	}
	if err = koan.UnmarshalWithConf(rc.Prefix(),
		rc, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return err
	}

	return nil
}

func (c *CenterConfig) GetDynamicConfiguration() (config_center.DynamicConfiguration, error) {
	configCenterUrl, err := c.toURL()
	if err != nil {
		return nil, err
	}
	factory := extension.GetConfigCenterFactory(configCenterUrl.Protocol)
	if factory == nil {
		return nil, errors.New("get config center factory failed")
	}
	return factory.GetDynamicConfiguration(configCenterUrl)
}

func (c *CenterConfig) prepareEnvironment(configCenterUrl *common.URL) (string, error) {
	factory := extension.GetConfigCenterFactory(configCenterUrl.Protocol)
	if factory == nil {
		return "", errors.New("get config center factory failed")
	}
	dynamicConfig, err := factory.GetDynamicConfiguration(configCenterUrl)
	if err != nil {
		logger.Errorf("Get dynamic configuration error , error message is %v", err)
		return "", errors.WithStack(err)
	}
	envInstance := conf.GetEnvInstance()
	envInstance.SetDynamicConfiguration(dynamicConfig)

	return dynamicConfig.GetProperties(c.DataId, config_center.WithGroup(c.Group))
}

func NewConfigCenterConfigBuilder() *ConfigCenterConfigBuilder {
	return &ConfigCenterConfigBuilder{configCenterConfig: newEmptyConfigCenterConfig()}
}

type ConfigCenterConfigBuilder struct {
	configCenterConfig *CenterConfig
}

func (ccb *ConfigCenterConfigBuilder) SetProtocol(protocol string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Protocol = protocol
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetUserName(userName string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Username = userName
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetAddress(address string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Address = address
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetPassword(password string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Password = password
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetNamespace(namespace string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Namespace = namespace
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetDataID(dataID string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.DataId = dataID
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetGroup(group string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Group = group
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) Build() *CenterConfig {
	return ccb.configCenterConfig
}

func newEmptyConfigCenterConfig() *CenterConfig {
	return &CenterConfig{
		Params: make(map[string]string),
	}
}

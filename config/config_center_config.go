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

func GetConfigCenterInstance(opts ...CenterConfigOpt) *CenterConfig {
	cc := &CenterConfig{
		Params: make(map[string]string, 1),
	}
	for _, opt := range opts {
		opt(cc)
	}
	return cc
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
	//remoteRef := baseConfig.ConfigCenterConfig.RemoteRef
	//// if set remote ref use remote
	//if len(remoteRef) <= 0 {
	//	return common.NewURL(baseConfig.ConfigCenterConfig.Address,
	//		common.WithProtocol(baseConfig.ConfigCenterConfig.Protocol),
	//		common.WithParams(baseConfig.ConfigCenterConfig.GetUrlMap()))
	//}
	//rc, ok := baseConfig.GetRemoteConfig(remoteRef)
	//if !ok {
	//	return nil, perrors.New("Could not find out the remote ref config, name: " + remoteRef)
	//}
	//// set protocol if remote not set
	//if len(rc.Protocol) <= 0 {
	//	rc.Protocol = baseConfig.ConfigCenterConfig.Protocol
	//}
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
	//if err != nil {
	//	logger.Errorf("Get config content in dynamic configuration error , error message is %v", err)
	//	return errors.WithStack(err)
	//}
	//yaml.Unmarshal([]byte(conten),rootConfig)
	//var appGroup string
	//var appContent string
	//if config2.providerConfig != nil && config2.providerConfig.ApplicationConfig != nil &&
	//	reflect.ValueOf(baseConfig.fatherConfig).Elem().Type().Name() == "ProviderConfig" {
	//	appGroup = config2.providerConfig.ApplicationConfig.Name
	//} else if config2.consumerConfig != nil && config2.consumerConfig.ApplicationConfig != nil &&
	//	reflect.ValueOf(baseConfig.fatherConfig).Elem().Type().Name() == "ConsumerConfig" {
	//	appGroup = config2.consumerConfig.ApplicationConfig.Name
	//}
	//
	//if len(appGroup) != 0 {
	//	configFile := baseConfig.ConfigCenterConfig.AppConfigFile
	//	if len(configFile) == 0 {
	//		configFile = baseConfig.ConfigCenterConfig.ConfigFile
	//	}
	//	appContent, err = dynamicConfig.GetProperties(configFile, config_center.WithGroup(appGroup))
	//	if err != nil {
	//		return perrors.WithStack(err)
	//	}
	//}
	//// global config file
	//mapContent, err := dynamicConfig.Parser().Parse(content)
	//if err != nil {
	//	return perrors.WithStack(err)
	//}
	//envInstance.UpdateExternalConfigMap(mapContent)
	//
	//// appGroup config file
	//if len(appContent) != 0 {
	//	appMapContent, err := dynamicConfig.Parser().Parse(appContent)
	//	if err != nil {
	//		return perrors.WithStack(err)
	//	}
	//	envInstance.UpdateAppExternalConfigMap(appMapContent)
	//}
}

type CenterConfigOpt func(config *CenterConfig)

func NewConfigCenterConfig(opts ...CenterConfigOpt) *CenterConfig {
	centerConfig := &CenterConfig{
		Params: make(map[string]string),
	}
	for _, o := range opts {
		o(centerConfig)
	}
	return centerConfig
}

// WithConfigCenterProtocol set ProtocolConfig with given protocolName protocol
func WithConfigCenterProtocol(protocol string) CenterConfigOpt {
	return func(config *CenterConfig) {
		config.Protocol = protocol
	}
}

// WithConfigCenterAddress set ProtocolConfig with given @addr
func WithConfigCenterAddress(addr string) CenterConfigOpt {
	return func(config *CenterConfig) {
		config.Address = addr
	}
}

// WithConfigCenterDataID set ProtocolConfig with given @dataID
func WithConfigCenterDataID(dataID string) CenterConfigOpt {
	return func(config *CenterConfig) {
		config.DataId = dataID
	}
}

// WithConfigCenterGroup set ProtocolConfig with given @group
func WithConfigCenterGroup(group string) CenterConfigOpt {
	return func(config *CenterConfig) {
		config.Group = group
	}
}

// WithConfigCenterUsername set ProtocolConfig with given @username
func WithConfigCenterUsername(username string) CenterConfigOpt {
	return func(config *CenterConfig) {
		config.Username = username
	}
}

// WithConfigCenterPassword set ProtocolConfig with given @password
func WithConfigCenterPassword(password string) CenterConfigOpt {
	return func(config *CenterConfig) {
		config.Password = password
	}
}

// WithConfigCenterNamespace set ProtocolConfig with given @namespace
func WithConfigCenterNamespace(namespace string) CenterConfigOpt {
	return func(config *CenterConfig) {
		config.Namespace = namespace
	}
}

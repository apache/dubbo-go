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
	Protocol      string `yaml:"protocol"  json:"protocol,omitempty"`
	Address       string `yaml:"address" json:"address,omitempty"`
	Cluster       string `yaml:"cluster" json:"cluster,omitempty"`
	Group         string `default:"dubbo" yaml:"group" json:"group,omitempty"`
	Username      string `yaml:"username" json:"username,omitempty"`
	Password      string `yaml:"password" json:"password,omitempty"`
	LogDir        string `yaml:"log-dir" json:"log-dir,omitempty"`
	ConfigFile    string `default:"dubbo.properties" yaml:"config-file"  json:"config-file,omitempty"`
	Namespace     string `default:"dubbo" yaml:"namespace"  json:"namespace,omitempty"`
	AppConfigFile string `default:"dubbo.properties" yaml:"app-config-file"  json:"app-config-file,omitempty"`
	AppID         string `default:"dubbo" yaml:"app-id"  json:"app-id,omitempty"`
	Timeout       string `default:"10s" yaml:"timeout"  json:"timeout,omitempty"`
	// Deprecated
	RemoteRef string            `required:"false"  yaml:"remote-ref"  json:"remote-ref,omitempty"`
	Params    map[string]string `yaml:"params"  json:"parameters,omitempty"`
}

func (c *CenterConfig) CheckConfig() error {
	// todo check
	defaults.MustSet(c)
	c.translateConfigAddress()
	return verify(c)
}

func (c *CenterConfig) Validate() {
	// todo set default application
}

// UnmarshalYAML unmarshal the ConfigCenterConfig by @unmarshal function
func (c *CenterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain CenterConfig
	return unmarshal((*plain)(c))
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
func (c *CenterConfig) startConfigCenter() error {
	newUrl, err := c.toURL()
	if err != nil {
		return err
	}
	if err = c.prepareEnvironment(newUrl); err != nil {
		return errors.WithMessagef(err, "start config center error!")
	}
	// c.fresh()
	return nil
}

func (c *CenterConfig) prepareEnvironment(configCenterUrl *common.URL) error {
	factory := extension.GetConfigCenterFactory(configCenterUrl.Protocol)
	dynamicConfig, err := factory.GetDynamicConfiguration(configCenterUrl)
	if err != nil {
		logger.Errorf("Get dynamic configuration error , error message is %v", err)
		return errors.WithStack(err)
	}
	envInstance := conf.GetEnvInstance()
	envInstance.SetDynamicConfiguration(dynamicConfig)
	_, err = dynamicConfig.GetProperties(c.ConfigFile, config_center.WithGroup(c.Group))
	if err != nil {
		logger.Errorf("Get config content in dynamic configuration error , error message is %v", err)
		return errors.WithStack(err)
	}
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

	return nil
}

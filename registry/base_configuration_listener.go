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
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// BaseConfigurationListener provides shared logic for processing configuration center updates.
type BaseConfigurationListener struct {
	lock                    sync.RWMutex
	configurators           []config_center.Configurator
	dynamicConfiguration    config_center.DynamicConfiguration
	defaultConfiguratorFunc func(url *common.URL) config_center.Configurator
}

// Configurators gets Configurator from config center
func (bcl *BaseConfigurationListener) Configurators() []config_center.Configurator {
	return bcl.snapshotConfigurators()
}

// InitWith will init BaseConfigurationListener by @key+@Listener+@f
func (bcl *BaseConfigurationListener) InitWith(key string, listener config_center.ConfigurationListener,
	f func(url *common.URL) config_center.Configurator) {

	bcl.lock.Lock()
	bcl.dynamicConfiguration = config.GetEnvInstance().GetDynamicConfiguration()
	if bcl.dynamicConfiguration == nil {
		//set configurators to empty
		bcl.configurators = []config_center.Configurator{}
		bcl.lock.Unlock()
		return
	}
	bcl.defaultConfiguratorFunc = f
	dynamicConfiguration := bcl.dynamicConfiguration
	bcl.lock.Unlock()
	dynamicConfiguration.AddListener(key, listener)
	if rawConfig, err := dynamicConfiguration.GetRule(key,
		config_center.WithGroup(constant.Dubbo)); err != nil {
		//set configurators to empty
		bcl.setConfigurators([]config_center.Configurator{})
		return
	} else if len(rawConfig) > 0 {
		if err := bcl.genConfiguratorFromRawRule(rawConfig); err != nil {
			logger.Error("bcl.genConfiguratorFromRawRule(rawConfig:%v) = error:%v", rawConfig, err)
		}
	}
}

// Process the notification event once there's any change happens on the config.
func (bcl *BaseConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of overriding rule, change type is: %v , raw config content is:%v", event.ConfigType, event.Value)
	if event.ConfigType == remoting.EventTypeDel {
		bcl.setConfigurators(nil)
	} else {
		if err := bcl.genConfiguratorFromRawRule(event.Value.(string)); err != nil {
			logger.Error(perrors.WithStack(err))
		}
	}
}

func (bcl *BaseConfigurationListener) genConfiguratorFromRawRule(rawConfig string) error {
	bcl.lock.RLock()
	dynamicConfiguration := bcl.dynamicConfiguration
	defaultConfiguratorFunc := bcl.defaultConfiguratorFunc
	bcl.lock.RUnlock()

	urls, err := dynamicConfiguration.Parser().ParseToUrls(rawConfig)
	if err != nil {
		return perrors.WithMessage(err, "Failed to parse raw dynamic config and it will not take effect, the raw config is: "+
			rawConfig)
	}
	bcl.setConfigurators(ToConfigurators(urls, defaultConfiguratorFunc))
	return nil
}

// OverrideUrl gets existing configuration rule and overrides provider url before exporting.
func (bcl *BaseConfigurationListener) OverrideUrl(url *common.URL) {
	for _, v := range bcl.Configurators() {
		v.Configure(url)
	}
}

func (bcl *BaseConfigurationListener) setConfigurators(configurators []config_center.Configurator) {
	bcl.lock.Lock()
	defer bcl.lock.Unlock()
	bcl.configurators = configurators
}

func (bcl *BaseConfigurationListener) snapshotConfigurators() []config_center.Configurator {
	bcl.lock.RLock()
	defer bcl.lock.RUnlock()
	if bcl.configurators == nil {
		return nil
	}
	configurators := make([]config_center.Configurator, len(bcl.configurators))
	copy(configurators, bcl.configurators)
	return configurators
}

// ToConfigurators converts @urls by @f to config_center.Configurators
func ToConfigurators(urls []*common.URL, f func(url *common.URL) config_center.Configurator) []config_center.Configurator {
	if len(urls) == 0 {
		return nil
	}
	var configurators []config_center.Configurator
	for _, url := range urls {
		if url.Protocol == constant.EmptyProtocol {
			configurators = []config_center.Configurator{}
			break
		}

		override := url.GetParams()
		delete(override, constant.AnyhostKey)
		if len(override) == 0 {
			continue
		}
		configurators = append(configurators, f(url))
	}
	return configurators
}

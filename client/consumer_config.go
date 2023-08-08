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

package client

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"fmt"
	"strings"
	"time"
)

import (
	"github.com/creasty/defaults"

	"github.com/dubbogo/gost/log/logger"

	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	MaxWheelTimeSpan = 900e9 // 900s, 15 minute
)

// ConsumerConfig is Consumer default configuration
type ConsumerConfig struct {
	Filter                         string                      `yaml:"filter" json:"filter,omitempty" property:"filter"`
	RegistryIDs                    []string                    `yaml:"registry-ids" json:"registry-ids,omitempty" property:"registry-ids"`
	Protocol                       string                      `yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	RequestTimeout                 string                      `default:"3s" yaml:"request-timeout" json:"request-timeout,omitempty" property:"request-timeout"`
	ProxyFactory                   string                      `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	Check                          bool                        `yaml:"check" json:"check,omitempty" property:"check"`
	AdaptiveService                bool                        `default:"false" yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`
	References                     map[string]*ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	TracingKey                     string                      `yaml:"tracing-key" json:"tracing-key" property:"tracing-key"`
	FilterConf                     interface{}                 `yaml:"filter-conf" json:"filter-conf,omitempty" property:"filter-conf"`
	MaxWaitTimeForServiceDiscovery string                      `default:"3s" yaml:"max-wait-time-for-service-discovery" json:"max-wait-time-for-service-discovery,omitempty" property:"max-wait-time-for-service-discovery"`
	MeshEnabled                    bool                        `yaml:"mesh-enabled" json:"mesh-enabled,omitempty" property:"mesh-enabled"`
	rootConfig                     *config.RootConfig
}

// Prefix dubbo.consumer
func (ConsumerConfig) Prefix() string {
	return constant.ConsumerConfigPrefix
}

func (cc *ConsumerConfig) Init(rc *config.RootConfig) error {
	if cc == nil {
		return nil
	}

	buildDebugMsg := func() string {
		if len(cc.References) == 0 {
			return "empty"
		}
		consumerNames := make([]string, 0, len(cc.References))
		for k := range cc.References {
			consumerNames = append(consumerNames, k)
		}
		return strings.Join(consumerNames, ", ")
	}
	logger.Debugf("Registered consumer clients are %v", buildDebugMsg())

	cc.RegistryIDs = config.TranslateIds(cc.RegistryIDs)
	if len(cc.RegistryIDs) <= 0 {
		cc.RegistryIDs = rc.GetRegistryIds()
	}
	if cc.TracingKey == "" && len(rc.Tracing) > 0 {
		for k := range rc.Tracing {
			cc.TracingKey = k
			break
		}
	}
	for key, referenceConfig := range cc.References {
		if referenceConfig.InterfaceName == "" {
			reference := config.GetConsumerService(key)
			// try to use interface name defined by pb
			triplePBService, ok := reference.(common.TriplePBService)
			if !ok {
				logger.Errorf("Dubbo-go cannot get interface name with reference = %s."+
					"Please run the command 'go install github.com/dubbogo/dubbogo-cli/cmd/protoc-gen-go-triple@latest' to get the latest "+
					"protoc-gen-go-triple,  and then re-generate your pb file again by this tool."+
					"If you are not using pb serialization, please set 'interfaceName' field in reference config to let dubbogo get the interface name.", key)
				continue
			} else {
				// use interface name defined by pb
				referenceConfig.InterfaceName = triplePBService.XXX_InterfaceName()
			}
		}
		if err := referenceConfig.Init(rc); err != nil {
			return err
		}
	}
	if err := defaults.Set(cc); err != nil {
		return err
	}
	if err := verify(cc); err != nil {
		return err
	}

	cc.rootConfig = rc
	return nil
}

func (cc *ConsumerConfig) Load() {
	for registeredTypeName, refRPCService := range config.GetConsumerServiceMap() {
		refConfig, ok := cc.References[registeredTypeName]
		if !ok {
			// not found configuration, now new a configuration with default.
			refConfig = config.NewReferenceConfigBuilder().SetProtocol(tripleConstant.TRIPLE).Build()
			triplePBService, ok := refRPCService.(common.TriplePBService)
			if !ok {
				logger.Errorf("Dubbo-go cannot get interface name with registeredTypeName = %s."+
					"Please run the command 'go install github.com/dubbogo/dubbogo-cli/cmd/protoc-gen-go-triple@latest' to get the latest "+
					"protoc-gen-go-triple,  and then re-generate your pb file again by this tool."+
					"If you are not using pb serialization, please set 'interfaceName' field in reference config to let dubbogo get the interface name.", registeredTypeName)
				continue
			} else {
				// use interface name defined by pb
				refConfig.InterfaceName = triplePBService.XXX_InterfaceName()
			}
			if err := refConfig.Init(config.GetRootConfig()); err != nil {
				logger.Errorf(fmt.Sprintf("reference with registeredTypeName = %s init failed! err: %#v", registeredTypeName, err))
				continue
			}
		}
		//refConfig.id = registeredTypeName
		refConfig.Refer(refRPCService)
		//refConfig.Implement(refRPCService)
	}

	for interfaceName, info := range config.GetClientInfosMap() {
		refConfig, ok := cc.References[interfaceName]
		if !ok {
			logger.Errorf("Dubbo-go can not find %s Reference in References config, please check your configuration file", interfaceName)
		}
		refConfig.ReferWithInfo(info)
	}

	var maxWait int

	if maxWaitDuration, err := time.ParseDuration(cc.MaxWaitTimeForServiceDiscovery); err != nil {
		logger.Warnf("Invalid consumer max wait time for service discovery: %s, fallback to 3s", cc.MaxWaitTimeForServiceDiscovery)
		maxWait = 3
	} else {
		maxWait = int(maxWaitDuration.Seconds())
	}

	// wait for invoker is available, if wait over default 3s, then panic
	var count int
	for {
		checkok := true
		for key, ref := range cc.References {
			if (ref.Check != nil && *ref.Check && config.GetProviderService(key) == nil) ||
				(ref.Check == nil && cc.Check && config.GetProviderService(key) == nil) ||
				(ref.Check == nil && config.GetProviderService(key) == nil) { // default to true

				if !ref.CheckAvailable() {
					checkok = false
					count++
					if count > maxWait {
						errMsg := fmt.Sprintf("No provider available of the service %v.please check configuration.", ref.InterfaceName)
						logger.Error(errMsg)
						panic(errMsg)
					}
					time.Sleep(time.Second * 1)
					break
				}
			}
		}
		if checkok {
			break
		}
	}
}

func (cc *ConsumerConfig) SetRootConfig(root *config.RootConfig) {
	cc.rootConfig = root
}

// DynamicUpdateProperties dynamically update properties.
func (cc *ConsumerConfig) DynamicUpdateProperties(newConsumerConfig *ConsumerConfig) {
	if newConsumerConfig != nil && newConsumerConfig.RequestTimeout != cc.RequestTimeout {
		cc.RequestTimeout = newConsumerConfig.RequestTimeout
		logger.Infof("ConsumerConfig's RequestTimeout was dynamically updated, new value:%v", cc.RequestTimeout)
	}
}

type ConsumerOption func(*ConsumerConfig)

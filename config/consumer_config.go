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
	"fmt"
	"time"
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config/generic"
)

const (
	MaxWheelTimeSpan = 900e9 // 900s, 15 minute
)

// ConsumerConfig is Consumer default configuration
type ConsumerConfig struct {
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// support string
	RegistryIDs []string `yaml:"registry-ids" json:"registry-ids,omitempty" property:"registry-ids"`

	RequestTimeout string `default:"3s" yaml:"request-timeout" json:"request-timeout,omitempty" property:"request-timeout"`
	ProxyFactory   string `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	Check          bool   `yaml:"check" json:"check,omitempty" property:"check"`
	// adaptive service
	AdaptiveService bool `default:"false" yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`

	References map[string]*ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	TracingKey string                      `yaml:"tracing-key" json:"tracing-key" property:"tracing-key"`

	FilterConf                     interface{} `yaml:"filter-conf" json:"filter-conf,omitempty" property:"filter-conf"`
	MaxWaitTimeForServiceDiscovery string      `default:"3s" yaml:"max-wait-time-for-service-discovery" json:"max-wait-time-for-service-discovery,omitempty" property:"max-wait-time-for-service-discovery"`

	rootConfig *RootConfig
}

// Prefix dubbo.consumer
func (ConsumerConfig) Prefix() string {
	return constant.ConsumerConfigPrefix
}

func (cc *ConsumerConfig) Init(rc *RootConfig) error {
	if cc == nil {
		return nil
	}
	cc.RegistryIDs = translateRegistryIds(cc.RegistryIDs)
	if len(cc.RegistryIDs) <= 0 {
		cc.RegistryIDs = rc.getRegistryIds()
	}
	if cc.TracingKey == "" && len(rc.Tracing) > 0 {
		for k, _ := range rc.Tracing {
			cc.TracingKey = k
			break
		}
	}
	for key, referenceConfig := range cc.References {
		if referenceConfig.InterfaceName == "" {
			reference := GetConsumerService(key)
			// try to use interface name defined by pb
			triplePBService, ok := reference.(common.TriplePBService)
			if !ok {
				logger.Errorf("Dubbo-go cannot get interface name with reference = %s."+
					"Please run the command 'go install github.com/dubbogo/tools/cmd/protoc-gen-go-triple@latest' to get the latest "+
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
	for key, ref := range cc.References {
		if ref.Generic != "" {
			genericService := generic.NewGenericService(key)
			SetConsumerService(genericService)
		}
		rpcService := GetConsumerService(key)
		if rpcService == nil {
			logger.Warnf("%s does not exist!", key)
			continue
		}
		ref.id = key
		ref.Refer(rpcService)
		ref.Implement(rpcService)
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
		for _, refconfig := range cc.References {
			if (refconfig.Check != nil && *refconfig.Check) ||
				(refconfig.Check == nil && cc.Check) ||
				(refconfig.Check == nil) { // default to true

				if refconfig.invoker != nil && !refconfig.invoker.IsAvailable() {
					checkok = false
					count++
					if count > maxWait {
						errMsg := fmt.Sprintf("No provider available of the service %v.please check configuration.", refconfig.InterfaceName)
						logger.Error(errMsg)
						panic(errMsg)
					}
					time.Sleep(time.Second * 1)
					break
				}
				if refconfig.invoker == nil {
					logger.Warnf("The interface %s invoker not exist, may you should check your interface config.", refconfig.InterfaceName)
				}
			}
		}
		if checkok {
			break
		}
	}
}

// SetConsumerConfig sets consumerConfig by @c
func SetConsumerConfig(c ConsumerConfig) {
	rootConfig.Consumer = &c
}

func newEmptyConsumerConfig() *ConsumerConfig {
	newConsumerConfig := &ConsumerConfig{
		References:     make(map[string]*ReferenceConfig, 8),
		RequestTimeout: "3s",
		Check:          true,
	}
	return newConsumerConfig
}

type ConsumerConfigBuilder struct {
	consumerConfig *ConsumerConfig
}

func NewConsumerConfigBuilder() *ConsumerConfigBuilder {
	return &ConsumerConfigBuilder{consumerConfig: newEmptyConsumerConfig()}
}

func (ccb *ConsumerConfigBuilder) SetFilter(filter string) *ConsumerConfigBuilder {
	ccb.consumerConfig.Filter = filter
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetRegistryIDs(RegistryIDs ...string) *ConsumerConfigBuilder {
	ccb.consumerConfig.RegistryIDs = RegistryIDs
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetRequestTimeout(requestTimeout string) *ConsumerConfigBuilder {
	ccb.consumerConfig.RequestTimeout = requestTimeout
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetMaxWaitTimeForServiceDiscovery(maxWaitTimeForServiceDiscovery string) *ConsumerConfigBuilder {
	ccb.consumerConfig.MaxWaitTimeForServiceDiscovery = maxWaitTimeForServiceDiscovery
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetProxyFactory(proxyFactory string) *ConsumerConfigBuilder {
	ccb.consumerConfig.ProxyFactory = proxyFactory
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetCheck(check bool) *ConsumerConfigBuilder {
	ccb.consumerConfig.Check = check
	return ccb
}

func (ccb *ConsumerConfigBuilder) AddReference(referenceKey string, referenceConfig *ReferenceConfig) *ConsumerConfigBuilder {
	ccb.consumerConfig.References[referenceKey] = referenceConfig
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetReferences(references map[string]*ReferenceConfig) *ConsumerConfigBuilder {
	ccb.consumerConfig.References = references
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetFilterConf(filterConf interface{}) *ConsumerConfigBuilder {
	ccb.consumerConfig.FilterConf = filterConf
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetRootConfig(rootConfig *RootConfig) *ConsumerConfigBuilder {
	ccb.consumerConfig.rootConfig = rootConfig
	return ccb
}

func (ccb *ConsumerConfigBuilder) Build() *ConsumerConfig {
	return ccb.consumerConfig
}

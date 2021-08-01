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
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config/generic"
	"fmt"
	"time"
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	MaxWheelTimeSpan = 900e9 // 900s, 15 minute
)

// ConsumerConfig is Consumer default configuration
type ConsumerConfig struct {
	//root.RootConfig         `yaml:",inline" property:"base"`
	//center.ConsumerConfig `yaml:"-"`
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// client
	Connect_Timeout string `default:"100ms"  yaml:"connect_timeout" json:"connect_timeout,omitempty" property:"connect_timeout"`
	ConnectTimeout  time.Duration

	Registry []string `yaml:"registry" json:"registry,omitempty" property:"registry"`
	//Registries map[string]*RegistryConfig `default:"{}" yaml:"registriesConfig" json:"registriesConfig" property:"registriesConfig"`

	Request_Timeout string `yaml:"request_timeout" default:"5s" json:"request_timeout,omitempty" property:"request_timeout"`
	RequestTimeout  time.Duration
	ProxyFactory    string `yaml:"proxy_factory" default:"default" json:"proxy_factory,omitempty" property:"proxy_factory"`
	Check           *bool  `yaml:"check"  json:"check,omitempty" property:"check"`

	References   map[string]*ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	ProtocolConf interface{}                 `yaml:"protocol_conf" json:"protocol_conf,omitempty" property:"protocol_conf"`
	FilterConf   interface{}                 `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	//ShutdownConfig *ShutdownConfig                             `yaml:"shutdown_conf" json:"shutdown_conf,omitempty" property:"shutdown_conf"`
	ConfigType map[string]string `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`

	rootConfig *RootConfig
}

// Prefix dubbo.consumer
func (ConsumerConfig) Prefix() string {
	return constant.ConsumerConfigPrefix
}

// UnmarshalYAML unmarshal the ConsumerConfig by @unmarshal function
func (c *ConsumerConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ConsumerConfig
	return unmarshal((*plain)(c))
}

func (c *ConsumerConfig) CheckConfig() error {
	// todo check
	defaults.MustSet(c)
	return verify(c)
}

func (c *ConsumerConfig) Validate(rootConfig *RootConfig) {
	for k, _ := range c.References {
		c.References[k].Validate(rootConfig)
	}
	c.rootConfig = rootConfig
	// todo set default application
}

func (c *ConsumerConfig) Load() {
	for key, ref := range c.References {
		if ref.Generic {
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

	// todo Write current configuration to cache file.
	//if c.CacheFile != "" {
	//	if data, err := yaml.MarshalYML(c); err != nil {
	//		logger.Errorf("Marshal consumer config err: %s", err.Error())
	//	} else {
	//		if err := ioutil.WriteFile(c.CacheFile, data, 0666); err != nil {
	//			logger.Errorf("Write consumer config cache file err: %s", err.Error())
	//		}
	//	}
	//}

	// wait for invoker is available, if wait over default 3s, then panic
	var count int
	for {
		checkok := true
		for _, refconfig := range c.References {
			if (refconfig.Check != nil && *refconfig.Check) ||
				(refconfig.Check == nil && c.Check != nil && *c.Check) ||
				(refconfig.Check == nil && c.Check == nil) { // default to true

				if refconfig.invoker != nil && !refconfig.invoker.IsAvailable() {
					checkok = false
					count++
					if count > maxWait {
						errMsg := fmt.Sprintf("Failed to check the status of the service %v. No provider available for the service to the consumer use dubbo version %v", refconfig.InterfaceName, constant.Version)
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

// ConsumerInit loads config file to init consumer config
func ConsumerInit(confConFile string) error {
	//if confConFile == "" {
	//	return perrors.Errorf("applicationConfig configure(consumer) file name is nil")
	//}
	//consumerConfig = &ShutdownConfig{}
	//fileStream, err := yaml.UnmarshalYMLConfig(confConFile, consumerConfig)
	//if err != nil {
	//	return perrors.Errorf("unmarshalYmlConfig error %v", perrors.WithStack(err))
	//}
	//consumerConfig.fileStream = bytes.NewBuffer(fileStream)
	//// set method interfaceId & interfaceName
	//for k, v := range consumerConfig.References {
	//	// set id for reference
	//	for _, n := range consumerConfig.References[k].Methods {
	//		n.InterfaceName = v.InterfaceName
	//		n.InterfaceId = k
	//	}
	//}
	//if consumerConfig.Request_Timeout != "" {
	//	if consumerConfig.RequestTimeout, err = time.ParseDuration(consumerConfig.Request_Timeout); err != nil {
	//		return perrors.WithMessagef(err, "time.ParseDuration(Request_Timeout{%#v})", consumerConfig.Request_Timeout)
	//	}
	//	if consumerConfig.RequestTimeout >= time.Duration(MaxWheelTimeSpan) {
	//		return perrors.WithMessagef(err, "request_timeout %s should be less than %s",
	//			consumerConfig.Request_Timeout, time.Duration(MaxWheelTimeSpan))
	//	}
	//}
	//if consumerConfig.Connect_Timeout != "" {
	//	if consumerConfig.ConnectTimeout, err = time.ParseDuration(consumerConfig.Connect_Timeout); err != nil {
	//		return perrors.WithMessagef(err, "time.ParseDuration(Connect_Timeout{%#v})", consumerConfig.Connect_Timeout)
	//	}
	//}
	//
	//logger.Debugf("consumer config{%#v}\n", consumerConfig)

	return nil
}

func configCenterRefreshConsumer() error {
	//// fresh it
	//var err error
	//if consumerConfig.Request_Timeout != "" {
	//	if consumerConfig.RequestTimeout, err = time.ParseDuration(consumerConfig.Request_Timeout); err != nil {
	//		return perrors.WithMessagef(err, "time.ParseDuration(Request_Timeout{%#v})", consumerConfig.Request_Timeout)
	//	}
	//}
	//if consumerConfig.Connect_Timeout != "" {
	//	if consumerConfig.ConnectTimeout, err = time.ParseDuration(consumerConfig.Connect_Timeout); err != nil {
	//		return perrors.WithMessagef(err, "time.ParseDuration(Connect_Timeout{%#v})", consumerConfig.Connect_Timeout)
	//	}
	//}
	//if consumerConfig.ConfigCenterConfig != nil {
	//	consumerConfig.SetFatherConfig(consumerConfig)
	//	if err = consumerConfig.startConfigCenter((*consumerConfig).BaseConfig); err != nil {
	//		return perrors.Errorf("start config center error , error message is {%v}", perrors.WithStack(err))
	//	}
	//	consumerConfig.fresh()
	//}
	return nil
}

///////////////////////////////////// consumer config api
// ConsumerConfigOpt is the options to init ConsumerConfig
type ConsumerConfigOpt func(config *ConsumerConfig) *ConsumerConfig

// NewEmptyConsumerConfig returns default ConsumerConfig
// with connection timeout = 3s, request timeout = 3s
func NewEmptyConsumerConfig() *ConsumerConfig {
	check := true
	newConsumerConfig := &ConsumerConfig{
		//Registries:     make(map[string]*RegistryConfig, 8),
		References:     make(map[string]*ReferenceConfig, 8),
		ConnectTimeout: 3 * time.Second,
		RequestTimeout: 3 * time.Second,
		Check:          &check,
	}
	return newConsumerConfig
}

// NewConsumerConfig returns ConsumerConfig with @opts
func NewConsumerConfig(opts ...ConsumerConfigOpt) *ConsumerConfig {
	newConfig := NewEmptyConsumerConfig()
	for _, v := range opts {
		v(newConfig)
	}
	return newConfig
}

// WithRootConfig returns ConsumerConfigOpt with given @rootConfig
func WithRootConfig(rootConfig *RootConfig) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.rootConfig = rootConfig
		return config
	}
}

// WithConsumerReferenceConfig returns ConsumerConfigOpt with
func WithConsumerReferenceConfig(referenceKey string, refConfig *ReferenceConfig) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.References[referenceKey] = refConfig
		return config
	}
}

// WithConsumerConnTimeout returns ConsumerConfigOpt with given consumer conn @timeout
func WithConsumerConnTimeout(timeout time.Duration) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.ConnectTimeout = timeout
		return config
	}
}

// WithConsumerRequestTimeout returns ConsumerConfigOpt with given consumer request @timeout
func WithConsumerRequestTimeout(timeout time.Duration) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.RequestTimeout = timeout
		return config
	}
}

// WithConsumerConfigCheck returns ConsumerConfigOpt with given @check flag
func WithConsumerConfigCheck(check bool) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		*config.Check = check
		return config
	}
}

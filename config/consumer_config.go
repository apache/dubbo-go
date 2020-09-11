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
	"bytes"
	"time"
)

import (
	"github.com/creasty/defaults"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/yaml"
)

const (
	MaxWheelTimeSpan = 900e9 // 900s, 15 minute
)

/////////////////////////
// consumerConfig
/////////////////////////

// ConsumerConfig is Consumer default configuration
type ConsumerConfig struct {
	BaseConfig `yaml:",inline"`
	configCenter
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// client
	Connect_Timeout string `default:"100ms"  yaml:"connect_timeout" json:"connect_timeout,omitempty" property:"connect_timeout"`
	ConnectTimeout  time.Duration

	Registry   *RegistryConfig            `yaml:"registry" json:"registry,omitempty" property:"registry"`
	Registries map[string]*RegistryConfig `default:"{}" yaml:"registries" json:"registries" property:"registries"`

	Request_Timeout string `yaml:"request_timeout" default:"5s" json:"request_timeout,omitempty" property:"request_timeout"`
	RequestTimeout  time.Duration
	ProxyFactory    string `yaml:"proxy_factory" default:"default" json:"proxy_factory,omitempty" property:"proxy_factory"`
	Check           *bool  `yaml:"check"  json:"check,omitempty" property:"check"`

	References     map[string]*ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	ProtocolConf   interface{}                 `yaml:"protocol_conf" json:"protocol_conf,omitempty" property:"protocol_conf"`
	FilterConf     interface{}                 `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	ShutdownConfig *ShutdownConfig             `yaml:"shutdown_conf" json:"shutdown_conf,omitempty" property:"shutdown_conf"`
	ConfigType     map[string]string           `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`
}

// UnmarshalYAML unmarshals the ConsumerConfig by @unmarshal function
func (c *ConsumerConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ConsumerConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// nolint
func (*ConsumerConfig) Prefix() string {
	return constant.ConsumerConfigPrefix
}

// SetConsumerConfig sets consumerConfig by @c
func SetConsumerConfig(c ConsumerConfig) {
	consumerConfig = &c
}

// ConsumerInit loads config file to init consumer config
func ConsumerInit(confConFile string) error {
	if confConFile == "" {
		return perrors.Errorf("application configure(consumer) file name is nil")
	}
	consumerConfig = &ConsumerConfig{}
	fileStream, err := yaml.UnmarshalYMLConfig(confConFile, consumerConfig)
	if err != nil {
		return perrors.Errorf("unmarshalYmlConfig error %v", perrors.WithStack(err))
	}
	consumerConfig.fileStream = bytes.NewBuffer(fileStream)
	//set method interfaceId & interfaceName
	for k, v := range consumerConfig.References {
		//set id for reference
		for _, n := range consumerConfig.References[k].Methods {
			n.InterfaceName = v.InterfaceName
			n.InterfaceId = k
		}
	}
	if consumerConfig.Request_Timeout != "" {
		if consumerConfig.RequestTimeout, err = time.ParseDuration(consumerConfig.Request_Timeout); err != nil {
			return perrors.WithMessagef(err, "time.ParseDuration(Request_Timeout{%#v})", consumerConfig.Request_Timeout)
		}
		if consumerConfig.RequestTimeout >= time.Duration(MaxWheelTimeSpan) {
			return perrors.WithMessagef(err, "request_timeout %s should be less than %s",
				consumerConfig.Request_Timeout, time.Duration(MaxWheelTimeSpan))
		}
	}
	if consumerConfig.Connect_Timeout != "" {
		if consumerConfig.ConnectTimeout, err = time.ParseDuration(consumerConfig.Connect_Timeout); err != nil {
			return perrors.WithMessagef(err, "time.ParseDuration(Connect_Timeout{%#v})", consumerConfig.Connect_Timeout)
		}
	}

	logger.Debugf("consumer config{%#v}\n", consumerConfig)

	return nil
}

func configCenterRefreshConsumer() error {
	//fresh it
	var err error
	if consumerConfig.Request_Timeout != "" {
		if consumerConfig.RequestTimeout, err = time.ParseDuration(consumerConfig.Request_Timeout); err != nil {
			return perrors.WithMessagef(err, "time.ParseDuration(Request_Timeout{%#v})", consumerConfig.Request_Timeout)
		}
	}
	if consumerConfig.Connect_Timeout != "" {
		if consumerConfig.ConnectTimeout, err = time.ParseDuration(consumerConfig.Connect_Timeout); err != nil {
			return perrors.WithMessagef(err, "time.ParseDuration(Connect_Timeout{%#v})", consumerConfig.Connect_Timeout)
		}
	}
	if consumerConfig.ConfigCenterConfig != nil {
		consumerConfig.SetFatherConfig(consumerConfig)
		if err = consumerConfig.startConfigCenter((*consumerConfig).BaseConfig); err != nil {
			return perrors.Errorf("start config center error , error message is {%v}", perrors.WithStack(err))
		}
		consumerConfig.fresh()
	}
	return nil
}

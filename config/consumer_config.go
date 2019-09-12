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
	"context"
	"io/ioutil"
	"path"
	"time"
)

import (
	"github.com/creasty/defaults"
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

/////////////////////////
// consumerConfig
/////////////////////////

type ConsumerConfig struct {
	BaseConfig `yaml:",inline"`
	Filter     string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// application
	ApplicationConfig *ApplicationConfig `yaml:"application" json:"application,omitempty" property:"application"`
	// client
	Connect_Timeout string `default:"100ms"  yaml:"connect_timeout" json:"connect_timeout,omitempty" property:"connect_timeout"`
	ConnectTimeout  time.Duration

	Request_Timeout string `yaml:"request_timeout" default:"5s" json:"request_timeout,omitempty" property:"request_timeout"`
	RequestTimeout  time.Duration
	ProxyFactory    string `yaml:"proxy_factory" default:"default" json:"proxy_factory,omitempty" property:"proxy_factory"`
	Check           *bool  `yaml:"check"  json:"check,omitempty" property:"check"`

	Registry     *RegistryConfig             `yaml:"registry" json:"registry,omitempty" property:"registry"`
	Registries   map[string]*RegistryConfig  `yaml:"registries" json:"registries,omitempty" property:"registries"`
	References   map[string]*ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	ProtocolConf interface{}                 `yaml:"protocol_conf" json:"protocol_conf,omitempty" property:"protocol_conf"`
	FilterConf   interface{}                 `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf" `
}

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

func (*ConsumerConfig) Prefix() string {
	return constant.ConsumerConfigPrefix
}

func SetConsumerConfig(c ConsumerConfig) {
	consumerConfig = &c
}

func GetConsumerConfig() ConsumerConfig {
	if consumerConfig == nil {
		logger.Warnf("consumerConfig is nil!")
		return ConsumerConfig{}
	}
	return *consumerConfig
}

func ConsumerInit(confConFile string) error {
	if confConFile == "" {
		return perrors.Errorf("application configure(consumer) file name is nil")
	}

	if path.Ext(confConFile) != ".yml" {
		return perrors.Errorf("application configure file name{%v} suffix must be .yml", confConFile)
	}

	confFileStream, err := ioutil.ReadFile(confConFile)
	if err != nil {
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", confConFile, perrors.WithStack(err))
	}
	consumerConfig = &ConsumerConfig{}
	err = yaml.Unmarshal(confFileStream, consumerConfig)
	if err != nil {
		return perrors.Errorf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
	}

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
	if consumerConfig.ConfigCenterConfig != nil {
		consumerConfig.SetFatherConfig(consumerConfig)
		if err := consumerConfig.startConfigCenter(context.Background()); err != nil {
			return perrors.Errorf("start config center error , error message is {%v}", perrors.WithStack(err))
		}
		consumerConfig.fresh()
	}
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
	return err
}

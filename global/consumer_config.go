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

package global

type ConsumerConfig struct {
	Filter          string   `yaml:"filter" json:"filter,omitempty" property:"filter"`
	RegistryIDs     []string `yaml:"registry-ids" json:"registry-ids,omitempty" property:"registry-ids"`
	Protocol        string   `yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	RequestTimeout  string   `default:"3s" yaml:"request-timeout" json:"request-timeout,omitempty" property:"request-timeout"`
	ProxyFactory    string   `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	Check           bool     `yaml:"check" json:"check,omitempty" property:"check"`
	AdaptiveService bool     `default:"false" yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`
	// there is no need to configure References, it will be replaced by instance.NewClient
	//References                     map[string]*client.ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	TracingKey                     string      `yaml:"tracing-key" json:"tracing-key" property:"tracing-key"`
	FilterConf                     interface{} `yaml:"filter-conf" json:"filter-conf,omitempty" property:"filter-conf"`
	MaxWaitTimeForServiceDiscovery string      `default:"3s" yaml:"max-wait-time-for-service-discovery" json:"max-wait-time-for-service-discovery,omitempty" property:"max-wait-time-for-service-discovery"`
	MeshEnabled                    bool        `yaml:"mesh-enabled" json:"mesh-enabled,omitempty" property:"mesh-enabled"`
}

func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		RequestTimeout: "3s",
		Check:          true,
	}
}

type ConsumerOption func(*ConsumerConfig)

func WithConsumer_Filter(filter string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Filter = filter
	}
}

func WithConsumer_RegistryIDs(ids []string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.RegistryIDs = ids
	}
}

func WithConsumer_Protocol(protocol string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Protocol = protocol
	}
}

func WithConsumer_RequestTimeout(timeout string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.RequestTimeout = timeout
	}
}

func WithConsumer_ProxyFactory(factory string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.ProxyFactory = factory
	}
}

func WithConsumer_Check(flag bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Check = flag
	}
}

func WithConsumer_AdaptiveService(flag bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.AdaptiveService = flag
	}
}

func WithConsumer_TracingKey(key string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.TracingKey = key
	}
}

func WithConsumer_FilterConf(conf interface{}) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.FilterConf = conf
	}
}

func WithConsumer_MaxWaitTimeForServiceDiscovery(time string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.MaxWaitTimeForServiceDiscovery = time
	}
}

func WithConsumer_MeshEnabled(flag bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.MeshEnabled = flag
	}
}

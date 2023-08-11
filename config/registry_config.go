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
	"dubbo.apache.org/dubbo-go/v3/registry"
)

const (
	defaultZKAddr          = "127.0.0.1:2181" // default registry address of zookeeper
	defaultNacosAddr       = "127.0.0.1:8848" // the default registry address of nacos
	defaultRegistryTimeout = "3s"             // the default registry timeout
)

type RegistryConfigOpt func(config *registry.RegistryConfig) *registry.RegistryConfig

// NewRegistryConfigWithProtocolDefaultPort New default registry config
// the input @protocol can only be:
// "zookeeper" with default addr "127.0.0.1:2181"
// "nacos" with default addr "127.0.0.1:8848"
func NewRegistryConfigWithProtocolDefaultPort(protocol string) *registry.RegistryConfig {
	switch protocol {
	case "zookeeper":
		return &registry.RegistryConfig{
			Protocol: protocol,
			Address:  defaultZKAddr,
			Timeout:  defaultRegistryTimeout,
		}
	case "nacos":
		return &registry.RegistryConfig{
			Protocol: protocol,
			Address:  defaultNacosAddr,
			Timeout:  defaultRegistryTimeout,
		}
	default:
		return &registry.RegistryConfig{
			Protocol: protocol,
		}
	}
}

// NewRegistryConfig creates New registry.RegistryConfig with @opts
func NewRegistryConfig(opts ...RegistryConfigOpt) *registry.RegistryConfig {
	newRegistryConfig := NewRegistryConfigWithProtocolDefaultPort("")
	for _, v := range opts {
		newRegistryConfig = v(newRegistryConfig)
	}
	return newRegistryConfig
}

// WithRegistryProtocol returns RegistryConfigOpt with given @regProtocol name
func WithRegistryProtocol(regProtocol string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Protocol = regProtocol
		return config
	}
}

// WithRegistryAddress returns RegistryConfigOpt with given @addr registry address
func WithRegistryAddress(addr string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Address = addr
		return config
	}
}

// WithRegistryTimeOut returns RegistryConfigOpt with given @timeout registry config
func WithRegistryTimeOut(timeout string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Timeout = timeout
		return config
	}
}

// WithRegistryGroup returns RegistryConfigOpt with given @group registry group
func WithRegistryGroup(group string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Group = group
		return config
	}
}

// WithRegistryTTL returns RegistryConfigOpt with given @ttl registry ttl
func WithRegistryTTL(ttl string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.TTL = ttl
		return config
	}
}

// WithRegistryUserName returns RegistryConfigOpt with given @userName registry userName
func WithRegistryUserName(userName string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Username = userName
		return config
	}
}

// WithRegistryPassword returns RegistryConfigOpt with given @psw registry password
func WithRegistryPassword(psw string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Password = psw
		return config
	}
}

// WithRegistrySimplified returns RegistryConfigOpt with given @simplified registry simplified flag
func WithRegistrySimplified(simplified bool) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Simplified = simplified
		return config
	}
}

// WithRegistryPreferred returns registry.RegistryConfig with given @preferred registry preferred flag
func WithRegistryPreferred(preferred bool) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Preferred = preferred
		return config
	}
}

// WithRegistryWeight returns RegistryConfigOpt with given @weight registry weight flag
func WithRegistryWeight(weight int64) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Weight = weight
		return config
	}
}

// WithRegistryParams returns RegistryConfigOpt with given registry @params
func WithRegistryParams(params map[string]string) RegistryConfigOpt {
	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
		config.Params = params
		return config
	}
}

func NewRegistryConfigBuilder() *RegistryConfigBuilder {
	return &RegistryConfigBuilder{
		registryConfig: &registry.RegistryConfig{},
	}
}

type RegistryConfigBuilder struct {
	registryConfig *registry.RegistryConfig
}

func (rcb *RegistryConfigBuilder) SetProtocol(protocol string) *RegistryConfigBuilder {
	rcb.registryConfig.Protocol = protocol
	return rcb
}

func (rcb *RegistryConfigBuilder) SetTimeout(timeout string) *RegistryConfigBuilder {
	rcb.registryConfig.Timeout = timeout
	return rcb
}

func (rcb *RegistryConfigBuilder) SetGroup(group string) *RegistryConfigBuilder {
	rcb.registryConfig.Group = group
	return rcb
}

func (rcb *RegistryConfigBuilder) SetNamespace(namespace string) *RegistryConfigBuilder {
	rcb.registryConfig.Namespace = namespace
	return rcb
}

func (rcb *RegistryConfigBuilder) SetTTL(ttl string) *RegistryConfigBuilder {
	rcb.registryConfig.TTL = ttl
	return rcb
}

func (rcb *RegistryConfigBuilder) SetAddress(address string) *RegistryConfigBuilder {
	rcb.registryConfig.Address = address
	return rcb
}

func (rcb *RegistryConfigBuilder) SetUsername(username string) *RegistryConfigBuilder {
	rcb.registryConfig.Username = username
	return rcb
}

func (rcb *RegistryConfigBuilder) SetPassword(password string) *RegistryConfigBuilder {
	rcb.registryConfig.Password = password
	return rcb
}

func (rcb *RegistryConfigBuilder) SetSimplified(simplified bool) *RegistryConfigBuilder {
	rcb.registryConfig.Simplified = simplified
	return rcb
}

func (rcb *RegistryConfigBuilder) SetPreferred(preferred bool) *RegistryConfigBuilder {
	rcb.registryConfig.Preferred = preferred
	return rcb
}

func (rcb *RegistryConfigBuilder) SetZone(zone string) *RegistryConfigBuilder {
	rcb.registryConfig.Zone = zone
	return rcb
}

func (rcb *RegistryConfigBuilder) SetWeight(weight int64) *RegistryConfigBuilder {
	rcb.registryConfig.Weight = weight
	return rcb
}

func (rcb *RegistryConfigBuilder) SetParams(params map[string]string) *RegistryConfigBuilder {
	rcb.registryConfig.Params = params
	return rcb
}

func (rcb *RegistryConfigBuilder) AddParam(key, value string) *RegistryConfigBuilder {
	if rcb.registryConfig.Params == nil {
		rcb.registryConfig.Params = make(map[string]string)
	}
	rcb.registryConfig.Params[key] = value
	return rcb
}

func (rcb *RegistryConfigBuilder) SetRegistryType(registryType string) *RegistryConfigBuilder {
	rcb.registryConfig.RegistryType = registryType
	return rcb
}

func (rcb *RegistryConfigBuilder) Build() *registry.RegistryConfig {
	if err := rcb.registryConfig.Init(); err != nil {
		panic(err)
	}
	return rcb.registryConfig
}

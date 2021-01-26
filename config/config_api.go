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
	"time"
)

//////////////////////////////////// default registry config
const (
	defaultZKAddr          = "127.0.0.1:2181"
	defaultConsulAddr      = "127.0.0.1:8500"
	defaultNacosAddr       = "127.0.0.1:8848"
	defaultRegistryTimeout = "3s"
)

func NewDefaultRegistryConfig(protocol string) *RegistryConfig {
	switch protocol {
	case "zookeeper":
		return &RegistryConfig{
			Protocol:   protocol,
			Address:    defaultZKAddr,
			TimeoutStr: defaultRegistryTimeout,
		}
	case "consul":
		return &RegistryConfig{
			Protocol:   protocol,
			Address:    defaultConsulAddr,
			TimeoutStr: defaultRegistryTimeout,
		}
	case "nacos":
		return &RegistryConfig{
			Protocol:   protocol,
			Address:    defaultNacosAddr,
			TimeoutStr: defaultRegistryTimeout,
		}
	default:
		return &RegistryConfig{
			Protocol: protocol,
		}
	}
}

///////////////////////////////////// registry config api
type RegistryConfigOpt func(config *RegistryConfig) *RegistryConfig

func NewRegistryConfig(opts ...RegistryConfigOpt) *RegistryConfig {
	newRegistryConfig := NewDefaultRegistryConfig("none")
	for _, v := range opts {
		newRegistryConfig = v(newRegistryConfig)
	}
	return newRegistryConfig
}

func WithRegistryProtocol(regProtocol string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Protocol = regProtocol
		return config
	}
}

func WithRegistryAddress(addr string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Address = addr
		return config
	}
}

func WithRegistryTimeOut(timeout string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.TimeoutStr = timeout
		return config
	}
}

func WithRegistryGroup(group string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Group = group
		return config
	}
}

func WithRegistryTTL(ttl string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.TTL = ttl
		return config
	}
}

func WithRegistryUserName(userName string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Username = userName
		return config
	}
}

func WithRegistryPassword(psw string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Password = psw
		return config
	}
}

func WithRegistrySimplified(simplified bool) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Simplified = simplified
		return config
	}
}

func WithRegistryPreferred(preferred bool) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Preferred = preferred
		return config
	}
}

func WithRegistryWeight(weight int64) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Weight = weight
		return config
	}
}

func WithRegistryParams(params map[string]string) RegistryConfigOpt {
	return func(config *RegistryConfig) *RegistryConfig {
		config.Params = params
		return config
	}
}

///////////////////////////////////// consumer config api
type ConsumerConfigOpt func(config *ConsumerConfig) *ConsumerConfig

func NewDefaultConsumerConfig() *ConsumerConfig {
	check := true
	newConsumerConfig := &ConsumerConfig{
		BaseConfig:     BaseConfig{},
		Registries:     make(map[string]*RegistryConfig, 8),
		References:     make(map[string]*ReferenceConfig, 8),
		ConnectTimeout: 3 * time.Second,
		RequestTimeout: 3 * time.Second,
		Check:          &check,
	}
	return newConsumerConfig
}

func NewConsumerConfig(opts ...ConsumerConfigOpt) *ConsumerConfig {
	newConfig := NewDefaultConsumerConfig()
	for _, v := range opts {
		v(newConfig)
	}
	return newConfig
}
func WithConsumerAppConfig(appConfig *ApplicationConfig) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.ApplicationConfig = appConfig
		return config
	}
}

func WithConsumerRegistryConfig(registryKey string, regConfig *RegistryConfig) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.Registries[registryKey] = regConfig
		return config
	}
}

func WithConsumerReferenceConfig(referenceKey string, refConfig *ReferenceConfig) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.References[referenceKey] = refConfig
		return config
	}
}

func WithConsumerConnTimeout(timeout time.Duration) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.ConnectTimeout = timeout
		return config
	}
}

func WithConsumerRequestTimeout(timeout time.Duration) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.RequestTimeout = timeout
		return config
	}
}

func WithConsumerConfigCenterConfig(configCenterConfig *ConfigCenterConfig) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		config.ConfigCenterConfig = configCenterConfig
		return config
	}
}

func WithConsumerConfigCheck(check bool) ConsumerConfigOpt {
	return func(config *ConsumerConfig) *ConsumerConfig {
		*config.Check = check
		return config
	}
}

//////////////////////////////////// reference config api
type ReferenceConfigOpt func(config *ReferenceConfig) *ReferenceConfig

func NewDefaultReferenceConfig() *ReferenceConfig {
	newReferenceConfig := NewReferenceConfig("", context.Background())
	newReferenceConfig.Methods = make([]*MethodConfig, 0, 8)
	newReferenceConfig.Params = make(map[string]string, 8)
	return newReferenceConfig
}

func NewReferenceConfigByAPI(opts ...ReferenceConfigOpt) *ReferenceConfig {
	newreferenceConfig := NewDefaultReferenceConfig()
	for _, v := range opts {
		v(newreferenceConfig)
	}
	return newreferenceConfig
}

func WithReferenceRegistry(registry string) ReferenceConfigOpt {
	return func(config *ReferenceConfig) *ReferenceConfig {
		config.Registry = registry
		return config
	}
}
func WithReferenceProtocol(protocol string) ReferenceConfigOpt {
	return func(config *ReferenceConfig) *ReferenceConfig {
		config.Protocol = protocol
		return config
	}
}
func WithReferenceInterface(interfaceName string) ReferenceConfigOpt {
	return func(config *ReferenceConfig) *ReferenceConfig {
		config.InterfaceName = interfaceName
		return config
	}
}
func WithReferenceCluster(cluster string) ReferenceConfigOpt {
	return func(config *ReferenceConfig) *ReferenceConfig {
		config.Cluster = cluster
		return config
	}
}
func WithReferenceMethod(methodName, retries, lb string) ReferenceConfigOpt {
	return func(config *ReferenceConfig) *ReferenceConfig {
		config.Methods = append(config.Methods, &MethodConfig{
			Name:        methodName,
			Retries:     retries,
			LoadBalance: lb,
		})
		return config
	}
}

///////////////////////////////////// provider config api
type ProviderConfigOpt func(config *ProviderConfig) *ProviderConfig

func NewDefaultProviderConfig() *ProviderConfig {
	newConsumerConfig := &ProviderConfig{
		BaseConfig: BaseConfig{
			ApplicationConfig: &ApplicationConfig{
				Name:         "dubbo",
				Module:       "module",
				Organization: "dubbo_org",
				Owner:        "dubbo",
			},
		},
		Services:   make(map[string]*ServiceConfig),
		Registries: make(map[string]*RegistryConfig, 8),
		Protocols:  make(map[string]*ProtocolConfig, 8),
	}
	return newConsumerConfig
}

func NewProviderConfig(opts ...ProviderConfigOpt) *ProviderConfig {
	newConfig := NewDefaultProviderConfig()
	for _, v := range opts {
		v(newConfig)
	}
	return newConfig
}

func WithPrividerRegistryConfig(regConfig *RegistryConfig) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Registries[regConfig.Protocol] = regConfig
		return config
	}
}

func WithProviderAppConfig(appConfig *ApplicationConfig) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.ApplicationConfig = appConfig
		return config
	}
}

func WithProviderServices(serviceName string, serviceConfig *ServiceConfig) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Services[serviceName] = serviceConfig
		return config
	}
}

func WithProviderProtocol(protocolKey, protocol, port string) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Protocols[protocolKey] = &ProtocolConfig{
			Name: protocol,
			Port: port,
		}
		return config
	}
}

func WithProviderRegistry(registryKey string, registryConfig *RegistryConfig) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Registries[registryKey] = registryConfig
		return config
	}
}

/////////////////////////////////////// service config api
type ServiceConfigOpt func(config *ServiceConfig) *ServiceConfig

func NewDefaultServiceConfig() *ServiceConfig {
	newServiceConfig := NewServiceConfig("", context.Background())
	newServiceConfig.Params = make(map[string]string)
	newServiceConfig.Methods = make([]*MethodConfig, 0, 8)
	return newServiceConfig
}

// NewServiceConfigByAPI is named as api, because there is NewServiceConfig func already declared
func NewServiceConfigByAPI(opts ...ServiceConfigOpt) *ServiceConfig {
	defaultServiceConfig := NewDefaultServiceConfig()
	for _, v := range opts {
		v(defaultServiceConfig)
	}
	return defaultServiceConfig
}

func WithServiceRegistry(registry string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Registry = registry
		return config
	}
}

func WithServiceProtocol(protocol string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Protocol = protocol
		return config
	}
}
func WithServiceInterface(interfaceName string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.InterfaceName = interfaceName
		return config
	}
}
func WithServiceLoadBalance(lb string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Loadbalance = lb
		return config
	}
}

func WithServiceWarmUpTime(warmUp string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Warmup = warmUp
		return config
	}
}

func WithServiceCluster(cluster string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Cluster = cluster
		return config
	}
}

func WithServiceMethod(name, retries, lb string) ServiceConfigOpt {
	return func(config *ServiceConfig) *ServiceConfig {
		config.Methods = append(config.Methods, &MethodConfig{
			Name:        name,
			Retries:     retries,
			LoadBalance: lb,
		})
		return config
	}
}

///////////////////////////////////////// Application config api
type ApplicationConfigOpt func(config *ApplicationConfig) *ApplicationConfig

func NewDefaultApplicationConfig() *ApplicationConfig {
	newAppConfig := &ApplicationConfig{
		Name:         "dubbo.io",
		Module:       "sample",
		Organization: "dubbo.io",
		Owner:        "dubbogo",
		Version:      "0.0.1",
		Environment:  "dev",
	}
	return newAppConfig
}

// NewApplicationConfig is named as api, because there is NewServiceConfig func already declared
func NewApplicationConfig(opts ...ApplicationConfigOpt) *ApplicationConfig {
	defaultServiceConfig := NewDefaultApplicationConfig()
	for _, v := range opts {
		v(defaultServiceConfig)
	}
	return defaultServiceConfig
}

func WithAppName(name string) ApplicationConfigOpt {
	return func(config *ApplicationConfig) *ApplicationConfig {
		config.Name = name
		return config
	}
}

func WithAppModule(module string) ApplicationConfigOpt {
	return func(config *ApplicationConfig) *ApplicationConfig {
		config.Module = module
		return config
	}
}

func WithAppOrganization(org string) ApplicationConfigOpt {
	return func(config *ApplicationConfig) *ApplicationConfig {
		config.Organization = org
		return config
	}
}

func WithAppOwner(owner string) ApplicationConfigOpt {
	return func(config *ApplicationConfig) *ApplicationConfig {
		config.Owner = owner
		return config
	}
}

func WithAppVersion(version string) ApplicationConfigOpt {
	return func(config *ApplicationConfig) *ApplicationConfig {
		config.Version = version
		return config
	}
}

func WithAppEnvironment(env string) ApplicationConfigOpt {
	return func(config *ApplicationConfig) *ApplicationConfig {
		config.Environment = env
		return config
	}
}

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

//
//import (
//	"dubbo.apache.org/dubbo-go/v3/config/applicationConfig"
//	"dubbo.apache.org/dubbo-go/v3/config/center"
//	"dubbo.apache.org/dubbo-go/v3/config/consumer"
//	"dubbo.apache.org/dubbo-go/v3/config/method"
//	"dubbo.apache.org/dubbo-go/v3/config/protocol"
//	"dubbo.apache.org/dubbo-go/v3/config/provider"
//	"dubbo.apache.org/dubbo-go/v3/config/reference"
//	"dubbo.apache.org/dubbo-go/v3/config/registry"
//	"dubbo.apache.org/dubbo-go/v3/config/service"
//	"google.golang.org/grpc/balancer/base"
//	"time"
//)
//
////////////////////////////////////// default registry config
//const (
//	// defaultZKAddr is the default registry address of zookeeper
//	defaultZKAddr = "127.0.0.1:2181"
//
//	// defaultNacosAddr is the default registry address of nacos
//	defaultNacosAddr = "127.0.0.1:8848"
//
//	// defaultRegistryTimeout is the default registry timeout
//	defaultRegistryTimeout = "3s"
//)
//
//// NewDefaultRegistryConfig New default registry config
//// the input @protocol can only be:
//// "zookeeper" with default addr "127.0.0.1:2181"
//// "nacos" with default addr "127.0.0.1:8848"
//func NewDefaultRegistryConfig(protocol string) *registry.RegistryConfig {
//	switch protocol {
//	case "zookeeper":
//		return &registry.RegistryConfig{
//			Protocol:   protocol,
//			Address:    defaultZKAddr,
//			TimeoutStr: defaultRegistryTimeout,
//		}
//	case "nacos":
//		return &registry.RegistryConfig{
//			Protocol:   protocol,
//			Address:    defaultNacosAddr,
//			TimeoutStr: defaultRegistryTimeout,
//		}
//	default:
//		return &registry.RegistryConfig{
//			Protocol: protocol,
//		}
//	}
//}
//
/////////////////////////////////////// registry config api
//type RegistryConfigOpt func(config *registry.RegistryConfig) *registry.RegistryConfig
//
//// NewRegistryConfig creates New RegistryConfig with @opts
//func NewRegistryConfig(opts ...RegistryConfigOpt) *registry.RegistryConfig {
//	newRegistryConfig := NewDefaultRegistryConfig("")
//	for _, v := range opts {
//		newRegistryConfig = v(newRegistryConfig)
//	}
//	return newRegistryConfig
//}
//
//// WithRegistryProtocol returns RegistryConfigOpt with given @regProtocol name
//func WithRegistryProtocol(regProtocol string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Protocol = regProtocol
//		return config
//	}
//}
//
//// WithRegistryAddress returns RegistryConfigOpt with given @addr registry address
//func WithRegistryAddress(addr string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Address = addr
//		return config
//	}
//}
//
//// WithRegistryTimeOut returns RegistryConfigOpt with given @timeout registry config
//func WithRegistryTimeOut(timeout string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.TimeoutStr = timeout
//		return config
//	}
//}
//
//// WithRegistryGroup returns RegistryConfigOpt with given @group registry group
//func WithRegistryGroup(group string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Group = group
//		return config
//	}
//}
//
//// WithRegistryTTL returns RegistryConfigOpt with given @ttl registry ttl
//func WithRegistryTTL(ttl string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.TTL = ttl
//		return config
//	}
//}
//
//// WithRegistryUserName returns RegistryConfigOpt with given @userName registry userName
//func WithRegistryUserName(userName string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Username = userName
//		return config
//	}
//}
//
//// WithRegistryPassword returns RegistryConfigOpt with given @psw registry password
//func WithRegistryPassword(psw string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Password = psw
//		return config
//	}
//}
//
//// WithRegistrySimplified returns RegistryConfigOpt with given @simplified registry simplified flag
//func WithRegistrySimplified(simplified bool) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Simplified = simplified
//		return config
//	}
//}
//
//// WithRegistryPreferred returns RegistryConfig with given @preferred registry preferred flag
//func WithRegistryPreferred(preferred bool) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Preferred = preferred
//		return config
//	}
//}
//
//// WithRegistryWeight returns RegistryConfigOpt with given @weight registry weight flag
//func WithRegistryWeight(weight int64) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Weight = weight
//		return config
//	}
//}
//
//// WithRegistryParams returns RegistryConfigOpt with given registry @params
//func WithRegistryParams(params map[string]string) RegistryConfigOpt {
//	return func(config *registry.RegistryConfig) *registry.RegistryConfig {
//		config.Params = params
//		return config
//	}
//}
//
/////////////////////////////////////// consumer config api
//// ConsumerConfigOpt is the options to init ShutdownConfig
//type ConsumerConfigOpt func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig
//
//// NewDefaultConsumerConfig returns default ShutdownConfig
//// with connection timeout = 3s, request timeout = 3s
//func NewDefaultConsumerConfig() *consumer.ShutdownConfig {
//	check := true
//	newConsumerConfig := &consumer.ShutdownConfig{
//		BaseConfig:     base.ShutdownConfig{},
//		Registries:     make(map[string]*registry.RegistryConfig, 8),
//		References:     make(map[string]*reference.ReferenceConfig, 8),
//		ConnectTimeout: 3 * time.Second,
//		RequestTimeout: 3 * time.Second,
//		Check:          &check,
//	}
//	return newConsumerConfig
//}
//
//// NewConsumerConfig returns ShutdownConfig with @opts
//func NewConsumerConfig(opts ...ConsumerConfigOpt) *consumer.ShutdownConfig {
//	newConfig := NewDefaultConsumerConfig()
//	for _, v := range opts {
//		v(newConfig)
//	}
//	return newConfig
//}
//
//// WithConsumerAppConfig returns ConsumerConfigOpt with given @appConfig
//func WithConsumerAppConfig(appConfig *applicationConfig.ShutdownConfig) ConsumerConfigOpt {
//	return func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig {
//		config.ApplicationConfig = appConfig
//		return config
//	}
//}
//
//// WithConsumerRegistryConfig returns ConsumerConfigOpt with given @registryKey and @regConfig
//func WithConsumerRegistryConfig(registryKey string, regConfig *registry.RegistryConfig) ConsumerConfigOpt {
//	return func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig {
//		config.Registries[registryKey] = regConfig
//		return config
//	}
//}
//
//// WithConsumerReferenceConfig returns ConsumerConfigOpt with
//func WithConsumerReferenceConfig(referenceKey string, refConfig *reference.ReferenceConfig) ConsumerConfigOpt {
//	return func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig {
//		config.References[referenceKey] = refConfig
//		return config
//	}
//}
//
//// WithConsumerConnTimeout returns ConsumerConfigOpt with given consumer conn @timeout
//func WithConsumerConnTimeout(timeout time.Duration) ConsumerConfigOpt {
//	return func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig {
//		config.ConnectTimeout = timeout
//		return config
//	}
//}
//
//// WithConsumerRequestTimeout returns ConsumerConfigOpt with given consumer request @timeout
//func WithConsumerRequestTimeout(timeout time.Duration) ConsumerConfigOpt {
//	return func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig {
//		config.RequestTimeout = timeout
//		return config
//	}
//}
//
//// WithConsumerConfigCenterConfig returns ConsumerConfigOpt with given @configCenterConfig
//func WithConsumerConfigCenterConfig(configCenterConfig *center.ShutdownConfig) ConsumerConfigOpt {
//	return func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig {
//		config.ConfigCenterConfig = configCenterConfig
//		return config
//	}
//}
//
//// WithConsumerConfigCheck returns ConsumerConfigOpt with given @check flag
//func WithConsumerConfigCheck(check bool) ConsumerConfigOpt {
//	return func(config *consumer.ShutdownConfig) *consumer.ShutdownConfig {
//		*config.Check = check
//		return config
//	}
//}
//
////////////////////////////////////// reference config api
//// ReferenceConfigOpt is consumer's reference config
//type ReferenceConfigOpt func(config *reference.ReferenceConfig) *reference.ReferenceConfig
//
//// NewDefaultReferenceConfig returns empty ReferenceConfig
//func NewDefaultReferenceConfig() *reference.ReferenceConfig {
//	newReferenceConfig := reference.NewReferenceConfig("")
//	newReferenceConfig.Methods = make([]*method.MethodConfig, 0, 8)
//	newReferenceConfig.Params = make(map[string]string, 8)
//	return newReferenceConfig
//}
//
//// NewReferenceConfigByAPI returns ReferenceConfig with given @opts
//func NewReferenceConfigByAPI(opts ...ReferenceConfigOpt) *reference.ReferenceConfig {
//	newReferenceConfig := NewDefaultReferenceConfig()
//	for _, v := range opts {
//		v(newReferenceConfig)
//	}
//	return newReferenceConfig
//}
//
//// WithReferenceRegistry returns ReferenceConfigOpt with given registryKey: @registry
//func WithReferenceRegistry(registry string) ReferenceConfigOpt {
//	return func(config *reference.ReferenceConfig) *reference.ReferenceConfig {
//		config.Registry = registry
//		return config
//	}
//}
//
//// WithReferenceProtocol returns ReferenceConfigOpt with given protocolKey: @protocol
//func WithReferenceProtocol(protocol string) ReferenceConfigOpt {
//	return func(config *reference.ReferenceConfig) *reference.ReferenceConfig {
//		config.Protocol = protocol
//		return config
//	}
//}
//
//// WithReferenceInterface returns ReferenceConfigOpt with given @interfaceName
//func WithReferenceInterface(interfaceName string) ReferenceConfigOpt {
//	return func(config *reference.ReferenceConfig) *reference.ReferenceConfig {
//		config.InterfaceName = interfaceName
//		return config
//	}
//}
//
//// WithReferenceCluster returns ReferenceConfigOpt with given cluster name: @cluster
//func WithReferenceCluster(cluster string) ReferenceConfigOpt {
//	return func(config *reference.ReferenceConfig) *reference.ReferenceConfig {
//		config.Cluster = cluster
//		return config
//	}
//}
//
//// WithReferenceMethod returns ReferenceConfigOpt with given @method, @retries, and load balance: @lb
//func WithReferenceMethod(methodName, retries, lb string) ReferenceConfigOpt {
//	return func(config *reference.ReferenceConfig) *reference.ReferenceConfig {
//		config.Methods = append(config.Methods, &method.MethodConfig{
//			Name:        methodName,
//			Retries:     retries,
//			LoadBalance: lb,
//		})
//		return config
//	}
//}
//
/////////////////////////////////////// provider config api
//// ProviderConfigOpt is the
//type ProviderConfigOpt func(config *provider.ProviderConfig) *provider.ProviderConfig
//
//// NewDefaultProviderConfig returns ProviderConfig with default ShutdownConfig
//func NewDefaultProviderConfig() *provider.ProviderConfig {
//	newConsumerConfig := &provider.ProviderConfig{
//		BaseConfig: base.ShutdownConfig{
//			applicationConfig.ShutdownConfig: &applicationConfig.ShutdownConfig{
//				Name:         "dubbo",
//				Module:       "module",
//				Organization: "dubbo_org",
//				Owner:        "dubbo",
//			},
//		},
//		Services:   make(map[string]*service.ShutdownConfig),
//		Registries: make(map[string]*registry.RegistryConfig, 8),
//		Protocols:  make(map[string]*protocol.ProtocolConfig, 8),
//	}
//	return newConsumerConfig
//}
//
//// NewProviderConfig returns ProviderConfig with given @opts
//func NewProviderConfig(opts ...ProviderConfigOpt) *provider.ProviderConfig {
//	newConfig := NewDefaultProviderConfig()
//	for _, v := range opts {
//		v(newConfig)
//	}
//	return newConfig
//}
//
//// WithProviderRegistryConfig returns ProviderConfigOpt with given registry config: @regConfig
//func WithProviderRegistryConfig(regConfig *registry.RegistryConfig) ProviderConfigOpt {
//	return func(config *provider.ProviderConfig) *provider.ProviderConfig {
//		config.Registries[regConfig.Protocol] = regConfig
//		return config
//	}
//}
//
//// WithProviderAppConfig returns ProviderConfigOpt with given @appConfig
//func WithProviderAppConfig(appConfig *applicationConfig.ShutdownConfig) ProviderConfigOpt {
//	return func(config *provider.ProviderConfig) *provider.ProviderConfig {
//		config.ApplicationConfig = appConfig
//		return config
//	}
//}
//
//// WithProviderServices returns ProviderConfig with given serviceNameKey @serviceName and @serviceConfig
//func WithProviderServices(serviceName string, serviceConfig *service.ShutdownConfig) ProviderConfigOpt {
//	return func(config *provider.ProviderConfig) *provider.ProviderConfig {
//		config.Services[serviceName] = serviceConfig
//		return config
//	}
//}
//
//// WithProviderProtocol returns ProviderConfigOpt with given @protocolKey, protocolName @protocol and @port
//func WithProviderProtocol(protocolKey, protocol, port string) ProviderConfigOpt {
//	return func(config *provider.ProviderConfig) *provider.ProviderConfig {
//		config.Protocols[protocolKey] = &protocol.ProtocolConfig{
//			Name: protocol,
//			Port: port,
//		}
//		return config
//	}
//}
//
//// WithProviderRegistry returns ProviderConfigOpt with given @registryKey and registry @registryConfig
//func WithProviderRegistry(registryKey string, registryConfig *registry.RegistryConfig) ProviderConfigOpt {
//	return func(config *provider.ProviderConfig) *provider.ProviderConfig {
//		config.Registries[registryKey] = registryConfig
//		return config
//	}
//}
//
///////////////////////////////////////// service config api
//// ServiceConfigOpt is the option to init ShutdownConfig
//type ServiceConfigOpt func(config *service.ShutdownConfig) *service.ShutdownConfig
//
//// NewDefaultServiceConfig returns default ShutdownConfig
//func NewDefaultServiceConfig() *service.ShutdownConfig {
//	newServiceConfig := service.NewServiceConfig("")
//	newServiceConfig.Params = make(map[string]string)
//	newServiceConfig.Methods = make([]*method.MethodConfig, 0, 8)
//	return newServiceConfig
//}
//
//// NewServiceConfigByAPI is named as api, because there is NewServiceConfig func already declared
//// NewServiceConfigByAPI returns ShutdownConfig with given @opts
//func NewServiceConfigByAPI(opts ...ServiceConfigOpt) *service.ShutdownConfig {
//	defaultServiceConfig := NewDefaultServiceConfig()
//	for _, v := range opts {
//		v(defaultServiceConfig)
//	}
//	return defaultServiceConfig
//}
//
//// WithServiceRegistry returns ServiceConfigOpt with given registryKey @registry
//func WithServiceRegistry(registry string) ServiceConfigOpt {
//	return func(config *service.ShutdownConfig) *service.ShutdownConfig {
//		config.Registry = registry
//		return config
//	}
//}
//
//// WithServiceProtocol returns ServiceConfigOpt with given protocolKey @protocol
//func WithServiceProtocol(protocol string) ServiceConfigOpt {
//	return func(config *service.ShutdownConfig) *service.ShutdownConfig {
//		config.Protocol = protocol
//		return config
//	}
//}
//
//// WithServiceInterface returns ServiceConfigOpt with given @interfaceName
//func WithServiceInterface(interfaceName string) ServiceConfigOpt {
//	return func(config *service.ShutdownConfig) *service.ShutdownConfig {
//		config.InterfaceName = interfaceName
//		return config
//	}
//}
//
//// WithServiceLoadBalance returns ServiceConfigOpt with given load balance @lb
//func WithServiceLoadBalance(lb string) ServiceConfigOpt {
//	return func(config *service.ShutdownConfig) *service.ShutdownConfig {
//		config.Loadbalance = lb
//		return config
//	}
//}
//
//// WithServiceWarmUpTime returns ServiceConfigOpt with given @warmUp time
//func WithServiceWarmUpTime(warmUp string) ServiceConfigOpt {
//	return func(config *service.ShutdownConfig) *service.ShutdownConfig {
//		config.Warmup = warmUp
//		return config
//	}
//}
//
//// WithServiceCluster returns ServiceConfigOpt with given cluster name @cluster
//func WithServiceCluster(cluster string) ServiceConfigOpt {
//	return func(config *service.ShutdownConfig) *service.ShutdownConfig {
//		config.Cluster = cluster
//		return config
//	}
//}
//
//// WithServiceMethod returns ServiceConfigOpt with given @name, @retries and load balance @lb
//func WithServiceMethod(name, retries, lb string) ServiceConfigOpt {
//	return func(config *service.ShutdownConfig) *service.ShutdownConfig {
//		config.Methods = append(config.Methods, &method.MethodConfig{
//			Name:        name,
//			Retries:     retries,
//			LoadBalance: lb,
//		})
//		return config
//	}
//}
//
/////////////////////////////////////////// Application config api
//// ApplicationConfigOpt is option to init ShutdownConfig
//type ApplicationConfigOpt func(config *applicationConfig.ShutdownConfig) *applicationConfig.ShutdownConfig
//
//// NewDefaultApplicationConfig returns ShutdownConfig with default
//// name: dubbo.io
//// module: sample
//// organization: dubbo.io
//// owner: dubbogo
//// version: 0.0.1
//// environment dev
//func NewDefaultApplicationConfig() *applicationConfig.ShutdownConfig {
//	newAppConfig := &applicationConfig.ShutdownConfig{
//		Name:         "dubbo.io",
//		Module:       "sample",
//		Organization: "dubbo.io",
//		Owner:        "dubbogo",
//		Version:      "0.0.1",
//		Environment:  "dev",
//	}
//	return newAppConfig
//}
//
//// NewApplicationConfig is named as api, because there is NewServiceConfig func already declared
//// NewApplicationConfig returns ShutdownConfig with default applicationConfig config
//func NewApplicationConfig(opts ...ApplicationConfigOpt) *applicationConfig.ShutdownConfig {
//	defaultServiceConfig := NewDefaultApplicationConfig()
//	for _, v := range opts {
//		v(defaultServiceConfig)
//	}
//	return defaultServiceConfig
//}
//
//// WithAppName returns ApplicationConfigOpt with given @name
//func WithAppName(name string) ApplicationConfigOpt {
//	return func(config *applicationConfig.ShutdownConfig) *applicationConfig.ShutdownConfig {
//		config.Name = name
//		return config
//	}
//}
//
//// WithAppModule returns ApplicationConfigOpt with given @module
//func WithAppModule(module string) ApplicationConfigOpt {
//	return func(config *applicationConfig.ShutdownConfig) *applicationConfig.ShutdownConfig {
//		config.Module = module
//		return config
//	}
//}
//
//// WithAppOrganization returns ApplicationConfigOpt wight given organization @org
//func WithAppOrganization(org string) ApplicationConfigOpt {
//	return func(config *applicationConfig.ShutdownConfig) *applicationConfig.ShutdownConfig {
//		config.Organization = org
//		return config
//	}
//}
//
//// WithAppOwner returns ApplicationConfigOpt with given @owner
//func WithAppOwner(owner string) ApplicationConfigOpt {
//	return func(config *applicationConfig.ShutdownConfig) *applicationConfig.ShutdownConfig {
//		config.Owner = owner
//		return config
//	}
//}
//
//// WithAppVersion returns ApplicationConfigOpt with given version @version
//func WithAppVersion(version string) ApplicationConfigOpt {
//	return func(config *applicationConfig.ShutdownConfig) *applicationConfig.ShutdownConfig {
//		config.Version = version
//		return config
//	}
//}
//
//// WithAppEnvironment returns ApplicationConfigOpt with given environment @env
//func WithAppEnvironment(env string) ApplicationConfigOpt {
//	return func(config *applicationConfig.ShutdownConfig) *applicationConfig.ShutdownConfig {
//		config.Environment = env
//		return config
//	}
//}

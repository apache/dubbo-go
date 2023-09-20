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

package server

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"go.uber.org/atomic"
	"sync"
)

type ServerOptions struct {
	Application *global.ApplicationConfig
	Provider    *global.ProviderConfig
	Server      *global.ServiceConfig
	Registries  map[string]*global.RegistryConfig

	Id              string
	unexported      *atomic.Bool
	exported        *atomic.Bool
	Export          bool
	metadataType    string
	info            *ServiceInfo
	ProxyFactoryKey string
	rpcService      common.RPCService
	cacheMutex      sync.Mutex
	cacheProtocol   protocol.Protocol
	exportersLock   sync.Mutex
	exporters       []protocol.Exporter
	adaptiveService bool

	methodsCompat     []*config.MethodConfig
	applicationCompat *config.ApplicationConfig
	registriesCompat  map[string]*config.RegistryConfig
	protocolCompat    map[string]*config.ProtocolConfig
}

func defaultServerOptions() *ServerOptions {
	return &ServerOptions{
		Provider: global.DefaultProviderConfig(),
	}
}

func (srvOpts *ServerOptions) init(opts ...ServerOption) error {
	for _, opt := range opts {
		opt(srvOpts)
	}
	return nil
}

type ServerOption func(*ServerOptions)

// ---------- For user ----------

func WithRegistryIDs(registryIDs []string) ServerOption {
	return func(cfg *ServerOptions) {
		if len(registryIDs) <= 0 {
			cfg.Server.RegistryIDs = registryIDs
		}
	}
}

func WithFilter(filter string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.Filter = filter
	}
}

func WithProtocolIDs(protocolIDs []string) ServerOption {
	return func(cfg *ServerOptions) {
		if len(protocolIDs) <= 0 {
			cfg.Server.ProtocolIDs = protocolIDs
		}
	}
}

func WithTracingKey(tracingKey string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.TracingKey = tracingKey
	}
}

func WithLoadBalance(loadBalance string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.Loadbalance = loadBalance
	}
}

func WithWarmUp(warmUp string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.Warmup = warmUp
	}
}

func WithCluster(cluster string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.Cluster = cluster
	}
}

func WithGroup(group string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.Group = group
	}
}

func WithVersion(version string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.Version = version
	}
}

func WithSerialization(serialization string) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.Serialization = serialization
	}
}

func WithNotRegister(notRegister bool) ServerOption {
	return func(cfg *ServerOptions) {
		cfg.Server.NotRegister = notRegister
	}
}

// ----------From framework----------

func WithApplicationConfig(opts ...global.ApplicationOption) ServerOption {
	appCfg := new(global.ApplicationConfig)
	for _, opt := range opts {
		opt(appCfg)
	}

	return func(opts *ServerOptions) {
		opts.Application = appCfg
	}
}

func WithProviderConfig(opts ...global.ProviderOption) ServerOption {
	providerCfg := new(global.ProviderConfig)
	for _, opt := range opts {
		opt(providerCfg)
	}

	return func(opts *ServerOptions) {
		opts.Provider = providerCfg
	}
}

func WithServiceConfig(opts ...global.ServiceOption) ServerOption {
	serviceCfg := new(global.ServiceConfig)
	for _, opt := range opts {
		opt(serviceCfg)
	}

	return func(opts *ServerOptions) {
		opts.Server = serviceCfg
	}
}

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
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/config"
	aslimiter "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/creasty/defaults"
	"github.com/dubbogo/gost/log/logger"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
	"sync"
)

type ServerOptions struct {
	Provider   *global.ProviderConfig
	Registries map[string]*global.RegistryConfig
	Protocols  map[string]*global.ProtocolConfig
	Tracings   map[string]*global.TracingConfig

	providerCompat *config.ProviderConfig
}

func defaultServerOptions() *ServerOptions {
	return &ServerOptions{
		Provider: global.DefaultProviderConfig(),
	}
}

// todo(DMwangnima): think about the timing to initialize Registry, Protocol, Tracing
func (srvOpts *ServerOptions) init(opts ...ServerOption) error {
	for _, opt := range opts {
		opt(srvOpts)
	}
	if err := defaults.Set(srvOpts); err != nil {
		return err
	}

	prov := srvOpts.Provider

	prov.RegistryIDs = commonCfg.TranslateIds(prov.RegistryIDs)
	if len(prov.RegistryIDs) <= 0 {
		prov.RegistryIDs = getRegistryIds(srvOpts.Registries)
	}

	prov.ProtocolIDs = commonCfg.TranslateIds(prov.ProtocolIDs)

	if prov.TracingKey == "" && len(srvOpts.Tracings) > 0 {
		for key := range srvOpts.Tracings {
			prov.TracingKey = key
			break
		}
	}

	if err := commonCfg.Verify(prov); err != nil {
		return err
	}

	// enable adaptive service verbose
	if prov.AdaptiveServiceVerbose {
		if !prov.AdaptiveService {
			return perrors.Errorf("The adaptive service is disabled, " +
				"adaptive service verbose should be disabled either.")
		}
		logger.Infof("adaptive service verbose is enabled.")
		logger.Debugf("debug-level info could be shown.")
		aslimiter.Verbose = true
	}

	return nil
}

type ServerOption func(*ServerOptions)

// ---------- For user ----------

func WithServer_Filter(filter string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Filter = filter
	}
}

func WithServer_Register(flag bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Register = flag
	}
}

func WithServer_RegistryIDs(registryIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.RegistryIDs = registryIDs
	}
}

func WithServer_ProtocolIDs(protocolIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ProtocolIDs = protocolIDs
	}
}

func WithServer_TracingKey(tracingKey string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TracingKey = tracingKey
	}
}

func WithServer_ProxyFactory(factory string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ProxyFactory = factory
	}
}

func WithServer_FilterConf(conf interface{}) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.FilterConf = conf
	}
}

func WithServer_ConfigType(key, value string) ServerOption {
	return func(opts *ServerOptions) {
		if opts.Provider.ConfigType == nil {
			opts.Provider.ConfigType = make(map[string]string)
		}
		opts.Provider.ConfigType[key] = value
	}
}

func WithServer_AdaptiveService(flag bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveService = flag
	}
}

func WithServer_AdaptiveServiceVerbose(flag bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveServiceVerbose = flag
	}
}

func WithServer_RegistryConfig(key string, opts ...global.RegistryOption) ServerOption {
	regCfg := new(global.RegistryConfig)
	for _, opt := range opts {
		opt(regCfg)
	}

	return func(opts *ServerOptions) {
		if opts.Registries == nil {
			opts.Registries = make(map[string]*global.RegistryConfig)
		}
		opts.Registries[key] = regCfg
	}
}

func WithServer_ProtocolConfig(key string, opts ...global.ProtocolOption) ServerOption {
	proCfg := new(global.ProtocolConfig)
	for _, opt := range opts {
		opt(proCfg)
	}

	return func(opts *ServerOptions) {
		if opts.Protocols == nil {
			opts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		opts.Protocols[key] = proCfg
	}
}

func WithServer_TracingConfig(key string, opts ...global.TracingOption) ServerOption {
	traCfg := new(global.TracingConfig)
	for _, opt := range opts {
		opt(traCfg)
	}

	return func(opts *ServerOptions) {
		if opts.Tracings == nil {
			opts.Tracings = make(map[string]*global.TracingConfig)
		}
		opts.Tracings[key] = traCfg
	}
}

type ServiceOptions struct {
	Application *global.ApplicationConfig
	Provider    *global.ProviderConfig
	Service     *global.ServiceConfig
	Registries  map[string]*global.RegistryConfig
	Protocols   map[string]*global.ProtocolConfig

	Id              string
	unexported      *atomic.Bool
	exported        *atomic.Bool
	needExport      bool
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
	protocolsCompat   map[string]*config.ProtocolConfig
}

func defaultServiceOptions() *ServiceOptions {
	return &ServiceOptions{
		Service:    global.DefaultServiceConfig(),
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
		needExport: true,
	}
}

func (svcOpts *ServiceOptions) init(opts ...ServiceOption) error {
	for _, opt := range opts {
		opt(svcOpts)
	}
	if err := defaults.Set(svcOpts); err != nil {
		return err
	}

	srv := svcOpts.Service

	svcOpts.exported = atomic.NewBool(false)

	application := svcOpts.Application
	if application != nil {
		svcOpts.applicationCompat = compatApplicationConfig(application)
		if err := svcOpts.applicationCompat.Init(); err != nil {
			return err
		}
		svcOpts.metadataType = svcOpts.applicationCompat.MetadataType
		if srv.Group == "" {
			srv.Group = svcOpts.applicationCompat.Group
		}
		if srv.Version == "" {
			srv.Version = svcOpts.applicationCompat.Version
		}
	}
	svcOpts.unexported = atomic.NewBool(false)

	// initialize Registries
	if len(srv.RCRegistriesMap) == 0 {
		srv.RCRegistriesMap = svcOpts.Registries
	}
	if len(srv.RCRegistriesMap) > 0 {
		svcOpts.registriesCompat = make(map[string]*config.RegistryConfig)
		for key, reg := range srv.RCRegistriesMap {
			svcOpts.registriesCompat[key] = compatRegistryConfig(reg)
			if err := svcOpts.registriesCompat[key].Init(); err != nil {
				return err
			}
		}
	}

	// initialize Protocols
	if len(srv.RCProtocolsMap) == 0 {
		srv.RCProtocolsMap = svcOpts.Protocols
	}
	if len(srv.RCProtocolsMap) > 0 {
		svcOpts.protocolsCompat = make(map[string]*config.ProtocolConfig)
		for key, pro := range srv.RCProtocolsMap {
			svcOpts.protocolsCompat[key] = compatProtocolConfig(pro)
			if err := svcOpts.protocolsCompat[key].Init(); err != nil {
				return err
			}
		}
	}

	srv.RegistryIDs = commonCfg.TranslateIds(srv.RegistryIDs)
	if len(srv.RegistryIDs) < 0 {
		srv.RegistryIDs = svcOpts.Provider.RegistryIDs
	}

	srv.ProtocolIDs = commonCfg.TranslateIds(srv.ProtocolIDs)
	if len(srv.ProtocolIDs) <= 0 {
		srv.ProtocolIDs = svcOpts.Provider.ProtocolIDs
	}
	if len(srv.ProtocolIDs) <= 0 {
		for name := range svcOpts.Protocols {
			srv.ProtocolIDs = append(srv.ProtocolIDs, name)
		}
	}

	if srv.TracingKey == "" {
		srv.TracingKey = svcOpts.Provider.TracingKey
	}

	err := svcOpts.check()
	if err != nil {
		panic(err)
	}
	svcOpts.needExport = true
	return commonCfg.Verify(svcOpts)
}

type ServiceOption func(*ServiceOptions)

// ---------- For user ----------

func WithRegistryIDs(registryIDs []string) ServiceOption {
	return func(cfg *ServiceOptions) {
		if len(registryIDs) <= 0 {
			cfg.Service.RegistryIDs = registryIDs
		}
	}
}

func WithFilter(filter string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Filter = filter
	}
}

func WithProtocolIDs(protocolIDs []string) ServiceOption {
	return func(cfg *ServiceOptions) {
		if len(protocolIDs) <= 0 {
			cfg.Service.ProtocolIDs = protocolIDs
		}
	}
}

func WithTracingKey(tracingKey string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.TracingKey = tracingKey
	}
}

func WithLoadBalance(loadBalance string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Loadbalance = loadBalance
	}
}

func WithWarmUp(warmUp string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Warmup = warmUp
	}
}

func WithCluster(cluster string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Cluster = cluster
	}
}

func WithGroup(group string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Group = group
	}
}

func WithVersion(version string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Version = version
	}
}

func WithSerialization(serialization string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Serialization = serialization
	}
}

func WithNotRegister(flag bool) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.NotRegister = flag
	}
}

// ----------From framework----------

func WithApplicationConfig(opts ...global.ApplicationOption) ServiceOption {
	appCfg := new(global.ApplicationConfig)
	for _, opt := range opts {
		opt(appCfg)
	}

	return func(opts *ServiceOptions) {
		opts.Application = appCfg
	}
}

func WithProviderConfig(opts ...global.ProviderOption) ServiceOption {
	providerCfg := new(global.ProviderConfig)
	for _, opt := range opts {
		opt(providerCfg)
	}

	return func(opts *ServiceOptions) {
		opts.Provider = providerCfg
	}
}

func WithServiceConfig(opts ...global.ServiceOption) ServiceOption {
	serviceCfg := new(global.ServiceConfig)
	for _, opt := range opts {
		opt(serviceCfg)
	}

	return func(opts *ServiceOptions) {
		opts.Service = serviceCfg
	}
}

func WithRegistryConfig(key string, opts ...global.RegistryOption) ServiceOption {
	regCfg := new(global.RegistryConfig)
	for _, opt := range opts {
		opt(regCfg)
	}

	return func(opts *ServiceOptions) {
		if opts.Registries == nil {
			opts.Registries = make(map[string]*global.RegistryConfig)
		}
		opts.Registries[key] = regCfg
	}
}

func WithProtocolConfig(key string, opts ...global.ProtocolOption) ServiceOption {
	proCfg := new(global.ProtocolConfig)
	for _, opt := range opts {
		opt(proCfg)
	}

	return func(opts *ServiceOptions) {
		if opts.Protocols == nil {
			opts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		opts.Protocols[key] = proCfg
	}
}

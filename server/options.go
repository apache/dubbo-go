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
	"strconv"
	"sync"
	"time"
)

import (
	"github.com/creasty/defaults"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	aslimiter "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type ServerOptions struct {
	Provider    *global.ProviderConfig
	Application *global.ApplicationConfig
	Registries  map[string]*global.RegistryConfig
	Protocols   map[string]*global.ProtocolConfig
	Tracings    map[string]*global.TracingConfig
	Shutdown    *global.ShutdownConfig

	providerCompat *config.ProviderConfig
}

func defaultServerOptions() *ServerOptions {
	return &ServerOptions{
		Application: global.DefaultApplicationConfig(),
		Provider:    global.DefaultProviderConfig(),
		Shutdown:    global.DefaultShutdownConfig(),
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

	// init graceful_shutdown
	graceful_shutdown.Init(graceful_shutdown.WithShutdown_Config(srvOpts.Shutdown))

	return nil
}

type ServerOption func(*ServerOptions)

// ---------- For user ----------

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithServer_Filter(filter string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithServer_RegistryIDs(registryIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.RegistryIDs = registryIDs
	}
}

func WithServer_Registry(key string, opts ...registry.Option) ServerOption {
	regOpts := registry.DefaultOptions()
	for _, opt := range opts {
		opt(regOpts)
	}

	return func(srvOpts *ServerOptions) {
		if srvOpts.Registries == nil {
			srvOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		srvOpts.Registries[key] = regOpts.Registry
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithServer_ProtocolIDs(protocolIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ProtocolIDs = protocolIDs
	}
}

func WithServer_Protocol(key string, opts ...protocol.Option) ServerOption {
	proOpts := protocol.DefaultOptions()
	for _, opt := range opts {
		opt(proOpts)
	}

	return func(srvOpts *ServerOptions) {
		if srvOpts.Protocols == nil {
			srvOpts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		srvOpts.Protocols[key] = proOpts.Protocol
	}
}

func WithServer_TracingKey(tracingKey string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TracingKey = tracingKey
	}
}

// todo(DMwangnima): this configuration would be used by filter/hystrix
// think about a more ideal way to configure
func WithServer_FilterConf(conf interface{}) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.FilterConf = conf
	}
}

func WithServer_AdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveService = true
	}
}

func WithServer_AdaptiveServiceVerbose() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveServiceVerbose = true
	}
}

// ========== For framework ==========
// These functions should not be invoked by users

func SetServer_Application(application *global.ApplicationConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Application = application
	}
}

func SetServer_Registries(regs map[string]*global.RegistryConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Registries = regs
	}
}

func SetServer_Protocols(pros map[string]*global.ProtocolConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Protocols = pros
	}
}

func SetServer_Tracings(tras map[string]*global.TracingConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Tracings = tras
	}
}

func SetServer_Shutdown(shutdown *global.ShutdownConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Shutdown = shutdown
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
		Service:     global.DefaultServiceConfig(),
		Application: global.DefaultApplicationConfig(),
		unexported:  atomic.NewBool(false),
		exported:    atomic.NewBool(false),
		needExport:  true,
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
		// todo(DMwangnima): make this clearer
		// this statement is responsible for setting rootConfig.Application
		// since many modules would retrieve this information directly.
		config.GetRootConfig().Application = svcOpts.applicationCompat
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
	if len(srv.RegistryIDs) <= 0 {
		srv.RegistryIDs = svcOpts.Provider.RegistryIDs
	}
	if srv.RegistryIDs == nil || len(srv.RegistryIDs) <= 0 {
		srv.NotRegister = true
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

// todo(DMwangnima): think about a more ideal configuration style
func WithRegistryIDs(registryIDs []string) ServiceOption {
	return func(cfg *ServiceOptions) {
		if len(registryIDs) <= 0 {
			cfg.Service.RegistryIDs = registryIDs
		}
	}
}

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithFilter(filter string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
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

// ========== LoadBalance Strategy ==========

func WithLoadBalanceConsistentHashing() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithLoadBalanceLeastActive() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithLoadBalanceRandom() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithLoadBalanceRoundRobin() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithLoadBalanceP2C() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithLoadBalanceXDSRingHash() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// warmUp is in seconds
func WithWarmUp(warmUp time.Duration) ServiceOption {
	return func(opts *ServiceOptions) {
		warmUpSec := int(warmUp / time.Second)
		opts.Service.Warmup = strconv.Itoa(warmUpSec)
	}
}

// ========== Cluster Strategy ==========

func WithClusterAvailable() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyAvailable
	}
}

func WithClusterBroadcast() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithClusterFailBack() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailback
	}
}

func WithClusterFailFast() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailfast
	}
}

func WithClusterFailOver() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailover
	}
}

func WithClusterFailSafe() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithClusterForking() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyForking
	}
}

func WithClusterZoneAware() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithClusterAdaptiveService() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyAdaptiveService
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

func WithJSON() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Serialization = constant.JSONSerialization
	}
}

// WithToken should be used with WithFilter("token")
func WithToken(token string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Token = token
	}
}

func WithNotRegister() ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.NotRegister = true
	}
}

// ----------For framework----------
// These functions should not be invoked by users

func SetApplication(application *global.ApplicationConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Application = application
	}
}

func SetProvider(provider *global.ProviderConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Provider = provider
	}
}

func SetService(service *global.ServiceConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service = service
	}
}

func SetRegistries(regs map[string]*global.RegistryConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Registries = regs
	}
}

func SetProtocols(pros map[string]*global.ProtocolConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Protocols = pros
	}
}

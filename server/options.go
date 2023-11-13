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
	"reflect"
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
	"dubbo.apache.org/dubbo-go/v3/common/dubboutil"
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
	graceful_shutdown.Init(graceful_shutdown.SetShutdown_Config(srvOpts.Shutdown))

	return nil
}

type ServerOption func(*ServerOptions)

// ---------- For user ----------

// ========== LoadBalance Strategy ==========

func WithServerLoadBalanceConsistentHashing() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithServerLoadBalanceLeastActive() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithServerLoadBalanceRandom() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithServerLoadBalanceRoundRobin() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithServerLoadBalanceP2C() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithServerLoadBalanceXDSRingHash() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// warmUp is in seconds
func WithServerWarmUp(warmUp time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		warmUpSec := int(warmUp / time.Second)
		opts.Provider.Warmup = strconv.Itoa(warmUpSec)
	}
}

// ========== Cluster Strategy ==========

func WithServerClusterAvailable() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAvailable
	}
}

func WithServerClusterBroadcast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithServerClusterFailBack() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailback
	}
}

func WithServerClusterFailFast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailfast
	}
}

func WithServerClusterFailOver() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailover
	}
}

func WithServerClusterFailSafe() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithServerClusterForking() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyForking
	}
}

func WithServerClusterZoneAware() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithServerClusterAdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAdaptiveService
	}
}

func WithServerGroup(group string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Group = group
	}
}

func WithServerVersion(version string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Version = version
	}
}

func WithServerJSON() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Serialization = constant.JSONSerialization
	}
}

// WithToken should be used with WithFilter("token")
func WithServerToken(token string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Token = token
	}
}

func WithServerNotRegister() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.NotRegister = true
	}
}

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithServerFilter(filter string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithServerRegistryIDs(registryIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.RegistryIDs = registryIDs
	}
}

func WithServerRegistry(opts ...registry.Option) ServerOption {
	regOpts := registry.NewOptions(opts...)

	return func(srvOpts *ServerOptions) {
		if srvOpts.Registries == nil {
			srvOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		srvOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithServerProtocolIDs(protocolIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ProtocolIDs = protocolIDs
	}
}

func WithServerProtocol(opts ...protocol.Option) ServerOption {
	proOpts := protocol.NewOptions(opts...)

	return func(srvOpts *ServerOptions) {
		if srvOpts.Protocols == nil {
			srvOpts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		srvOpts.Protocols[proOpts.ID] = proOpts.Protocol
	}
}

// todo(DMwangnima): this configuration would be used by filter/hystrix
// think about a more ideal way to configure
func WithServerFilterConf(conf interface{}) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.FilterConf = conf
	}
}

func WithServerAdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveService = true
	}
}

func WithServerAdaptiveServiceVerbose() ServerOption {
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

func (svcOpts *ServiceOptions) init(srv *Server, opts ...ServiceOption) error {
	for _, opt := range opts {
		opt(svcOpts)
	}
	if err := defaults.Set(svcOpts); err != nil {
		return err
	}

	svc := svcOpts.Service
	dubboutil.CopyFields(reflect.ValueOf(srv.cfg.Provider).Elem(), reflect.ValueOf(svc).Elem())

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
		if svc.Group == "" {
			svc.Group = svcOpts.applicationCompat.Group
		}
		if svc.Version == "" {
			svc.Version = svcOpts.applicationCompat.Version
		}
	}
	svcOpts.unexported = atomic.NewBool(false)

	// initialize Registries
	if len(svc.RCRegistriesMap) == 0 {
		svc.RCRegistriesMap = svcOpts.Registries
	}
	if len(svc.RCRegistriesMap) > 0 {
		svcOpts.registriesCompat = make(map[string]*config.RegistryConfig)
		for key, reg := range svc.RCRegistriesMap {
			svcOpts.registriesCompat[key] = compatRegistryConfig(reg)
			if err := svcOpts.registriesCompat[key].Init(); err != nil {
				return err
			}
		}
	}

	// initialize Protocols
	if len(svc.RCProtocolsMap) == 0 {
		svc.RCProtocolsMap = svcOpts.Protocols
	}
	if len(svc.RCProtocolsMap) > 0 {
		svcOpts.protocolsCompat = make(map[string]*config.ProtocolConfig)
		for key, pro := range svc.RCProtocolsMap {
			svcOpts.protocolsCompat[key] = compatProtocolConfig(pro)
			if err := svcOpts.protocolsCompat[key].Init(); err != nil {
				return err
			}
		}
	}

	svc.RegistryIDs = commonCfg.TranslateIds(svc.RegistryIDs)
	if len(svc.RegistryIDs) <= 0 {
		svc.RegistryIDs = svcOpts.Provider.RegistryIDs
	}
	if svc.RegistryIDs == nil || len(svc.RegistryIDs) <= 0 {
		svc.NotRegister = true
	}

	svc.ProtocolIDs = commonCfg.TranslateIds(svc.ProtocolIDs)
	if len(svc.ProtocolIDs) <= 0 {
		svc.ProtocolIDs = svcOpts.Provider.ProtocolIDs
	}
	if len(svc.ProtocolIDs) <= 0 {
		for name := range svcOpts.Protocols {
			svc.ProtocolIDs = append(svc.ProtocolIDs, name)
		}
	}

	if svc.TracingKey == "" {
		svc.TracingKey = svcOpts.Provider.TracingKey
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

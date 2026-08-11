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
	aslimiter "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/internal"
	"dubbo.apache.org/dubbo-go/v3/metrics/probe"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/tls"
)

type ServerOptions struct {
	Provider    *global.ProviderConfig
	Application *global.ApplicationConfig
	Registries  map[string]*global.RegistryConfig
	Protocols   map[string]*global.ProtocolConfig
	Shutdown    *global.ShutdownConfig
	Metrics     *global.MetricsConfig
	Otel        *global.OtelConfig
	TLS         *global.TLSConfig
}

func defaultServerOptions() *ServerOptions {
	return &ServerOptions{
		Application: global.DefaultApplicationConfig(),
		Provider:    global.DefaultProviderConfig(),
		Shutdown:    global.DefaultShutdownConfig(),
		Metrics:     global.DefaultMetricsConfig(),
		Otel:        global.DefaultOtelConfig(),
		TLS:         global.DefaultTLSConfig(),
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

	providerConf := srvOpts.Provider

	providerConf.RegistryIDs = commonCfg.TranslateIds(providerConf.RegistryIDs)
	if len(providerConf.RegistryIDs) <= 0 {
		providerConf.RegistryIDs = getRegistryIds(srvOpts.Registries)
	}
	if err := internal.ValidateRegistryIDs(providerConf.RegistryIDs, srvOpts.Registries); err != nil {
		return err
	}

	providerConf.ProtocolIDs = commonCfg.TranslateIds(providerConf.ProtocolIDs)

	if err := commonCfg.Verify(providerConf); err != nil {
		return err
	}

	// enable adaptive service verbose
	if providerConf.AdaptiveServiceVerbose {
		if !providerConf.AdaptiveService {
			return perrors.Errorf("The adaptive service is disabled, " +
				"adaptive service verbose should be disabled either.")
		}
		logger.Info("[Server] adaptive service verbose is enabled")
		logger.Debug("[Server] debug-level info could be shown")
		aslimiter.Verbose = true
	}

	// init graceful_shutdown
	graceful_shutdown.Init(graceful_shutdown.SetShutdownConfig(srvOpts.Shutdown))

	// init probe
	if probeCfg := probe.BuildProbeConfig(srvOpts.Metrics.Probe); probeCfg != nil {
		probe.Init(probeCfg)
	}

	return nil
}

type ServerOption func(*ServerOptions)

// ---------- For user ----------

// ========== LoadBalance Strategy ==========

// WithServerLoadBalanceConsistentHashing sets ServerOptions.Provider.Loadbalance to consistent hashing.
func WithServerLoadBalanceConsistentHashing() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithServerLoadBalanceLeastActive sets ServerOptions.Provider.Loadbalance to least active.
func WithServerLoadBalanceLeastActive() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithServerLoadBalanceRandom sets ServerOptions.Provider.Loadbalance to random.
func WithServerLoadBalanceRandom() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithServerLoadBalanceRoundRobin sets ServerOptions.Provider.Loadbalance to round robin.
func WithServerLoadBalanceRoundRobin() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithServerLoadBalanceP2C sets ServerOptions.Provider.Loadbalance to power of two choices.
func WithServerLoadBalanceP2C() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithServerLoadBalance sets ServerOptions.Provider.Loadbalance to a custom strategy name.
func WithServerLoadBalance(lb string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = lb
	}
}

// WithServerWarmUp sets ServerOptions.Provider.Warmup to the whole-second value of warmUp.
func WithServerWarmUp(warmUp time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		warmUpSec := int(warmUp / time.Second)
		opts.Provider.Warmup = strconv.Itoa(warmUpSec)
	}
}

// ========== Cluster Strategy ==========

// WithServerClusterAvailable sets ServerOptions.Provider.Cluster to available.
func WithServerClusterAvailable() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAvailable
	}
}

// WithServerClusterBroadcast sets ServerOptions.Provider.Cluster to broadcast.
func WithServerClusterBroadcast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithServerClusterFailBack sets ServerOptions.Provider.Cluster to failback.
func WithServerClusterFailBack() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailback
	}
}

// WithServerClusterFailFast sets ServerOptions.Provider.Cluster to fail-fast.
func WithServerClusterFailFast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailfast
	}
}

// WithServerClusterFailOver sets ServerOptions.Provider.Cluster to failover.
// Pair it with WithServerRetries to control the retry count.
func WithServerClusterFailOver() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailover
	}
}

// WithServerClusterFailSafe sets ServerOptions.Provider.Cluster to fail-safe.
func WithServerClusterFailSafe() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithServerClusterForking sets ServerOptions.Provider.Cluster to forking.
func WithServerClusterForking() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyForking
	}
}

// WithServerClusterZoneAware sets ServerOptions.Provider.Cluster to zone-aware.
func WithServerClusterZoneAware() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithServerClusterAdaptiveService sets ServerOptions.Provider.Cluster to adaptive-service.
func WithServerClusterAdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithServerCluster sets ServerOptions.Provider.Cluster to a custom strategy name.
func WithServerCluster(cluster string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = cluster
	}
}

// WithServerGroup sets ServerOptions.Provider.Group as the default service group.
// A service-level WithGroup can override it.
func WithServerGroup(group string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Group = group
	}
}

// WithServerVersion sets ServerOptions.Provider.Version as the default service version.
// A service-level WithVersion can override it.
func WithServerVersion(version string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Version = version
	}
}

// WithServerJSON sets ServerOptions.Provider.Serialization to JSON.
// Clients must select a compatible serialization.
func WithServerJSON() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Serialization = constant.JSONSerialization
	}
}

// WithServerToken sets ServerOptions.Provider.Token for provider authentication.
// Pair it with WithServerFilter("token") to enable token validation.
func WithServerToken(token string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Token = token
	}
}

// WithServerNotRegister sets ServerOptions.Provider.NotRegister to skip service registration by default.
func WithServerNotRegister() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.NotRegister = true
	}
}

// WithServerWarmup sets ServerOptions.Provider.Warmup to warmupDuration.String().
// Use WithServerWarmUp when the configuration requires a whole-second numeric value.
func WithServerWarmup(warmupDuration time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Warmup = warmupDuration.String()
	}
}

// WithServerRetries sets ServerOptions.Provider.Retries as the default retry count.
// It is commonly paired with WithServerClusterFailOver.
func WithServerRetries(retries int) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Retries = strconv.Itoa(retries)
	}
}

// WithServerSerialization sets ServerOptions.Provider.Serialization.
// Clients must select a compatible serialization.
func WithServerSerialization(ser string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Serialization = ser
	}
}

// WithServerAccesslog sets ServerOptions.Provider.AccessLog for provider access logging.
func WithServerAccesslog(accesslog string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AccessLog = accesslog
	}
}

// WithServerTpsLimiter sets ServerOptions.Provider.TpsLimiter.
// Pair it with the WithServerTpsLimitRate, strategy, and rejected-handler options as needed.
func WithServerTpsLimiter(limiter string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimiter = limiter
	}
}

// WithServerTpsLimitRate sets ServerOptions.Provider.TpsLimitRate.
// Pair it with WithServerTpsLimiter to enable TPS limiting.
func WithServerTpsLimitRate(rate int) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimitRate = strconv.Itoa(rate)
	}
}

// WithServerTpsLimitStrategy sets ServerOptions.Provider.TpsLimitStrategy.
// Pair it with WithServerTpsLimiter to enable TPS limiting.
func WithServerTpsLimitStrategy(strategy string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimitStrategy = strategy
	}
}

// WithServerTpsLimitRejectedHandler sets ServerOptions.Provider.TpsLimitRejectedHandler.
// Pair it with WithServerTpsLimiter to handle rejected requests.
func WithServerTpsLimitRejectedHandler(rejHandler string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimitRejectedHandler = rejHandler
	}
}

// WithServerExecuteLimit sets ServerOptions.Provider.ExecuteLimit.
// Pair it with WithServerExecuteLimitRejectedHandler when custom rejection handling is required.
func WithServerExecuteLimit(exeLimit string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ExecuteLimit = exeLimit
	}
}

// WithServerExecuteLimitRejectedHandler sets ServerOptions.Provider.ExecuteLimitRejectedHandler.
// Pair it with WithServerExecuteLimit.
func WithServerExecuteLimitRejectedHandler(exeRejHandler string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ExecuteLimitRejectedHandler = exeRejHandler
	}
}

// WithServerAuth sets ServerOptions.Provider.Auth for provider authentication metadata.
func WithServerAuth(auth string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Auth = auth
	}
}

// WithServerParamSign sets ServerOptions.Provider.ParamSign for parameter signing.
func WithServerParamSign(paramSign string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ParamSign = paramSign
	}
}

// WithServerTag sets ServerOptions.Provider.Tag for tag-based routing.
func WithServerTag(tag string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Tag = tag
	}
}

// WithServerParam adds or replaces one entry in ServerOptions.Provider.Params.
func WithServerParam(k, v string) ServerOption {
	return func(opts *ServerOptions) {
		if opts.Provider.Params == nil {
			opts.Provider.Params = make(map[string]string)
		}
		opts.Provider.Params[k] = v
	}
}

// WithServerFilter sets ServerOptions.Provider.Filter as the default provider filter chain.
// Pair it with options such as WithServerToken that require a matching filter.
func WithServerFilter(filter string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Filter = filter
	}
}

// WithServerRegistryIDs sets ServerOptions.Provider.RegistryIDs.
// Pair it with WithServerRegistry using matching registry IDs.
func WithServerRegistryIDs(registryIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.RegistryIDs = registryIDs
	}
}

// WithServerRegistry builds a registry configuration and adds it to ServerOptions.Registries.
// Use registry.WithID and select the same ID with WithServerRegistryIDs when needed.
func WithServerRegistry(opts ...registry.Option) ServerOption {
	regOpts := registry.NewOptions(opts...)

	return func(srvOpts *ServerOptions) {
		if srvOpts.Registries == nil {
			srvOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		srvOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// WithServerProtocolIDs sets ServerOptions.Provider.ProtocolIDs.
// Pair it with WithServerProtocol using matching protocol IDs.
func WithServerProtocolIDs(protocolIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ProtocolIDs = protocolIDs
	}
}

// WithServerProtocol builds a protocol configuration and adds it to ServerOptions.Protocols.
// Use protocol.WithID and select the same ID with WithServerProtocolIDs when multiple protocols exist.
func WithServerProtocol(opts ...protocol.ServerOption) ServerOption {
	proOpts := protocol.NewServerOptions(opts...)

	return func(srvOpts *ServerOptions) {
		if srvOpts.Protocols == nil {
			srvOpts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		srvOpts.Protocols[proOpts.ID] = proOpts.Protocol
	}
}

// WithServerAdaptiveService sets ServerOptions.Provider.AdaptiveService to enable adaptive services.
func WithServerAdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveService = true
	}
}

// WithServerAdaptiveServiceVerbose sets ServerOptions.Provider.AdaptiveServiceVerbose.
// Pair it with WithServerAdaptiveService to enable verbose adaptive-service output.
func WithServerAdaptiveServiceVerbose() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveServiceVerbose = true
	}
}

// WithServerTLSOption applies tls.Option values to ServerOptions.TLS.
// Configure compatible client settings with client.WithClientTLSOption.
func WithServerTLSOption(opts ...tls.Option) ServerOption {
	tlsOpts := tls.NewOptions(opts...)

	return func(srvOpts *ServerOptions) {
		if srvOpts.TLS == nil {
			srvOpts.TLS = new(global.TLSConfig)
		}
		srvOpts.TLS = tlsOpts.TLSConf
	}
}

// ========== For framework ==========
// These functions should not be invoked by users

// SetServerApplication assigns framework-loaded application configuration to ServerOptions.Application.
func SetServerApplication(application *global.ApplicationConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Application = application
	}
}

// SetServerRegistries replaces ServerOptions.Registries with framework-loaded configuration.
// User code should prefer WithServerRegistry and WithServerRegistryIDs.
func SetServerRegistries(regs map[string]*global.RegistryConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Registries = regs
	}
}

// SetServerProtocols replaces ServerOptions.Protocols with framework-loaded configuration.
// User code should prefer WithServerProtocol and WithServerProtocolIDs.
func SetServerProtocols(pros map[string]*global.ProtocolConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Protocols = pros
	}
}

// SetServerShutdown assigns framework-loaded shutdown configuration to ServerOptions.Shutdown.
func SetServerShutdown(shutdown *global.ShutdownConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Shutdown = shutdown
	}
}

// SetServerMetrics assigns framework-loaded metrics configuration to ServerOptions.Metrics.
func SetServerMetrics(metrics *global.MetricsConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Metrics = metrics
	}
}

// SetServerOtel assigns framework-loaded OpenTelemetry configuration to ServerOptions.Otel.
func SetServerOtel(otel *global.OtelConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Otel = otel
	}
}

// SetServerTLS assigns framework-loaded TLS configuration to ServerOptions.TLS.
// User code should prefer WithServerTLSOption.
func SetServerTLS(tls *global.TLSConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.TLS = tls
	}
}

// SetServerProvider assigns framework-loaded provider defaults to ServerOptions.Provider.
func SetServerProvider(provider *global.ProviderConfig) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider = provider
	}
}

// FIXME: ServiceOptions contains ServerOptions?
// Not ServerOptions contains ServiceOptions?
// we need to find a way to fix it.
type ServiceOptions struct {
	Application *global.ApplicationConfig
	Provider    *global.ProviderConfig
	Service     *global.ServiceConfig
	Registries  map[string]*global.RegistryConfig
	Protocols   map[string]*global.ProtocolConfig

	srvOpts *ServerOptions

	Id              string
	unexported      *atomic.Bool
	exported        *atomic.Bool
	needExport      bool
	metadataType    string
	info            *common.ServiceInfo
	ProxyFactoryKey string
	rpcService      common.RPCService
	cacheMutex      sync.Mutex
	cacheProtocol   base.Protocol
	exportersLock   sync.Mutex
	exporters       []base.Exporter
	adaptiveService bool

	// for triple non-IDL mode
	// consider put here or global.ServiceConfig
	// string for url
	//
	// Deprecated: this implementation will be removed in the next release.
	// The IDLMode switch will no longer be supported by dubbo-go.
	IDLMode string

	// openapi group for documentation
	openapiGroup string
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

	svcOpts.srvOpts = srv.cfg
	svc := svcOpts.Service

	dubboutil.CopyFields(reflect.ValueOf(srv.cfg.Provider).Elem(), reflect.ValueOf(svc).Elem())

	svcOpts.exported = atomic.NewBool(false)

	application := svcOpts.Application
	if application != nil {
		if svc.Group == "" {
			svc.Group = application.Group
		}
		if svc.Version == "" {
			svc.Version = application.Version
		}
	}
	svcOpts.unexported = atomic.NewBool(false)

	// initialize Registries
	if len(svc.RCRegistriesMap) == 0 {
		svc.RCRegistriesMap = svcOpts.Registries
	}

	// initialize Protocols
	if len(svc.RCProtocolsMap) == 0 {
		svc.RCProtocolsMap = svcOpts.Protocols
	}

	svc.RegistryIDs = commonCfg.TranslateIds(svc.RegistryIDs)
	if len(svc.RegistryIDs) <= 0 {
		svc.RegistryIDs = svcOpts.Provider.RegistryIDs
	}
	if len(svc.RegistryIDs) <= 0 {
		svc.NotRegister = true
	} else if err := internal.ValidateRegistryIDs(svc.RegistryIDs, svc.RCRegistriesMap); err != nil {
		return err
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
	for _, method := range svc.Methods {
		if err := internal.ValidateMethodConfig(method); err != nil {
			return err
		}
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

// WithInterface sets ServiceOptions.Service.Interface for the service being exposed.
//
// As a functional option, it is passed to a service registration function
// (e.g., RegisterGreetServiceHandler) to configure the service's properties.
//
// The `interfaceName` acts as the unique identifier for this service in the registry.
// Clients (consumers) must use this exact name to discover and invoke the service.
//
// Usage:
//
//	err := greet.RegisterGreetServiceHandler(
//	    srv,
//	    &GreetTripleServer{},
//	    server.WithInterface("com.your.company.GreetService"),
//	)
func WithInterface(interfaceName string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Interface = interfaceName
	}
}

// WithRegistryIDs sets ServiceOptions.Service.RegistryIDs for this service.
// Pair it with WithRegistry using matching registry IDs.
func WithRegistryIDs(registryIDs []string) ServiceOption {
	return func(cfg *ServiceOptions) {
		if len(registryIDs) > 0 {
			cfg.Service.RegistryIDs = registryIDs
		}
	}
}

// WithFilter sets ServiceOptions.Service.Filter for this service.
// Pair it with options such as WithToken that require a matching filter.
func WithFilter(filter string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Filter = filter
	}
}

// WithProtocolIDs sets ServiceOptions.Service.ProtocolIDs for this service.
// Pair it with WithProtocol using matching protocol IDs.
func WithProtocolIDs(protocolIDs []string) ServiceOption {
	return func(cfg *ServiceOptions) {
		if len(protocolIDs) > 0 {
			cfg.Service.ProtocolIDs = protocolIDs
		}
	}
}

// ========== LoadBalance Strategy ==========

// WithLoadBalanceConsistentHashing sets ServiceOptions.Service.Loadbalance to consistent hashing.
func WithLoadBalanceConsistentHashing() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithLoadBalanceLeastActive sets ServiceOptions.Service.Loadbalance to least active.
func WithLoadBalanceLeastActive() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithLoadBalanceRandom sets ServiceOptions.Service.Loadbalance to random.
func WithLoadBalanceRandom() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithLoadBalanceRoundRobin sets ServiceOptions.Service.Loadbalance to round robin.
func WithLoadBalanceRoundRobin() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithLoadBalanceP2C sets ServiceOptions.Service.Loadbalance to power of two choices.
func WithLoadBalanceP2C() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithLoadBalance sets ServiceOptions.Service.Loadbalance to a custom strategy name.
func WithLoadBalance(lb string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = lb
	}
}

// WithWarmUp sets ServiceOptions.Service.Warmup to the whole-second value of warmUp.
func WithWarmUp(warmUp time.Duration) ServiceOption {
	return func(opts *ServiceOptions) {
		warmUpSec := int(warmUp / time.Second)
		opts.Service.Warmup = strconv.Itoa(warmUpSec)
	}
}

// ========== Cluster Strategy ==========

// WithClusterAvailable sets ServiceOptions.Service.Cluster to available.
func WithClusterAvailable() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyAvailable
	}
}

// WithClusterBroadcast sets ServiceOptions.Service.Cluster to broadcast.
func WithClusterBroadcast() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithClusterFailBack sets ServiceOptions.Service.Cluster to failback.
func WithClusterFailBack() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailback
	}
}

// WithClusterFailFast sets ServiceOptions.Service.Cluster to fail-fast.
func WithClusterFailFast() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailfast
	}
}

// WithClusterFailOver sets ServiceOptions.Service.Cluster to failover.
// Pair it with WithRetries to control the retry count.
func WithClusterFailOver() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailover
	}
}

// WithClusterFailSafe sets ServiceOptions.Service.Cluster to fail-safe.
func WithClusterFailSafe() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithClusterForking sets ServiceOptions.Service.Cluster to forking.
func WithClusterForking() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyForking
	}
}

// WithClusterZoneAware sets ServiceOptions.Service.Cluster to zone-aware.
func WithClusterZoneAware() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithClusterAdaptiveService sets ServiceOptions.Service.Cluster to adaptive-service.
func WithClusterAdaptiveService() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithCluster sets ServiceOptions.Service.Cluster to a custom strategy name.
func WithCluster(cluster string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = cluster
	}
}

// WithGroup sets ServiceOptions.Service.Group for provider discovery.
// Clients must use the same group with client.WithGroup or client.WithClientGroup.
func WithGroup(group string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Group = group
	}
}

// WithVersion sets ServiceOptions.Service.Version for provider discovery.
// Clients must use the same version with client.WithVersion or client.WithClientVersion.
func WithVersion(version string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Version = version
	}
}

// WithJSON sets ServiceOptions.Service.Serialization to JSON.
// Clients must select JSON or another compatible serialization.
func WithJSON() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Serialization = constant.JSONSerialization
	}
}

// WithToken sets ServiceOptions.Service.Token for service authentication.
// Pair it with WithFilter("token") to enable token validation.
func WithToken(token string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Token = token
	}
}

// WithNotRegister sets ServiceOptions.Service.NotRegister to skip registry publication.
func WithNotRegister() ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.NotRegister = true
	}
}

// WithWarmup sets ServiceOptions.Service.Warmup to warmupDuration.String().
// Use WithWarmUp when the configuration requires a whole-second numeric value.
func WithWarmup(warmupDuration time.Duration) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Warmup = warmupDuration.String()
	}
}

// WithRetries sets ServiceOptions.Service.Retries for this service.
// It is commonly paired with WithClusterFailOver.
func WithRetries(retries int) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Retries = strconv.Itoa(retries)
	}
}

// WithSerialization sets ServiceOptions.Service.Serialization.
// Clients must select a compatible serialization.
func WithSerialization(ser string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Serialization = ser
	}
}

// WithAccesslog sets ServiceOptions.Service.AccessLog for service access logging.
func WithAccesslog(accesslog string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.AccessLog = accesslog
	}
}

// WithTpsLimiter sets ServiceOptions.Service.TpsLimiter.
// Pair it with the TPS rate, strategy, and rejected-handler options as needed.
func WithTpsLimiter(limiter string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimiter = limiter
	}
}

// WithTpsLimitRate sets ServiceOptions.Service.TpsLimitRate.
// Pair it with WithTpsLimiter to enable TPS limiting.
func WithTpsLimitRate(rate int) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimitRate = strconv.Itoa(rate)
	}
}

// WithTpsLimitStrategy sets ServiceOptions.Service.TpsLimitStrategy.
// Pair it with WithTpsLimiter to enable TPS limiting.
func WithTpsLimitStrategy(strategy string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimitStrategy = strategy
	}
}

// WithTpsLimitRejectedHandler sets ServiceOptions.Service.TpsLimitRejectedHandler.
// Pair it with WithTpsLimiter to handle rejected requests.
func WithTpsLimitRejectedHandler(rejHandler string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimitRejectedHandler = rejHandler
	}
}

// WithExecuteLimit sets ServiceOptions.Service.ExecuteLimit.
// Pair it with WithExecuteLimitRejectedHandler when custom rejection handling is required.
func WithExecuteLimit(exeLimit string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.ExecuteLimit = exeLimit
	}
}

// WithExecuteLimitRejectedHandler sets ServiceOptions.Service.ExecuteLimitRejectedHandler.
// Pair it with WithExecuteLimit.
func WithExecuteLimitRejectedHandler(exeRejHandler string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.ExecuteLimitRejectedHandler = exeRejHandler
	}
}

// WithAuth sets ServiceOptions.Service.Auth for service authentication metadata.
func WithAuth(auth string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Auth = auth
	}
}

// WithParamSign sets ServiceOptions.Service.ParamSign for parameter signing.
func WithParamSign(paramSign string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.ParamSign = paramSign
	}
}

// WithTag sets ServiceOptions.Service.Tag for tag-based routing.
func WithTag(tag string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Tag = tag
	}
}

// WithProtocol builds a protocol configuration and adds it to ServiceOptions.Protocols.
// Use protocol.WithID and select the same ID with WithProtocolIDs when multiple protocols exist.
func WithProtocol(opts ...protocol.ServerOption) ServiceOption {
	proOpts := protocol.NewServerOptions(opts...)

	return func(opts *ServiceOptions) {
		if opts.Protocols == nil {
			opts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		opts.Protocols[proOpts.ID] = proOpts.Protocol
	}
}

// WithRegistry builds a registry configuration and adds it to ServiceOptions.Registries.
// Use registry.WithID and select the same ID with WithRegistryIDs when multiple registries exist.
func WithRegistry(opts ...registry.Option) ServiceOption {
	regOpts := registry.NewOptions(opts...)

	return func(opts *ServiceOptions) {
		if opts.Registries == nil {
			opts.Registries = make(map[string]*global.RegistryConfig)
		}
		opts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// WithMethod appends method to ServiceOptions.Service.Methods when it is non-nil.
// Use it with service-level options to override configuration for a specific method.
func WithMethod(method *global.MethodConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		if method == nil {
			return
		}
		if opts.Service.Methods == nil {
			opts.Service.Methods = make([]*global.MethodConfig, 0)
		}
		opts.Service.Methods = append(opts.Service.Methods, method)
	}
}

// WithParam adds or replaces one entry in ServiceOptions.Service.Params.
func WithParam(k, v string) ServiceOption {
	return func(opts *ServiceOptions) {
		if opts.Service.Params == nil {
			opts.Service.Params = make(map[string]string)
		}
		opts.Service.Params[k] = v
	}
}

// WithOpenAPIGroup sets ServiceOptions.openapiGroup used to group generated OpenAPI operations.
func WithOpenAPIGroup(group string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.openapiGroup = group
	}
}

// WithIDLMode sets ServiceOptions.IDLMode for legacy services.
//
// Deprecated: this option will be removed in the next version. The IDL mode
// switch is no longer supported by dubbo-go.
func WithIDLMode(IDLMode string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.IDLMode = IDLMode
	}
}

// ----------For framework----------
// These functions should not be invoked by users

// SetApplication assigns framework-loaded application configuration to ServiceOptions.Application.
func SetApplication(application *global.ApplicationConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Application = application
	}
}

// SetProvider assigns framework-loaded provider configuration to ServiceOptions.Provider.
func SetProvider(provider *global.ProviderConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Provider = provider
	}
}

// SetService assigns framework-loaded service configuration to ServiceOptions.Service.
func SetService(service *global.ServiceConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service = service
	}
}

// SetRegistries replaces ServiceOptions.Registries with framework-loaded configuration.
// User code should prefer WithRegistry and WithRegistryIDs.
func SetRegistries(regs map[string]*global.RegistryConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Registries = regs
	}
}

// SetProtocols replaces ServiceOptions.Protocols with framework-loaded configuration.
// User code should prefer WithProtocol and WithProtocolIDs.
func SetProtocols(pros map[string]*global.ProtocolConfig) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Protocols = pros
	}
}

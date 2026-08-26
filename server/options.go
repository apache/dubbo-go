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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/dubboutil"
	"dubbo.apache.org/dubbo-go/v3/common/dubboutil/atomic"
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

// WithServerLoadBalanceConsistentHashing advertises consistent hashing as the default
// consumer load balancer for every service. Calls with the same configured arguments tend
// to reach the same provider while the provider set is stable. Use it when most services need
// cache or session affinity; service-level load-balancing options can override it.
func WithServerLoadBalanceConsistentHashing() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithServerLoadBalanceLeastActive advertises least-active load balancing as the default for
// consumers, favoring providers with fewer in-flight requests and using weight to break ties.
// Use it when request duration varies and busy instances should receive less work.
func WithServerLoadBalanceLeastActive() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithServerLoadBalanceRandom advertises weighted-random provider selection as the default
// consumer load-balancing policy. It is a low-overhead general default for statistically even
// traffic across services.
func WithServerLoadBalanceRandom() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithServerLoadBalanceRoundRobin advertises smooth weighted round-robin provider selection
// as the default consumer load-balancing policy. Use it when requests have similar cost and
// predictable per-instance traffic shares are desirable.
func WithServerLoadBalanceRoundRobin() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithServerLoadBalanceP2C advertises P2C as the default consumer load balancer. Consumers
// sample two providers and favor the one reporting more remaining capacity. Use it together
// with WithServerAdaptiveService for capacity-aware routing.
func WithServerLoadBalanceP2C() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithServerLoadBalance advertises a registered load-balancing extension as the default for
// consumers. Use it for domain-specific provider placement; a service-level option overrides it.
func WithServerLoadBalance(lb string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = lb
	}
}

// WithServerWarmUp gradually increases newly started providers' effective weight over the
// supplied duration, reducing traffic while caches and other resources warm up. Durations
// shorter than one second are truncated. Use it when cold instances cannot safely receive full
// traffic immediately; WithServerWarmup preserves sub-second duration strings.
func WithServerWarmUp(warmUp time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		warmUpSec := int(warmUp / time.Second)
		opts.Provider.Warmup = strconv.Itoa(warmUpSec)
	}
}

// ========== Cluster Strategy ==========

// WithServerClusterAvailable tells consumers to invoke the first available provider without
// load balancing or retries by default. Use it only when any healthy provider is sufficient
// and balanced traffic is not required.
func WithServerClusterAvailable() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAvailable
	}
}

// WithServerClusterBroadcast tells consumers to invoke every provider sequentially by
// default and report an error if any provider fails. Use it for operations such as cache
// invalidation that intentionally run on every instance.
func WithServerClusterBroadcast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithServerClusterFailBack tells consumers to hide an initial failure and retry the call in
// the background with exponential backoff. Use it for best-effort notifications where eventual
// delivery matters more than returning the initial error.
func WithServerClusterFailBack() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailback
	}
}

// WithServerClusterFailFast tells consumers to invoke once and return the error without
// retrying another provider. Use it for non-idempotent operations where duplicate execution
// would be more harmful than an immediate failure.
func WithServerClusterFailFast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailfast
	}
}

// WithServerClusterFailOver tells consumers to retry non-business failures on reselected
// providers. Use it for idempotent calls that should survive one unavailable instance;
// WithServerRetries controls the additional attempts after the first call.
func WithServerClusterFailOver() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailover
	}
}

// WithServerClusterFailSafe tells consumers to log and suppress invocation failures,
// returning an empty result. Use it only for optional best-effort work, such as audit events,
// because callers cannot distinguish a suppressed failure from an empty success.
func WithServerClusterFailSafe() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithServerClusterForking tells consumers to invoke multiple providers concurrently and
// return the first completed result. Use it for idempotent, latency-sensitive reads and accept
// the duplicate work and extra provider load it creates.
func WithServerClusterForking() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyForking
	}
}

// WithServerClusterZoneAware tells consumers using multiple registries to prefer an explicitly
// preferred registry, then the request's zone, before falling back by registry weight. Use it
// for multi-region services that should keep traffic local while retaining fallback.
func WithServerClusterZoneAware() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithServerClusterAdaptiveService advertises adaptive remaining-capacity routing to consumers.
// Consumers must also use P2C and providers must publish adaptive capacity metrics. Use it for
// services whose effective instance capacity changes significantly under load.
func WithServerClusterAdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithServerCluster advertises a registered cluster extension as the default consumer fault
// handling policy. Use it for an application-specific failure policy; a service-level cluster
// option overrides it.
func WithServerCluster(cluster string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = cluster
	}
}

// WithServerGroup publishes services in the supplied group by default, allowing multiple
// logical implementations of one interface to coexist. Use groups for environments, tenants,
// or alternate implementations; consumers must request the same group.
func WithServerGroup(group string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Group = group
	}
}

// WithServerVersion publishes services under the supplied version by default. Consumers with
// another version cannot discover them. Use it during incompatible API migrations; a
// service-level WithVersion overrides this value.
func WithServerVersion(version string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Version = version
	}
}

// WithServerJSON uses JSON as the default wire serialization for exported services. Consumers
// and the selected protocol must support JSON or requests cannot be decoded. Use it for
// interoperability when readable payloads matter more than compact binary encoding.
func WithServerJSON() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Serialization = constant.JSONSerialization
	}
}

// WithServerToken requires consumers to present the same service token when the token provider
// filter is active. Use it for simple shared-secret protection and pair it with
// WithServerFilter("token") or a chain containing that filter.
func WithServerToken(token string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Token = token
	}
}

// WithServerNotRegister prevents services from being published to registries by default while
// still allowing the server to listen. Use it for local tests or private fixed endpoints; such
// services must be reached by a direct URL.
func WithServerNotRegister() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.NotRegister = true
	}
}

// WithServerWarmup gradually increases newly started providers' effective load-balancing
// weight over the supplied duration. Use it for cold-start protection when caches or connection
// pools need time to fill. It preserves values such as "500ms" in the provider URL.
func WithServerWarmup(warmupDuration time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Warmup = warmupDuration.String()
	}
}

// WithServerRetries advertises how many additional attempts retry-capable consumers may make
// after the initial call. Zero means one total attempt. Use retries only for idempotent services;
// service-level settings take precedence.
func WithServerRetries(retries int) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Retries = strconv.Itoa(retries)
	}
}

// WithServerSerialization selects the default wire serialization by extension name. Consumers
// and the selected protocol must support the same serialization. Use it when both sides install
// the same non-default serialization extension.
func WithServerSerialization(ser string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Serialization = ser
	}
}

// WithServerAccesslog enables provider access logging by default. A file path writes access
// records there; "true" or "default" sends them to the application logger. Logging is
// asynchronous and records may be dropped if its channel is full. Use it for request auditing
// or troubleshooting, accounting for payload visibility and storage cost.
func WithServerAccesslog(accesslog string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AccessLog = accesslog
	}
}

// WithServerTpsLimiter enables the named TPS limiter for services by default. An empty name
// disables TPS limiting; an unregistered name causes service validation to panic. Use it to
// protect provider capacity from bursts, together with rate, strategy, and rejection settings.
func WithServerTpsLimiter(limiter string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimiter = limiter
	}
}

// WithServerTpsLimitRate sets the default maximum request rate enforced by the selected TPS
// limiter. Use it to express the sustainable throughput of services. It has no effect until
// WithServerTpsLimiter selects a limiter.
func WithServerTpsLimitRate(rate int) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimitRate = strconv.Itoa(rate)
	}
}

// WithServerTpsLimitStrategy selects the registered rate-limiting strategy used by the default
// TPS limiter. Use it to choose how bursts are measured, such as a fixed or sliding window;
// an unregistered name causes service validation to panic.
func WithServerTpsLimitStrategy(strategy string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimitStrategy = strategy
	}
}

// WithServerTpsLimitRejectedHandler selects the handler invoked when the default TPS limit is
// exceeded. Use a custom handler to return a domain-specific error or fallback result;
// an unregistered name causes service validation to panic.
func WithServerTpsLimitRejectedHandler(rejHandler string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.TpsLimitRejectedHandler = rejHandler
	}
}

// WithServerExecuteLimit caps concurrent in-flight provider invocations by default. The value
// must be an integer string; a negative value disables the cap, while an invalid value returns
// an empty result without invoking the service. Use it when concurrency, rather than request
// rate, is the scarce resource, such as a bounded database connection pool.
func WithServerExecuteLimit(exeLimit string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ExecuteLimit = exeLimit
	}
}

// WithServerExecuteLimitRejectedHandler selects the registered handler for calls rejected after
// the default in-flight limit is reached. Use it to return a specific overload response. If
// lookup fails, the call proceeds after a warning.
func WithServerExecuteLimitRejectedHandler(exeRejHandler string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ExecuteLimitRejectedHandler = exeRejHandler
	}
}

// WithServerAuth enables AK/SK request-signature verification for services by default when set
// to "true" and the provider filter chain contains "auth". Missing or invalid signatures are
// rejected before service execution. Use it when providers must authenticate calling applications;
// configure access-key storage and signing on both provider and consumer.
func WithServerAuth(auth string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Auth = auth
	}
}

// WithServerParamSign includes request parameters in AK/SK signature verification by default
// when set to "true". Use it when signatures must detect parameter tampering; both sides must
// canonicalize the same values. It requires authentication and the "auth" filter.
func WithServerParamSign(paramSign string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ParamSign = paramSign
	}
}

// WithServerTag publishes services with the supplied routing tag by default, allowing tagged
// consumers to target this provider group. Use tags for canary, tenant, or hardware-specific
// pools without changing the service interface.
func WithServerTag(tag string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Tag = tag
	}
}

// WithServerParam publishes one custom provider URL parameter for filters, routers, protocols,
// or extensions. Use it to configure an extension not covered by a typed option. A later call
// with the same key replaces the earlier value.
func WithServerParam(k, v string) ServerOption {
	return func(opts *ServerOptions) {
		if opts.Provider.Params == nil {
			opts.Provider.Params = make(map[string]string)
		}
		opts.Provider.Params[k] = v
	}
}

// WithServerFilter selects the comma-separated provider filter chain applied to incoming calls
// by default, in execution order. Use it for shared middleware such as authentication, metrics,
// or custom validation. A service-level WithFilter replaces this chain.
//
// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithServerFilter(filter string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Filter = filter
	}
}

// WithServerRegistryIDs limits service publication to the named registries by default. Each ID
// must match a registry added with WithServerRegistry or server initialization fails. Use it to
// publish all services to selected environments or regions when several registries exist.
//
// todo(DMwangnima): think about a more ideal configuration style
func WithServerRegistryIDs(registryIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.RegistryIDs = registryIDs
	}
}

// WithServerRegistry makes a registry available for publishing services. Give each registry a
// distinct registry.WithID and use WithServerRegistryIDs to publish only to a subset. Configure
// shared registries here instead of repeating WithRegistry for every service.
func WithServerRegistry(opts ...registry.Option) ServerOption {
	regOpts := registry.NewOptions(opts...)

	return func(srvOpts *ServerOptions) {
		if srvOpts.Registries == nil {
			srvOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		srvOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// WithServerProtocolIDs limits service export to the named server protocols by default. Each ID
// must match a protocol added with WithServerProtocol. Use it when the server listens on several
// endpoints but most services should be exposed through only a selected subset.
//
// todo(DMwangnima): think about a more ideal configuration style
func WithServerProtocolIDs(protocolIDs []string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.ProtocolIDs = protocolIDs
	}
}

// WithServerProtocol configures a protocol endpoint on which services may be exported. Give
// each endpoint a distinct protocol.WithID when serving multiple protocols or ports. Configure
// shared listeners here, then choose them globally or per service with protocol IDs.
//
// For example, this exposes services on a named Triple endpoint:
//
//	server.NewServer(
//		server.WithServerProtocol(
//			protocol.WithTriple(),
//			protocol.WithID("triple"),
//			protocol.WithPort(20000),
//		),
//		server.WithServerProtocolIDs([]string{"triple"}),
//	)
func WithServerProtocol(opts ...protocol.ServerOption) ServerOption {
	proOpts := protocol.NewServerOptions(opts...)

	return func(srvOpts *ServerOptions) {
		if srvOpts.Protocols == nil {
			srvOpts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		srvOpts.Protocols[proOpts.ID] = proOpts.Protocol
	}
}

// WithServerAdaptiveService enables provider-side capacity measurement and publishes the
// remaining-capacity metrics required by adaptive-service consumers. Use it with consumer-side
// adaptive cluster and P2C options when static provider weights do not reflect current load.
func WithServerAdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveService = true
	}
}

// WithServerAdaptiveServiceVerbose enables detailed adaptive limiter diagnostics. Server
// initialization fails unless WithServerAdaptiveService is also enabled. Use it while tuning or
// diagnosing adaptive limits; verbose output may be too noisy for normal production operation.
func WithServerAdaptiveServiceVerbose() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.AdaptiveServiceVerbose = true
	}
}

// WithServerTLSOption configures credentials and peer verification for encrypted server
// connections. Use it for transport encryption and, when configured, mutual authentication.
// Clients must use compatible trust and certificate settings.
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

// WithInterface publishes this service under the supplied discovery and routing identifier.
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

// WithRegistryIDs publishes this service only to the named registries. Each ID must match a
// registry added with WithRegistry or inherited from the server, otherwise registration fails.
// Use it when one service belongs in a different environment or region from the server default.
//
// todo(DMwangnima): think about a more ideal configuration style
func WithRegistryIDs(registryIDs []string) ServiceOption {
	return func(cfg *ServiceOptions) {
		if len(registryIDs) > 0 {
			cfg.Service.RegistryIDs = registryIDs
		}
	}
}

// WithFilter selects the comma-separated provider filter chain applied to incoming calls for
// this service, in execution order. Use it to add service-specific middleware such as "auth"
// or a custom validator. It replaces the server-level default filter chain.
//
// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithFilter(filter string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Filter = filter
	}
}

// WithProtocolIDs exports this service only through the named protocol endpoints. Each ID must
// match a protocol added with WithProtocol or inherited from the server. Use it when this
// service should expose only Triple, Dubbo, or a dedicated listener.
//
// todo(DMwangnima): think about a more ideal configuration style
func WithProtocolIDs(protocolIDs []string) ServiceOption {
	return func(cfg *ServiceOptions) {
		if len(protocolIDs) > 0 {
			cfg.Service.ProtocolIDs = protocolIDs
		}
	}
}

// ========== LoadBalance Strategy ==========

// WithLoadBalanceConsistentHashing tells consumers of this service to route calls with the same
// configured arguments to the same provider while the provider set is stable. Use it for
// per-user caches or session affinity; membership changes can remap some keys.
func WithLoadBalanceConsistentHashing() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithLoadBalanceLeastActive tells consumers to favor providers with fewer in-flight requests,
// using warm-up-adjusted weight when active counts are equal. Use it when this service has
// uneven request durations and busy instances should receive less new work.
func WithLoadBalanceLeastActive() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithLoadBalanceRandom tells consumers to choose providers randomly in proportion to their
// effective weight. Use it as a low-overhead general choice for statistically even traffic.
func WithLoadBalanceRandom() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithLoadBalanceRoundRobin tells consumers to distribute calls using smooth weighted
// round-robin selection. Use it when calls have similar cost and predictable instance shares
// are desirable.
func WithLoadBalanceRoundRobin() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithLoadBalanceP2C tells consumers to sample two providers and favor the one reporting more
// remaining capacity. Use it with adaptive-service metrics when instance capacity changes
// dynamically under load.
func WithLoadBalanceP2C() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithLoadBalance advertises a registered load-balancing extension to consumers of this
// service. Use it for a domain-specific placement rule; it overrides the server-level default.
func WithLoadBalance(lb string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Loadbalance = lb
	}
}

// WithWarmUp gradually increases this provider's effective weight over the supplied duration,
// reducing traffic after startup. Use it when this service needs to fill caches or pools before
// receiving full traffic. Durations shorter than one second are truncated.
func WithWarmUp(warmUp time.Duration) ServiceOption {
	return func(opts *ServiceOptions) {
		warmUpSec := int(warmUp / time.Second)
		opts.Service.Warmup = strconv.Itoa(warmUpSec)
	}
}

// ========== Cluster Strategy ==========

// WithClusterAvailable tells consumers to invoke the first available provider without load
// balancing or retries. Use it only when any healthy provider is sufficient and even traffic
// distribution is not required.
func WithClusterAvailable() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyAvailable
	}
}

// WithClusterBroadcast tells consumers to invoke every provider sequentially and report an
// error if any provider fails. Use it for operations such as invalidating a cache on every
// instance; the operation should tolerate repeated calls.
func WithClusterBroadcast() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithClusterFailBack tells consumers to hide an initial failure and retry in the background
// with exponential backoff, which suits eventual-delivery notifications.
func WithClusterFailBack() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailback
	}
}

// WithClusterFailFast tells consumers to invoke once and return the error without retrying
// another provider. Use it for non-idempotent operations where duplicate execution would be
// more harmful than an immediate failure.
func WithClusterFailFast() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailfast
	}
}

// WithClusterFailOver tells consumers to retry non-business failures on reselected providers.
// Use it for idempotent operations that should survive one unavailable instance. WithRetries
// controls the additional attempts after the first call.
func WithClusterFailOver() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailover
	}
}

// WithClusterFailSafe tells consumers to log and suppress invocation failures, returning an
// empty result. Use it only for optional best-effort work because callers cannot distinguish a
// suppressed failure from an empty success.
func WithClusterFailSafe() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithClusterForking tells consumers to invoke multiple providers concurrently and return the
// first completed result. Use it for idempotent, latency-sensitive reads and accept the
// duplicate work and extra provider load.
func WithClusterForking() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyForking
	}
}

// WithClusterZoneAware tells consumers using multiple registries to prefer an explicitly
// preferred registry, then the request's zone, before falling back by registry weight. Use it
// for multi-region services that should keep traffic local while retaining fallback.
func WithClusterZoneAware() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithClusterAdaptiveService tells consumers to route using remaining-capacity metrics. It
// requires P2C load balancing and providers with adaptive-service metrics enabled. Use it when
// this service's instance capacity varies significantly at runtime.
func WithClusterAdaptiveService() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithCluster advertises a registered cluster extension as the consumer fault-handling policy
// for this service. Use it for a domain-specific failure policy; it overrides the server default.
func WithCluster(cluster string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Cluster = cluster
	}
}

// WithGroup publishes this service in the supplied group, allowing multiple implementations
// of one interface to coexist. Use groups for environments, tenants, or alternate implementations;
// consumers requesting another group cannot discover it.
func WithGroup(group string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Group = group
	}
}

// WithVersion publishes this service under the supplied version. Consumers requesting another
// version cannot discover it even when the interface and group match. Use it during incompatible
// API migrations so old and new providers can run concurrently.
func WithVersion(version string) ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.Version = version
	}
}

// WithJSON encodes this service's request and response payloads with JSON. Consumers and the
// selected protocol must support JSON or requests cannot be decoded. Use it for interoperability
// when readable payloads matter more than compact binary encoding.
func WithJSON() ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Serialization = constant.JSONSerialization
	}
}

// WithToken requires consumers to present the same service token when the token provider filter
// is active. Use it for simple shared-secret protection and pair it with WithFilter("token") or
// a chain containing that filter.
func WithToken(token string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Token = token
	}
}

// WithNotRegister keeps this service out of all registries while still exporting it on its
// protocol endpoint. Use it for tests, internal health services, or fixed private endpoints;
// consumers must use a direct URL to reach it.
func WithNotRegister() ServiceOption {
	return func(cfg *ServiceOptions) {
		cfg.Service.NotRegister = true
	}
}

// WithWarmup gradually increases this provider's effective load-balancing weight over the
// supplied duration. Use it for cold-start protection while caches or pools initialize. It
// preserves values such as "500ms" in the provider URL.
func WithWarmup(warmupDuration time.Duration) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Warmup = warmupDuration.String()
	}
}

// WithRetries advertises how many additional attempts retry-capable consumers may make after
// the initial call. Zero means one total attempt. Use retries only for idempotent service methods.
func WithRetries(retries int) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Retries = strconv.Itoa(retries)
	}
}

// WithSerialization selects this service's wire serialization by extension name. Consumers
// and the selected protocol must support the same serialization. Use it when both sides install
// the same non-default serialization extension.
func WithSerialization(ser string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Serialization = ser
	}
}

// WithAccesslog enables access logging for this service. A file path writes records there;
// "true" or "default" sends them to the application logger. Logging is asynchronous and
// records may be dropped if its channel is full. Use it for auditing or troubleshooting while
// accounting for payload visibility and storage cost.
func WithAccesslog(accesslog string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.AccessLog = accesslog
	}
}

// WithTpsLimiter enables the named requests-per-second limiter for this service. An empty name
// disables TPS limiting; an unregistered name causes service validation to panic. Use it to
// protect this provider from request bursts, together with rate and rejection settings. For
// example, use WithTpsLimiter("default") and WithTpsLimitRate(100) to allow 100 requests per
// configured limiter interval before invoking the default rejection handler.
func WithTpsLimiter(limiter string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimiter = limiter
	}
}

// WithTpsLimitRate sets the maximum request rate enforced by this service's selected TPS
// limiter. Set it to the service's sustainable throughput; it has no effect until WithTpsLimiter
// selects a limiter.
func WithTpsLimitRate(rate int) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimitRate = strconv.Itoa(rate)
	}
}

// WithTpsLimitStrategy selects the registered rate-limiting strategy used for this service.
// Use it to choose how bursts are measured, such as a fixed or sliding window. An unregistered
// name causes service validation to panic.
func WithTpsLimitStrategy(strategy string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimitStrategy = strategy
	}
}

// WithTpsLimitRejectedHandler selects the handler invoked when this service exceeds its TPS
// limit. Use a custom handler to return a domain-specific overload error or fallback result;
// an unregistered name causes service validation to panic.
func WithTpsLimitRejectedHandler(rejHandler string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.TpsLimitRejectedHandler = rejHandler
	}
}

// WithExecuteLimit caps concurrent in-flight invocations of this service. The value must be an
// integer string; a negative value disables the cap, while an invalid value returns an empty
// result without invoking the service. Use it when concurrency is bounded by resources such as
// database connections. For example, WithExecuteLimit("32") allows at most 32 simultaneous
// invocations; method-level execution limits can override it.
func WithExecuteLimit(exeLimit string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.ExecuteLimit = exeLimit
	}
}

// WithExecuteLimitRejectedHandler selects the registered handler for calls rejected after this
// service's in-flight limit is reached. Use it to return a specific overload response. If lookup
// fails, the call proceeds after a warning.
func WithExecuteLimitRejectedHandler(exeRejHandler string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.ExecuteLimitRejectedHandler = exeRejHandler
	}
}

// WithAuth enables AK/SK request-signature verification when set to "true" and the provider
// filter chain contains "auth". Use it when this service must authenticate calling applications;
// missing or invalid signatures are rejected before execution. Enable both parts with
// WithFilter("auth") and WithAuth("true"), then configure compatible access-key storage and
// consumer signing.
func WithAuth(auth string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Auth = auth
	}
}

// WithParamSign includes request parameters in AK/SK signature verification when set to
// "true". Use it when the signature must also detect parameter tampering. It only has an effect
// when WithAuth and the "auth" provider filter are enabled on both sides.
func WithParamSign(paramSign string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.ParamSign = paramSign
	}
}

// WithTag publishes this service instance with the supplied routing tag, allowing tagged
// consumers to select it. Use tags for canary, tenant, or hardware-specific pools without
// changing the interface, group, or version.
func WithTag(tag string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.Service.Tag = tag
	}
}

// WithProtocol configures a protocol endpoint available to this service. Give each endpoint a
// distinct protocol.WithID and use WithProtocolIDs when the service should use only a subset.
// Use it for a listener needed only by this service; shared listeners belong on NewServer.
func WithProtocol(opts ...protocol.ServerOption) ServiceOption {
	proOpts := protocol.NewServerOptions(opts...)

	return func(opts *ServiceOptions) {
		if opts.Protocols == nil {
			opts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		opts.Protocols[proOpts.ID] = proOpts.Protocol
	}
}

// WithRegistry makes a registry available for publishing this service. Give each registry a
// distinct registry.WithID and use WithRegistryIDs to publish only to a subset. Use it for a
// service-specific registry; shared registries are normally configured on NewServer.
func WithRegistry(opts ...registry.Option) ServiceOption {
	regOpts := registry.NewOptions(opts...)

	return func(opts *ServiceOptions) {
		if opts.Registries == nil {
			opts.Registries = make(map[string]*global.RegistryConfig)
		}
		opts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// WithMethod adds method-specific behavior such as timeout, retries, or execution limits.
// Method settings take precedence over the corresponding service defaults. Use it when one
// method is slower, non-idempotent, or has a different capacity limit from the rest.
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

// WithParam publishes one custom service URL parameter for filters, routers, protocols, or
// extensions. A later call with the same key replaces the earlier value.
func WithParam(k, v string) ServiceOption {
	return func(opts *ServiceOptions) {
		if opts.Service.Params == nil {
			opts.Service.Params = make(map[string]string)
		}
		opts.Service.Params[k] = v
	}
}

// WithOpenAPIGroup places this service's generated operations in the supplied OpenAPI group,
// allowing related services to be presented together in generated API documentation. Use the
// same group for APIs that should appear as one logical section to documentation consumers.
func WithOpenAPIGroup(group string) ServiceOption {
	return func(opts *ServiceOptions) {
		opts.openapiGroup = group
	}
}

// WithIDLMode sets ServiceOptions.IDLMode for legacy services.
//
// TODO: remove when config package is removed
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

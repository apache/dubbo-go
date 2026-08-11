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

package client

import (
	"net/http"
	"strconv"
	"time"
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/internal"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/proxy"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/tls"
)

type ReferenceOptions struct {
	Reference   *global.ReferenceConfig
	Consumer    *global.ConsumerConfig
	Application *global.ApplicationConfig
	Shutdown    *global.ShutdownConfig
	Metrics     *global.MetricsConfig
	Otel        *global.OtelConfig
	TLS         *global.TLSConfig
	Protocols   map[string]*global.ProtocolConfig
	Registries  map[string]*global.RegistryConfig
	Routers     []*global.RouterConfig

	pxy          *proxy.Proxy
	id           string
	invoker      base.Invoker
	urls         []*common.URL
	metaDataType string
	info         *ClientInfo
}

func defaultReferenceOptions() *ReferenceOptions {
	return &ReferenceOptions{
		Reference:   global.DefaultReferenceConfig(),
		Application: global.DefaultApplicationConfig(),
		Shutdown:    global.DefaultShutdownConfig(),
		Metrics:     global.DefaultMetricsConfig(),
		Otel:        global.DefaultOtelConfig(),
		TLS:         global.DefaultTLSConfig(),
		Protocols:   make(map[string]*global.ProtocolConfig),
		Registries:  global.DefaultRegistriesConfig(),
	}
}

func (refOpts *ReferenceOptions) init(opts ...ReferenceOption) error {
	for _, opt := range opts {
		opt(refOpts)
	}
	if err := defaults.Set(refOpts); err != nil {
		return err
	}

	refConf := refOpts.Reference

	app := refOpts.Application
	if app != nil {
		refOpts.metaDataType = app.MetadataType
		if refConf.Group == "" {
			refConf.Group = app.Group
		}
		if refConf.Version == "" {
			refConf.Version = app.Version
		}
	}

	// init method
	if refConf.MethodsConfig == nil {
		refConf.MethodsConfig = make([]*global.MethodConfig, 0)
	} else {
		for _, method := range refConf.MethodsConfig {
			if err := internal.ValidateMethodConfig(method); err != nil {
				return err
			}
		}
	}

	// init cluster
	if refConf.Cluster == "" {
		refConf.Cluster = constant.ClusterKeyFailover
	}

	// init registries
	if len(refOpts.Registries) > 0 {
		regs := refOpts.Registries
		if len(refConf.RegistryIDs) <= 0 {
			refConf.RegistryIDs = make([]string, 0, len(regs))
			for key := range regs {
				refConf.RegistryIDs = append(refConf.RegistryIDs, key)
			}
		}
		refConf.RegistryIDs = commonCfg.TranslateIds(refConf.RegistryIDs)
		if err := internal.ValidateRegistryIDs(refConf.RegistryIDs, regs); err != nil {
			return err
		}
	}

	// init protocol
	if refConf.Protocol == "" {
		refConf.Protocol = constant.TriProtocol
		if refOpts.Consumer != nil && refOpts.Consumer.Protocol != "" {
			refConf.Protocol = refOpts.Consumer.Protocol
		}
	}

	// init serialization
	if refConf.Serialization == "" {
		refConf.Serialization = constant.ProtobufSerialization
	}

	// validate generic type, fail fast on unknown value instead of
	// silently falling back to the Map generalizer at runtime
	if err := internal.ValidateGenericType(refConf.Generic); err != nil {
		return err
	}

	return commonCfg.Verify(refOpts)
}

type ReferenceOption func(*ReferenceOptions)

// ---------- For user ----------

// WithCheck sets ReferenceOptions.Reference.Check to true for this service reference.
func WithCheck() ReferenceOption {
	return func(opts *ReferenceOptions) {
		check := true
		opts.Reference.Check = &check
	}
}

// WithURL sets ReferenceOptions.Reference.URL for direct service invocation.
// It bypasses registry discovery for this reference when a direct URL is supplied.
func WithURL(url string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.URL = url
	}
}

// WithFilter sets ReferenceOptions.Reference.Filter for this service reference.
func WithFilter(filter string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Filter = filter
	}
}

// WithInterface sets ReferenceOptions.Reference.InterfaceName for the service reference.
//
// As a functional option, it is passed to a client constructor
// (e.g., NewGreetService) to configure which remote service to connect to.
//
// The interfaceName is a crucial identifier for service discovery and routing,
// and it must exactly match the name registered by the service provider.
//
// Usage:
//
//	svc, err := greet.NewGreetService(
//	    cli,
//	    client.WithInterface("com.your.company.GreetService"),
//	)
func WithInterface(interfaceName string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.InterfaceName = interfaceName
	}
}

// WithRegistryIDs sets ReferenceOptions.Reference.RegistryIDs for this reference.
// Pair it with WithRegistry or client-level registry configuration using matching IDs.
func WithRegistryIDs(registryIDs ...string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(registryIDs) > 0 {
			opts.Reference.RegistryIDs = registryIDs
		}
	}
}

// WithRegistry builds a registry configuration and adds it to ReferenceOptions.Registries.
// Use registry.WithID and select the same ID with WithRegistryIDs when multiple registries exist.
func WithRegistry(opts ...registry.Option) ReferenceOption {
	regOpts := registry.NewOptions(opts...)

	return func(refOpts *ReferenceOptions) {
		if refOpts.Registries == nil {
			refOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		refOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// ========== Cluster Strategy ==========

// WithClusterAvailable sets ReferenceOptions.Reference.Cluster to the available strategy.
func WithClusterAvailable() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAvailable
	}
}

// WithClusterBroadcast sets ReferenceOptions.Reference.Cluster to the broadcast strategy.
func WithClusterBroadcast() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithClusterFailBack sets ReferenceOptions.Reference.Cluster to the failback strategy.
func WithClusterFailBack() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailback
	}
}

// WithClusterFailFast sets ReferenceOptions.Reference.Cluster to the fail-fast strategy.
func WithClusterFailFast() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailfast
	}
}

// WithClusterFailOver sets ReferenceOptions.Reference.Cluster to the failover strategy.
// Pair it with WithRetries to control the retry count.
func WithClusterFailOver() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailover
	}
}

// WithClusterFailSafe sets ReferenceOptions.Reference.Cluster to the fail-safe strategy.
func WithClusterFailSafe() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithClusterForking sets ReferenceOptions.Reference.Cluster to the forking strategy.
func WithClusterForking() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyForking
	}
}

// WithClusterZoneAware sets ReferenceOptions.Reference.Cluster to the zone-aware strategy.
func WithClusterZoneAware() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithClusterAdaptiveService sets ReferenceOptions.Reference.Cluster to the adaptive-service strategy.
func WithClusterAdaptiveService() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithCluster sets ReferenceOptions.Reference.Cluster to a custom cluster strategy name.
func WithCluster(cluster string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = cluster
	}
}

// ========== LoadBalance Strategy ==========

// WithLoadBalanceConsistentHashing sets ReferenceOptions.Reference.Loadbalance to consistent hashing.
func WithLoadBalanceConsistentHashing() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithLoadBalanceLeastActive sets ReferenceOptions.Reference.Loadbalance to least active.
func WithLoadBalanceLeastActive() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithLoadBalanceRandom sets ReferenceOptions.Reference.Loadbalance to random.
func WithLoadBalanceRandom() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithLoadBalanceRoundRobin sets ReferenceOptions.Reference.Loadbalance to round robin.
func WithLoadBalanceRoundRobin() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithLoadBalanceP2C sets ReferenceOptions.Reference.Loadbalance to power of two choices.
func WithLoadBalanceP2C() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithLoadBalance sets ReferenceOptions.Reference.Loadbalance to a custom strategy name.
func WithLoadBalance(lb string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = lb
	}
}

// WithRetries sets ReferenceOptions.Reference.Retries for this service reference.
// It is commonly paired with WithClusterFailOver.
func WithRetries(retries int) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Retries = strconv.Itoa(retries)
	}
}

// WithGroup sets ReferenceOptions.Reference.Group used to identify the provider group.
// The value must match the server service group.
func WithGroup(group string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Group = group
	}
}

// WithVersion sets ReferenceOptions.Reference.Version used to identify the service version.
// The value must match the server service version.
func WithVersion(version string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Version = version
	}
}

// WithSerializationJSON sets ReferenceOptions.Reference.Serialization to JSON.
// The selected serialization must be supported by the provider protocol.
func WithSerializationJSON() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Serialization = constant.JSONSerialization
	}
}

// WithSerialization sets ReferenceOptions.Reference.Serialization for this reference.
// The selected serialization must be supported by the provider protocol.
func WithSerialization(serialization string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Serialization = serialization
	}
}

// WithProvidedBy sets ReferenceOptions.Reference.ProvidedBy for provider selection metadata.
func WithProvidedBy(providedBy string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ProvidedBy = providedBy
	}
}

// WithAsync sets ReferenceOptions.Reference.Async to enable asynchronous invocation.
func WithAsync() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Async = true
	}
}

// WithParams replaces ReferenceOptions.Reference.Params when params is non-empty.
// Use WithParam to add or override a single parameter.
func WithParams(params map[string]string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(params) <= 0 {
			return
		}
		opts.Reference.Params = params
	}
}

// WithGeneric sets ReferenceOptions.Reference.Generic to the default generic invocation mode.
// Use WithGenericType to select a specific generalization format.
func WithGeneric() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Generic = "true"
	}
}

// WithGenericType sets ReferenceOptions.Reference.Generic, which decides how
// business objects are generalized into a generic structure.
//
// Valid values: "true" (default, Map), "gson", "protobuf-json", "bean".
// "protobuf" is kept as a legacy compatibility value and is not recommended.
// An unknown value is rejected when the reference is created (see init()), rather
// than silently falling back to the Map generalizer.
//
// Note: the generic mode is different from the transport serialization set via
// WithSerialization; the latter controls the on-the-wire encoding (hessian2 /
// protobuf / json / msgpack).
func WithGenericType(genericType string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Generic = genericType
	}
}

// WithSticky sets ReferenceOptions.Reference.Sticky so invocations prefer the same provider.
func WithSticky() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Sticky = true
	}
}

// WithIDL sets ReferenceOptions.Reference.IDLMode for legacy clients.
//
// Deprecated: this option will be removed in the next version. The IDL mode
// switch is no longer supported by dubbo-go.
func WithIDL(IDLMode string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.IDLMode = IDLMode
	}
}

// ========== Protocol to consume ==========

// WithProtocolDubbo sets ReferenceOptions.Reference.Protocol to Dubbo.
func WithProtocolDubbo() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.DubboProtocol
	}
}

// WithProtocolTriple sets ReferenceOptions.Reference.Protocol to Triple.
func WithProtocolTriple() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.TriProtocol
	}
}

// WithProtocolJsonRPC sets ReferenceOptions.Reference.Protocol to JSON-RPC.
func WithProtocolJsonRPC() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.JSONRPCProtocol
	}
}

// WithProtocol sets ReferenceOptions.Reference.Protocol to a custom protocol name.
func WithProtocol(protocol string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = protocol
	}
}

// WithRequestTimeout sets ReferenceOptions.Reference.RequestTimeout for every call on this reference.
// A call-level WithCallRequestTimeout overrides it for an individual invocation.
func WithRequestTimeout(timeout time.Duration) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.RequestTimeout = timeout.String()
	}
}

// WithForceTag sets ReferenceOptions.Reference.ForceTag to require tag-based routing.
func WithForceTag() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ForceTag = true
	}
}

// WithMeshProviderPort sets ReferenceOptions.Reference.MeshProviderPort for mesh routing.
func WithMeshProviderPort(port int) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.MeshProviderPort = port
	}
}

// WithMethod appends method to ReferenceOptions.Reference.MethodsConfig when it is non-nil.
// Use it with reference-level options to override configuration for a specific method.
func WithMethod(method *global.MethodConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if method == nil {
			return
		}
		if opts.Reference.MethodsConfig == nil {
			opts.Reference.MethodsConfig = make([]*global.MethodConfig, 0)
		}
		opts.Reference.MethodsConfig = append(opts.Reference.MethodsConfig, method)
	}
}

// WithParam adds or replaces one entry in ReferenceOptions.Reference.Params.
// Use WithParams to replace the complete parameter map.
func WithParam(k, v string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if opts.Reference.Params == nil {
			opts.Reference.Params = make(map[string]string)
		}
		opts.Reference.Params[k] = v
	}
}

// WithRouter appends router configurations to ReferenceOptions.Routers.
// Use SetClientRouters only when framework-loaded configuration must replace the slice.
func WithRouter(routers ...*global.RouterConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(routers) > 0 {
			opts.Routers = append(opts.Routers, routers...)
		}
	}
}

// ---------- For framework ----------
// These functions should not be invoked by users

func setReference(reference *global.ReferenceConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference = reference
	}
}

func setInterfaceName(interfaceName string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.InterfaceName = interfaceName
	}
}

func setConsumer(consumer *global.ConsumerConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Consumer = consumer
	}
}

func setMetrics(mc *global.MetricsConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Metrics = mc
	}
}

func setOtel(oc *global.OtelConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Otel = oc
	}
}

func setTLS(tls *global.TLSConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.TLS = tls
	}
}

func setApplication(application *global.ApplicationConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Application = application
	}
}

// setProtocols sets the protocols configuration for the service reference.
// This is an internal function used by the framework to configure protocol settings.
// It accepts a map of protocol configurations where the key is the protocol name
// and the value is the corresponding protocol configuration.
func setProtocols(protocols map[string]*global.ProtocolConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Protocols = protocols
	}
}

func setShutdown(shutdown *global.ShutdownConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Shutdown = shutdown
	}
}

func setRegistries(regs map[string]*global.RegistryConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Registries = regs
	}
}

// setRouters sets the routers configuration for the service reference.
// This is an internal framework function for applying router settings to
// reference options. It replaces the current router slice.
func setRouters(routers []*global.RouterConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Routers = routers
	}
}

type ClientOptions struct {
	Consumer    *global.ConsumerConfig
	Application *global.ApplicationConfig
	Registries  map[string]*global.RegistryConfig
	Shutdown    *global.ShutdownConfig
	Metrics     *global.MetricsConfig
	Otel        *global.OtelConfig
	TLS         *global.TLSConfig
	Protocols   map[string]*global.ProtocolConfig
	Routers     []*global.RouterConfig

	overallReference *global.ReferenceConfig
}

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Consumer:         global.DefaultConsumerConfig(),
		Registries:       global.DefaultRegistriesConfig(),
		Application:      global.DefaultApplicationConfig(),
		Shutdown:         global.DefaultShutdownConfig(),
		Metrics:          global.DefaultMetricsConfig(),
		Otel:             global.DefaultOtelConfig(),
		TLS:              global.DefaultTLSConfig(),
		overallReference: global.DefaultReferenceConfig(),
	}
}

func (cliOpts *ClientOptions) init(opts ...ClientOption) error {
	for _, opt := range opts {
		opt(cliOpts)
	}

	if err := defaults.Set(cliOpts); err != nil {
		return err
	}

	consumerConf := cliOpts.Consumer

	// init registries
	regs := cliOpts.Registries
	if len(regs) > 0 {
		if len(consumerConf.RegistryIDs) <= 0 {
			consumerConf.RegistryIDs = make([]string, 0, len(regs))
			for key := range regs {
				consumerConf.RegistryIDs = append(consumerConf.RegistryIDs, key)
			}
		}
		consumerConf.RegistryIDs = commonCfg.TranslateIds(consumerConf.RegistryIDs)
		if err := internal.ValidateRegistryIDs(consumerConf.RegistryIDs, regs); err != nil {
			return err
		}
	}

	// init cluster
	if cliOpts.overallReference.Cluster == "" {
		cliOpts.overallReference.Cluster = constant.ClusterKeyFailover
	}

	// init protocol
	if cliOpts.Consumer.Protocol == "" {
		cliOpts.Consumer.Protocol = constant.TriProtocol
	}

	// init serialization
	if cliOpts.overallReference.Serialization == "" {
		cliOpts.overallReference.Serialization = constant.ProtobufSerialization
	}

	// todo(DMwangnima): is there any part that we should do compatibility processing?

	// init overallReference from Consumer config
	if consumerConf != nil {
		if cliOpts.overallReference.Filter == "" {
			cliOpts.overallReference.Filter = consumerConf.Filter
		}
		if len(cliOpts.overallReference.RegistryIDs) <= 0 {
			cliOpts.overallReference.RegistryIDs = consumerConf.RegistryIDs
		}
		if cliOpts.overallReference.TracingKey == "" {
			cliOpts.overallReference.TracingKey = consumerConf.TracingKey
		}
		if cliOpts.overallReference.Check == nil {
			cliOpts.overallReference.Check = &consumerConf.Check
		}
	}
	// init graceful_shutdown
	graceful_shutdown.Init(graceful_shutdown.SetShutdownConfig(cliOpts.Shutdown))
	return nil
}

type ClientOption func(*ClientOptions)

// WithClientNoCheck sets ClientOptions.Consumer.Check to false for all client references.
// A reference-level WithCheck can enable checking for a specific service.
func WithClientNoCheck() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Check = false
	}
}

// WithClientURL sets ClientOptions.overallReference.URL as the default direct invocation URL.
// A reference-level WithURL overrides it for a specific service.
func WithClientURL(url string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.URL = url
	}
}

// WithClientFilter sets ClientOptions.overallReference.Filter as the default filter chain.
// A reference-level WithFilter overrides it for a specific service.
func WithClientFilter(filter string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Filter = filter
	}
}

// WithClientRegistryIDs sets ClientOptions.Consumer.RegistryIDs.
// Pair it with WithClientRegistry using matching registry IDs.
func WithClientRegistryIDs(registryIDs ...string) ClientOption {
	return func(opts *ClientOptions) {
		if len(registryIDs) > 0 {
			opts.Consumer.RegistryIDs = registryIDs
		}
	}
}

// WithClientRegistry builds a registry configuration and adds it to ClientOptions.Registries.
// Use registry.WithID and select the same ID with WithClientRegistryIDs when needed.
func WithClientRegistry(opts ...registry.Option) ClientOption {
	regOpts := registry.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		cliOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// WithClientShutdown applies graceful shutdown options to ClientOptions.Shutdown.
func WithClientShutdown(opts ...graceful_shutdown.Option) ClientOption {
	sdOpts := graceful_shutdown.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		cliOpts.Shutdown = sdOpts.Shutdown
	}
}

// WithClientTLSOption applies tls.Option values to ClientOptions.TLS.
// Configure compatible TLS settings on the server with server.WithServerTLSOption.
func WithClientTLSOption(opts ...tls.Option) ClientOption {
	tlsOpts := tls.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		if cliOpts.TLS == nil {
			cliOpts.TLS = new(global.TLSConfig)
		}
		cliOpts.TLS = tlsOpts.TLSConf
	}
}

// ========== Cluster Strategy ==========

// WithClientClusterAvailable sets ClientOptions.overallReference.Cluster to available.
func WithClientClusterAvailable() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyAvailable
	}
}

// WithClientClusterBroadcast sets ClientOptions.overallReference.Cluster to broadcast.
func WithClientClusterBroadcast() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithClientClusterFailBack sets ClientOptions.overallReference.Cluster to failback.
func WithClientClusterFailBack() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailback
	}
}

// WithClientClusterFailFast sets ClientOptions.overallReference.Cluster to fail-fast.
func WithClientClusterFailFast() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailfast
	}
}

// WithClientClusterFailOver sets ClientOptions.overallReference.Cluster to failover.
// Pair it with WithClientRetries to control the default retry count.
func WithClientClusterFailOver() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailover
	}
}

// WithClientClusterFailSafe sets ClientOptions.overallReference.Cluster to fail-safe.
func WithClientClusterFailSafe() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithClientClusterForking sets ClientOptions.overallReference.Cluster to forking.
func WithClientClusterForking() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyForking
	}
}

// WithClientClusterZoneAware sets ClientOptions.overallReference.Cluster to zone-aware.
func WithClientClusterZoneAware() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithClientClusterAdaptiveService sets ClientOptions.overallReference.Cluster to adaptive-service.
func WithClientClusterAdaptiveService() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithClientClusterStrategy sets ClientOptions.overallReference.Cluster to a custom strategy name.
func WithClientClusterStrategy(strategy string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = strategy
	}
}

// WithKeepAliveInterval is retained for compatibility and panics when applied.
//
// Deprecated: pass triple.WithKeepAliveInterval through protocol.WithTriple and WithClientProtocol.
func WithKeepAliveInterval(keepAliveInterval time.Duration) ClientOption {
	return func(_ *ClientOptions) {
		panic("use triple.WithKeepAliveInterval()")
	}
}

// WithKeepAliveTimeout is retained for compatibility and panics when applied.
//
// Deprecated: pass triple.WithKeepAliveTimeout through protocol.WithTriple and WithClientProtocol.
func WithKeepAliveTimeout(keepAliveTimeout time.Duration) ClientOption {
	return func(_ *ClientOptions) {
		panic("use triple.WithKeepAliveTimeout()")
	}
}

// ========== LoadBalance Strategy ==========

// WithClientLoadBalanceConsistentHashing sets ClientOptions.overallReference.Loadbalance to consistent hashing.
func WithClientLoadBalanceConsistentHashing() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithClientLoadBalanceLeastActive sets ClientOptions.overallReference.Loadbalance to least active.
func WithClientLoadBalanceLeastActive() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithClientLoadBalanceRandom sets ClientOptions.overallReference.Loadbalance to random.
func WithClientLoadBalanceRandom() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithClientLoadBalanceRoundRobin sets ClientOptions.overallReference.Loadbalance to round robin.
func WithClientLoadBalanceRoundRobin() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithClientLoadBalanceP2C sets ClientOptions.overallReference.Loadbalance to power of two choices.
func WithClientLoadBalanceP2C() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithClientLoadBalance sets ClientOptions.overallReference.Loadbalance to a custom strategy name.
func WithClientLoadBalance(lb string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = lb
	}
}

// WithClientRetries sets ClientOptions.overallReference.Retries as the default retry count.
// It is commonly paired with WithClientClusterFailOver.
func WithClientRetries(retries int) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Retries = strconv.Itoa(retries)
	}
}

// WithClientGroup sets ClientOptions.overallReference.Group as the default provider group.
// A reference-level WithGroup can override it and must match the server service group.
func WithClientGroup(group string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Group = group
	}
}

// WithClientVersion sets ClientOptions.overallReference.Version as the default service version.
// A reference-level WithVersion can override it and must match the server service version.
func WithClientVersion(version string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Version = version
	}
}

// WithClientSerializationJSON sets ClientOptions.overallReference.Serialization to JSON.
// The selected serialization must be supported by the provider protocol.
func WithClientSerializationJSON() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Serialization = constant.JSONSerialization
	}
}

// WithClientSerialization sets ClientOptions.overallReference.Serialization.
// The selected serialization must be supported by the provider protocol.
func WithClientSerialization(ser string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Serialization = ser
	}
}

// WithClientProvidedBy sets ClientOptions.overallReference.ProvidedBy as default provider metadata.
func WithClientProvidedBy(providedBy string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.ProvidedBy = providedBy
	}
}

// todo(DMwangnima): implement this functionality
// func WithAsync() ClientOption {
//	return func(opts *ClientOptions) {
//		opts.Consumer.Async = true
//	}
// }

// WithClientParams replaces ClientOptions.overallReference.Params when params is non-empty.
// Use WithClientParam to add or override one default parameter.
func WithClientParams(params map[string]string) ClientOption {
	return func(opts *ClientOptions) {
		if len(params) <= 0 {
			return
		}
		opts.overallReference.Params = params
	}
}

// WithClientParam adds or replaces one entry in ClientOptions.overallReference.Params.
// Use WithClientParams to replace the complete default parameter map.
func WithClientParam(k, v string) ClientOption {
	return func(opts *ClientOptions) {
		if opts.overallReference.Params == nil {
			opts.overallReference.Params = make(map[string]string, 8)
		}
		opts.overallReference.Params[k] = v
	}
}

// WithClientRouter appends router configurations to ClientOptions.Routers.
// Use SetClientRouters only when framework-loaded configuration must replace the slice.
func WithClientRouter(routers ...*global.RouterConfig) ClientOption {
	return func(opts *ClientOptions) {
		if len(routers) > 0 {
			opts.Routers = append(opts.Routers, routers...)
		}
	}
}

// todo(DMwangnima): implement this functionality
// func WithClientGeneric(generic bool) ClientOption {
//	return func(opts *ClientOptions) {
//		if generic {
//			opts.Consumer.Generic = "true"
//		} else {
//			opts.Consumer.Generic = "false"
//		}
//	}
// }

// WithClientSticky sets ClientOptions.overallReference.Sticky for all references by default.
// A reference-level WithSticky enables the same behavior for one service.
func WithClientSticky() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Sticky = true
	}
}

// ========== Protocol to consume ==========

// WithClientProtocolDubbo sets ClientOptions.Consumer.Protocol to Dubbo.
func WithClientProtocolDubbo() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.DubboProtocol
	}
}

// WithClientProtocolTriple sets ClientOptions.Consumer.Protocol to Triple.
func WithClientProtocolTriple() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.TriProtocol
	}
}

// WithClientProtocolJsonRPC sets ClientOptions.Consumer.Protocol to JSON-RPC.
func WithClientProtocolJsonRPC() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.JSONRPCProtocol
	}
}

// WithClientProtocol builds a client protocol configuration and assigns it to
// ClientOptions.overallReference.ProtocolClientConfig. Pass protocol.WithTriple,
// protocol.WithDubbo, protocol.WithJSONRPC, or another protocol.ClientOption.
func WithClientProtocol(opts ...protocol.ClientOption) ClientOption {
	proOpts := protocol.NewClientOptions(opts...)

	return func(srvOpts *ClientOptions) {
		if srvOpts.overallReference.ProtocolClientConfig == nil {
			srvOpts.overallReference.ProtocolClientConfig = new(global.ClientProtocolConfig)
		}
		srvOpts.overallReference.ProtocolClientConfig = proOpts.ProtocolClient
	}
}

// WithClientRequestTimeout sets ClientOptions.Consumer.RequestTimeout as the client default.
// A reference-level WithRequestTimeout or call-level WithCallRequestTimeout can override it.
func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.RequestTimeout = timeout.String()
	}
}

// WithClientForceTag sets ClientOptions.overallReference.ForceTag for default tag routing.
func WithClientForceTag() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.ForceTag = true
	}
}

// WithClientMeshProviderPort sets ClientOptions.overallReference.MeshProviderPort for mesh routing.
func WithClientMeshProviderPort(port int) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.MeshProviderPort = port
	}
}

// SetClientRegistries replaces ClientOptions.Registries with framework-loaded configuration.
// User code should prefer WithClientRegistry and WithClientRegistryIDs.
func SetClientRegistries(regs map[string]*global.RegistryConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Registries = regs
	}
}

// SetClientApplication assigns framework-loaded application configuration to ClientOptions.Application.
func SetClientApplication(application *global.ApplicationConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Application = application
	}
}

// SetClientConsumer assigns framework-loaded consumer configuration to ClientOptions.Consumer.
func SetClientConsumer(consumer *global.ConsumerConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer = consumer
	}
}

// SetClientShutdown assigns framework-loaded shutdown configuration to ClientOptions.Shutdown.
// User code should prefer WithClientShutdown.
func SetClientShutdown(shutdown *global.ShutdownConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Shutdown = shutdown
	}
}

// SetClientMetrics assigns framework-loaded metrics configuration to ClientOptions.Metrics.
func SetClientMetrics(metrics *global.MetricsConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Metrics = metrics
	}
}

// SetClientOtel assigns framework-loaded OpenTelemetry configuration to ClientOptions.Otel.
func SetClientOtel(otel *global.OtelConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Otel = otel
	}
}

// SetClientTLS assigns framework-loaded TLS configuration to ClientOptions.TLS.
// User code should prefer WithClientTLSOption.
func SetClientTLS(tls *global.TLSConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.TLS = tls
	}
}

// SetClientProtocols replaces ClientOptions.Protocols with framework-loaded configuration.
// User code should prefer WithClientProtocol.
func SetClientProtocols(protocols map[string]*global.ProtocolConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Protocols = protocols
	}
}

// SetClientRouters replaces ClientOptions.Routers with framework-loaded configuration.
// User code should prefer WithClientRouter or reference-level WithRouter.
func SetClientRouters(routers []*global.RouterConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Routers = routers
	}
}

// todo: need to be consistent with MethodConfig
type CallOptions struct {
	RequestTimeout  string
	Retries         string
	ResponseHeader  *http.Header
	ResponseTrailer *http.Header
}

type CallOption func(*CallOptions)

func newDefaultCallOptions() *CallOptions {
	return &CallOptions{}
}

// WithCallRequestTimeout sets CallOptions.RequestTimeout for one Triple or Dubbo invocation.
// It overrides WithRequestTimeout and WithClientRequestTimeout for that call.
func WithCallRequestTimeout(timeout time.Duration) CallOption {
	return func(opts *CallOptions) {
		opts.RequestTimeout = timeout.String()
	}
}

// WithCallRetries sets CallOptions.Retries for one Triple or Dubbo invocation.
// It overrides WithRetries and WithClientRetries for that call.
func WithCallRetries(retries int) CallOption {
	return func(opts *CallOptions) {
		opts.Retries = strconv.Itoa(retries)
	}
}

// WithResponseHeader sets CallOptions.ResponseHeader as the target for response headers.
// Currently, only Triple unary calls populate this option (including error
// responses when metadata is available).
func WithResponseHeader(header *http.Header) CallOption {
	return func(opts *CallOptions) {
		opts.ResponseHeader = header
	}
}

// WithResponseTrailer sets CallOptions.ResponseTrailer as the target for response trailers.
// Currently, only Triple unary calls populate this option (including error
// responses when metadata is available).
func WithResponseTrailer(trailer *http.Header) CallOption {
	return func(opts *CallOptions) {
		opts.ResponseTrailer = trailer
	}
}

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

// WithCheck requires this reference to pass its availability check during initialization.
// Use it to fail early when no usable provider can be resolved instead of discovering the
// problem on the first invocation, for example when a mandatory dependency must be ready
// before the application starts serving traffic. It overrides WithClientNoCheck for this reference.
func WithCheck() ReferenceOption {
	return func(opts *ReferenceOptions) {
		check := true
		opts.Reference.Check = &check
	}
}

// WithURL invokes this service through the supplied direct provider URL and bypasses registry
// discovery. It is useful for local testing or fixed endpoints but does not follow registry
// instance changes or fail over to providers that are not encoded in the URL.
func WithURL(url string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.URL = url
	}
}

// WithFilter selects the consumer filter chain that wraps invocations for this reference.
// The value is a comma-separated list of registered filter names, in execution order. Use it
// to add cross-cutting behavior such as tracing, metrics, authentication, or custom middleware.
func WithFilter(filter string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Filter = filter
	}
}

// WithInterface identifies the remote service that this reference discovers and invokes.
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

// WithRegistryIDs limits this reference to the named registries. Each ID must match a
// registry added with WithRegistry or WithClientRegistry; when omitted, the reference
// inherits the client-level selection. Use it when one client connects to several registries
// but a service must be discovered from only one environment or region.
func WithRegistryIDs(registryIDs ...string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(registryIDs) > 0 {
			opts.Reference.RegistryIDs = registryIDs
		}
	}
}

// WithRegistry makes a registry configuration available to this reference for service
// discovery. Give each registry a distinct registry.WithID and use WithRegistryIDs when
// only a subset should be queried. Use it for service-specific discovery; shared registries
// are usually configured once with WithClientRegistry.
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

// WithClusterAvailable invokes the first provider currently reporting itself available.
// It performs neither load balancing nor retries and fails when none are available. Use it
// only when selecting any healthy endpoint is more important than distributing traffic.
func WithClusterAvailable() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAvailable
	}
}

// WithClusterBroadcast invokes every provider sequentially. The call reports an error if
// any provider fails, so use it for operations that must reach all instances, such as
// refreshing local state on every node. The service operation should tolerate repeated calls.
func WithClusterBroadcast() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithClusterFailBack returns an empty successful result when the initial invocation fails
// and schedules background retries with exponential backoff. It suits notifications where
// eventual delivery matters more than reporting the first failure to the caller.
func WithClusterFailBack() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailback
	}
}

// WithClusterFailFast selects one provider, invokes it once, and returns its error without
// retrying another provider. It suits non-idempotent operations where retries are unsafe.
func WithClusterFailFast() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailfast
	}
}

// WithClusterFailOver retries non-business failures on reselected providers. WithRetries
// controls the additional attempts after the initial call; business errors are returned
// immediately and are not retried. Use it for idempotent calls that should survive an
// unavailable provider, and avoid it when repeating the operation can duplicate side effects.
func WithClusterFailOver() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailover
	}
}

// WithClusterFailSafe logs and suppresses provider or discovery errors, returning an empty
// result to the caller. It is intended for best-effort operations such as audit logging.
func WithClusterFailSafe() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithClusterForking invokes multiple selected providers concurrently and returns the first
// completed result. Use it for idempotent, latency-sensitive reads; it reduces tail latency at
// the cost of duplicate work and extra provider load.
func WithClusterForking() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyForking
	}
}

// WithClusterZoneAware chooses among multiple registries by preferring a registry marked
// preferred, then one in the request's zone, and finally a weighted available registry. Use
// it for multi-region deployments that should keep traffic local while retaining fallback.
func WithClusterZoneAware() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithClusterAdaptiveService selects providers using adaptive remaining-capacity metrics.
// It requires P2C load balancing and participating providers that return adaptive metrics.
// Use the pair for workloads whose instance capacity varies significantly at runtime.
func WithClusterAdaptiveService() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithCluster selects a registered cluster extension by name. Reference creation or
// invocation fails if no extension has been registered under that name. Use it when a built-in
// failure policy does not match the service and the application has registered a custom one.
func WithCluster(cluster string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = cluster
	}
}

// ========== LoadBalance Strategy ==========

// WithLoadBalanceConsistentHashing routes calls with the same configured argument values to
// the same provider while the provider set is stable. Use it for affinity workloads such as
// per-user caches; provider membership changes can remap some keys.
func WithLoadBalanceConsistentHashing() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithLoadBalanceLeastActive favors providers with the fewest in-flight requests. Ties are
// resolved by warm-up-adjusted weight. Use it when request durations vary and queueing work on
// a busy instance would hurt latency.
func WithLoadBalanceLeastActive() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithLoadBalanceRandom chooses providers randomly in proportion to their effective weight.
// It is a low-overhead general-purpose choice for statistically even traffic distribution.
func WithLoadBalanceRandom() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithLoadBalanceRoundRobin distributes calls in a smooth weighted round-robin sequence. Use
// it when requests have similar cost and predictable per-instance traffic is desirable.
func WithLoadBalanceRoundRobin() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithLoadBalanceP2C samples two providers and chooses the one with more recorded remaining
// capacity. Use it with WithClusterAdaptiveService for providers that publish adaptive metrics.
func WithLoadBalanceP2C() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithLoadBalance selects a registered load-balancing extension by name. Use it when the
// built-in algorithms do not satisfy a domain-specific placement requirement.
func WithLoadBalance(lb string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = lb
	}
}

// WithRetries sets the number of additional attempts made after an initial failure by
// retry-capable cluster strategies such as failover. A value of zero means one attempt total.
// Enable retries only for idempotent operations because another provider may repeat the work.
func WithRetries(retries int) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Retries = strconv.Itoa(retries)
	}
}

// WithGroup restricts discovery to providers exported in the same group, allowing multiple
// logical implementations of one interface to coexist. Use groups to separate environments,
// tenants, or implementations; a mismatched group yields no provider.
func WithGroup(group string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Group = group
	}
}

// WithVersion restricts discovery to providers exporting the same service version. A
// mismatched version yields no provider even when the interface name matches.
func WithVersion(version string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Version = version
	}
}

// WithSerializationJSON encodes request and response payloads with JSON. The provider and
// selected protocol must support JSON or the invocation cannot be decoded. Use it for
// interoperability when human-readable JSON matters more than compact binary payloads.
func WithSerializationJSON() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Serialization = constant.JSONSerialization
	}
}

// WithSerialization selects the wire serialization by extension name. The provider and
// selected protocol must support the same serialization. Use it when both sides have installed
// a non-default serialization extension.
func WithSerialization(serialization string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Serialization = serialization
	}
}

// WithProvidedBy supplies a comma-separated list of provider application names for
// application-level service discovery. The registry subscribes to these applications
// directly instead of resolving the interface through dynamic service-name mapping. Use it
// when the provider applications are known ahead of time or mapping metadata is unavailable.
func WithProvidedBy(providedBy string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ProvidedBy = providedBy
	}
}

// WithAsync builds an asynchronous proxy for this reference. When the service implements
// common.AsyncCallbackService, completed invocations are delivered to its callback. Use it
// when the caller should continue work instead of waiting synchronously for the response.
func WithAsync() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Async = true
	}
}

// WithParams replaces the custom URL parameters published with this reference. Filters,
// routers, protocols, and extensions may consume these keys. Use it to configure an extension
// with several related values; use WithParam to change one key without replacing the map.
func WithParams(params map[string]string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(params) <= 0 {
			return
		}
		opts.Reference.Params = params
	}
}

// WithGeneric enables map-based generic invocation, allowing calls without generated service
// stubs by representing business objects as generic maps. Use WithGenericType for another
// supported generalization format.
func WithGeneric() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Generic = "true"
	}
}

// WithGenericType enables generic invocation and selects how business objects are represented
// when generated service types are unavailable.
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

// WithSticky keeps selecting the previously chosen provider while it remains available,
// reducing provider churn but potentially weakening load distribution. Use it for providers
// that keep session-local state and prefer consistent hashing when a stable key is available.
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

// WithProtocolDubbo restricts this reference to providers exported with the Dubbo protocol.
// Use it when consuming an existing Dubbo-protocol service rather than the default Triple endpoint.
func WithProtocolDubbo() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.DubboProtocol
	}
}

// WithProtocolTriple restricts this reference to providers exported with the Triple protocol.
// Use it for Triple or gRPC-compatible services and their HTTP/2 features.
func WithProtocolTriple() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.TriProtocol
	}
}

// WithProtocolJsonRPC restricts this reference to providers exported with JSON-RPC. Use it when
// interoperating with a provider exposed through JSON-RPC rather than Dubbo or Triple.
func WithProtocolJsonRPC() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.JSONRPCProtocol
	}
}

// WithProtocol restricts this reference to a protocol registered under the supplied name. Use
// it for a custom protocol extension; prefer the named helpers for built-in protocols.
func WithProtocol(protocol string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = protocol
	}
}

// WithRequestTimeout limits how long each invocation on this reference may wait before it
// fails with a timeout. Set it to the service's expected latency budget so stalled providers do
// not hold resources indefinitely. A call-level WithCallRequestTimeout takes precedence.
func WithRequestTimeout(timeout time.Duration) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.RequestTimeout = timeout.String()
	}
}

// WithForceTag prevents tag routing from falling back to untagged providers when no provider
// matches the requested tag; the invocation fails instead. Use it for strict traffic isolation,
// such as canary or tenant pools that must never spill into the default provider group.
func WithForceTag() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ForceTag = true
	}
}

// WithMeshProviderPort overrides the provider port used to build the direct Kubernetes DNS
// address in mesh mode. Use it when the mesh-routed service listens on a non-default port; it
// has no effect unless mesh mode is enabled.
func WithMeshProviderPort(port int) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.MeshProviderPort = port
	}
}

// WithMethod adds method-specific settings such as timeout, retries, or load balancing.
// Method settings take precedence over the corresponding reference defaults. Use it when one
// method is slower, non-idempotent, or otherwise needs different invocation behavior.
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

// WithParam adds one custom URL parameter consumed by filters, routers, protocols, or other
// extensions. Use it for an extension setting that has no typed option. A later call with the
// same key replaces the earlier value.
func WithParam(k, v string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if opts.Reference.Params == nil {
			opts.Reference.Params = make(map[string]string)
		}
		opts.Reference.Params[k] = v
	}
}

// WithRouter adds routing rules that filter or reorder candidate providers before load
// balancing. Use it for conditions such as region, tag, or application routing. Multiple calls
// append rules and preserve their configured order.
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

// WithClientNoCheck allows client references to initialize even when no provider is currently
// available, so applications can start before their dependencies. Calls still fail until a
// provider appears. Use it for independently deployed or temporarily optional dependencies;
// WithCheck can restore fail-fast initialization for one mandatory reference.
func WithClientNoCheck() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Check = false
	}
}

// WithClientURL sends client references directly to the supplied URL instead of discovering
// providers through a registry. Use it for tests or a client dedicated to one fixed endpoint;
// it does not follow registry changes. A reference-level WithURL overrides this default.
func WithClientURL(url string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.URL = url
	}
}

// WithClientFilter selects the comma-separated consumer filter chain applied to references
// by default. Use it for cross-cutting behavior shared by all calls, such as tracing or
// authentication. A reference-level WithFilter replaces it for one service.
func WithClientFilter(filter string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Filter = filter
	}
}

// WithClientRegistryIDs limits service discovery to the named client registries by default.
// Each ID must match a registry added with WithClientRegistry. Use it to keep all references
// on a selected environment or region when several registries are configured.
func WithClientRegistryIDs(registryIDs ...string) ClientOption {
	return func(opts *ClientOptions) {
		if len(registryIDs) > 0 {
			opts.Consumer.RegistryIDs = registryIDs
		}
	}
}

// WithClientRegistry adds a registry that client references can use for service discovery.
// Assign distinct registry.WithID values when configuring more than one registry. Configure
// shared discovery here instead of repeating WithRegistry for every reference.
//
// For example, this configures Nacos as the default registry for the client:
//
//	client.NewClient(
//		client.WithClientRegistry(
//			registry.WithNacos(),
//			registry.WithID("nacos"),
//			registry.WithAddress("127.0.0.1:8848"),
//		),
//		client.WithClientRegistryIDs("nacos"),
//	)
func WithClientRegistry(opts ...registry.Option) ClientOption {
	regOpts := registry.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		cliOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// WithClientShutdown controls how long client shutdown waits for in-flight calls and cleanup
// steps before forcing progress to the next shutdown phase. Use it to align graceful shutdown
// with the process termination budget and avoid dropping active RPCs during deployment.
func WithClientShutdown(opts ...graceful_shutdown.Option) ClientOption {
	sdOpts := graceful_shutdown.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		cliOpts.Shutdown = sdOpts.Shutdown
	}
}

// WithClientTLSOption enables and configures TLS for client connections. The certificates,
// server name, and trust roots must be compatible with server.WithServerTLSOption. Use it when
// traffic must be encrypted or the client must authenticate the provider or itself.
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

// WithClientClusterAvailable makes references invoke the first available provider without
// load balancing or retries unless a reference selects another strategy. Use it only when any
// healthy endpoint is sufficient and even traffic distribution is not required.
func WithClientClusterAvailable() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyAvailable
	}
}

// WithClientClusterBroadcast makes references invoke every provider and report an error if
// any invocation fails unless a reference selects another strategy. Use it for operations such
// as cache invalidation that intentionally run on every provider.
func WithClientClusterBroadcast() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyBroadcast
	}
}

// WithClientClusterFailBack makes references suppress initial failures and retry them in the
// background with exponential backoff, which is suitable for eventual notifications.
func WithClientClusterFailBack() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailback
	}
}

// WithClientClusterFailFast makes references invoke one provider once and return its error
// immediately, avoiding unsafe retries for non-idempotent operations.
func WithClientClusterFailFast() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailfast
	}
}

// WithClientClusterFailOver makes references retry non-business failures on reselected
// providers. Use it as the client default only when calls are generally idempotent;
// WithClientRetries controls the additional attempts after the first call.
func WithClientClusterFailOver() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailover
	}
}

// WithClientClusterFailSafe makes references log and suppress invocation errors, returning
// an empty result for best-effort operations.
func WithClientClusterFailSafe() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailsafe
	}
}

// WithClientClusterForking makes references invoke multiple providers concurrently and use
// the first completed result, trading duplicate work for lower tail latency.
func WithClientClusterForking() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyForking
	}
}

// WithClientClusterZoneAware makes multi-registry references prefer a configured preferred
// registry, then the request's zone, before weighted fallback to another registry. Use it to
// keep traffic local in multi-region deployments while retaining disaster fallback.
func WithClientClusterZoneAware() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyZoneAware
	}
}

// WithClientClusterAdaptiveService makes references select providers from reported remaining
// capacity. It requires P2C load balancing and adaptive-service-enabled providers. Use it when
// instance capacity changes dynamically and simple static weights are insufficient.
func WithClientClusterAdaptiveService() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// WithClientClusterStrategy selects a registered cluster extension as the client default. Use
// it for an application-specific failure policy; a reference-level cluster option overrides it.
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

// WithClientLoadBalanceConsistentHashing keeps calls with the same configured argument values
// on the same provider while the provider set remains stable. Use it for cache or session
// affinity shared by most references on this client.
func WithClientLoadBalanceConsistentHashing() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

// WithClientLoadBalanceLeastActive favors providers with the fewest in-flight requests and
// uses effective weight to resolve ties. Use it when call durations vary and busy providers
// should receive less new work.
func WithClientLoadBalanceLeastActive() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// WithClientLoadBalanceRandom chooses providers randomly in proportion to effective weight.
// It is a low-overhead general default for statistically even traffic.
func WithClientLoadBalanceRandom() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

// WithClientLoadBalanceRoundRobin distributes calls using smooth weighted round robin. Use it
// when request costs are similar and predictable instance shares are useful.
func WithClientLoadBalanceRoundRobin() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

// WithClientLoadBalanceP2C samples two providers and chooses the one with more recorded
// remaining capacity. Use it with WithClientClusterAdaptiveService for adaptive providers.
func WithClientLoadBalanceP2C() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

// WithClientLoadBalance selects a registered load-balancing extension as the client default.
// Use it for domain-specific placement rules; a reference-level option overrides it.
func WithClientLoadBalance(lb string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = lb
	}
}

// WithClientRetries sets the default number of additional attempts after the initial call for
// retry-capable strategies. Use retries only for idempotent operations because another provider
// may repeat the work. A reference-level or call-level value takes precedence.
func WithClientRetries(retries int) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Retries = strconv.Itoa(retries)
	}
}

// WithClientGroup restricts references to providers in this group by default. A mismatched
// group produces no providers. Use it when this client should consume one logical deployment,
// such as a tenant or environment; WithGroup overrides it for one reference.
func WithClientGroup(group string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Group = group
	}
}

// WithClientVersion restricts references to this service version by default. WithVersion
// overrides it for one reference. Use it during incompatible API migrations when a client must
// remain on a specific provider version.
func WithClientVersion(version string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Version = version
	}
}

// WithClientSerializationJSON uses JSON payload encoding for references by default. Providers
// and protocols that do not support JSON cannot decode those calls. Use it for interoperability
// when human-readable JSON is preferred over compact binary serialization.
func WithClientSerializationJSON() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Serialization = constant.JSONSerialization
	}
}

// WithClientSerialization selects the default wire serialization by extension name. Providers
// must advertise a compatible serialization. Use it when both sides install the same custom
// serialization extension.
func WithClientSerialization(ser string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Serialization = ser
	}
}

// WithClientProvidedBy supplies the default comma-separated provider application names for
// application-level discovery, bypassing dynamic interface-to-application mapping. Use it when
// provider applications are known or service-name mapping metadata is unavailable.
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

// WithClientParams replaces the custom URL parameters inherited by client references. These
// parameters can configure filters, routers, protocols, and extensions. Use it for extension
// settings shared by all references; reference-level params take precedence.
func WithClientParams(params map[string]string) ClientOption {
	return func(opts *ClientOptions) {
		if len(params) <= 0 {
			return
		}
		opts.overallReference.Params = params
	}
}

// WithClientParam adds one custom URL parameter inherited by references. A later call with
// the same key replaces its value. Use it for an extension setting shared across references
// when no typed client option exists.
func WithClientParam(k, v string) ClientOption {
	return func(opts *ClientOptions) {
		if opts.overallReference.Params == nil {
			opts.overallReference.Params = make(map[string]string, 8)
		}
		opts.overallReference.Params[k] = v
	}
}

// WithClientRouter adds routing rules that filter or reorder candidate providers before load
// balancing for client references. Use it for routing policies shared by the client, such as
// preferring the local region; reference-level routers can add service-specific rules.
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

// WithClientSticky keeps each reference on its previously selected provider while that
// provider remains available. Use it for session-local provider state, accepting less even
// traffic distribution. WithSticky enables the same behavior for one reference.
func WithClientSticky() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Sticky = true
	}
}

// ========== Protocol to consume ==========

// WithClientProtocolDubbo discovers and invokes Dubbo-protocol providers by default. Use it
// when most services consumed by this client expose the classic Dubbo protocol.
func WithClientProtocolDubbo() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.DubboProtocol
	}
}

// WithClientProtocolTriple discovers and invokes Triple-protocol providers by default. Use it
// for Triple or gRPC-compatible services and their HTTP/2 features.
func WithClientProtocolTriple() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.TriProtocol
	}
}

// WithClientProtocolJsonRPC discovers and invokes JSON-RPC providers by default. Use it for a
// client whose services are primarily exposed through JSON-RPC.
func WithClientProtocolJsonRPC() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.JSONRPCProtocol
	}
}

// WithClientProtocol configures transport-specific client behavior such as Triple keepalive,
// message limits, or protocol selection. Use it to apply one transport policy to all references;
// a reference may supply its own protocol settings.
func WithClientProtocol(opts ...protocol.ClientOption) ClientOption {
	proOpts := protocol.NewClientOptions(opts...)

	return func(srvOpts *ClientOptions) {
		if srvOpts.overallReference.ProtocolClientConfig == nil {
			srvOpts.overallReference.ProtocolClientConfig = new(global.ClientProtocolConfig)
		}
		srvOpts.overallReference.ProtocolClientConfig = proOpts.ProtocolClient
	}
}

// WithClientRequestTimeout limits how long client calls wait by default before failing with
// a timeout. Set it to the application's usual dependency latency budget so stalled calls do
// not retain resources indefinitely. Reference and call-level timeouts take precedence.
func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.RequestTimeout = timeout.String()
	}
}

// WithClientForceTag prevents tag routing from falling back to untagged providers when no
// provider matches the requested tag. Use it when every reference must preserve strict canary,
// tenant, or environment isolation.
func WithClientForceTag() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.ForceTag = true
	}
}

// WithClientMeshProviderPort overrides the provider port used in Kubernetes service DNS
// addresses generated in mesh mode; it has no effect when mesh mode is disabled.
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

// WithCallRequestTimeout limits one Triple or Dubbo invocation. Use it when an individual call
// has a tighter or looser latency budget than the service default. It overrides WithRequestTimeout
// and WithClientRequestTimeout for that call.
func WithCallRequestTimeout(timeout time.Duration) CallOption {
	return func(opts *CallOptions) {
		opts.RequestTimeout = timeout.String()
	}
}

// WithCallRetries sets the additional attempts for one Triple or Dubbo invocation. Use it only
// when that operation is idempotent and needs a different resilience policy. It overrides
// WithRetries and WithClientRetries for that call.
func WithCallRetries(retries int) CallOption {
	return func(opts *CallOptions) {
		opts.Retries = strconv.Itoa(retries)
	}
}

// WithResponseHeader sets CallOptions.ResponseHeader as the target for response headers.
// Currently, only Triple unary calls populate this option (including error
// responses when metadata is available). Use it to inspect provider metadata such as tracing
// or application-specific response attributes after the call returns.
func WithResponseHeader(header *http.Header) CallOption {
	return func(opts *CallOptions) {
		opts.ResponseHeader = header
	}
}

// WithResponseTrailer sets CallOptions.ResponseTrailer as the target for response trailers.
// Currently, only Triple unary calls populate this option (including error
// responses when metadata is available). Use it for metadata emitted after the response body,
// such as final status or diagnostic information.
func WithResponseTrailer(trailer *http.Header) CallOption {
	return func(opts *CallOptions) {
		opts.ResponseTrailer = trailer
	}
}

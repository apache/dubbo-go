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
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/proxy"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/tls"
)

type ReferenceOptions struct {
	Reference *global.ReferenceConfig
	Consumer  *global.ConsumerConfig
	Metrics   *global.MetricsConfig
	Otel      *global.OtelConfig
	TLS       *global.TLSConfig

	pxy          *proxy.Proxy
	id           string
	invoker      base.Invoker
	urls         []*common.URL
	metaDataType string
	info         *ClientInfo

	methodsCompat     []*config.MethodConfig
	applicationCompat *config.ApplicationConfig
	registriesCompat  map[string]*config.RegistryConfig
}

func defaultReferenceOptions() *ReferenceOptions {
	return &ReferenceOptions{
		Reference: global.DefaultReferenceConfig(),
		Metrics:   global.DefaultMetricsConfig(),
		Otel:      global.DefaultOtelConfig(),
		TLS:       global.DefaultTLSConfig(),
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

	app := refOpts.applicationCompat
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
	methods := refConf.MethodsConfig
	if length := len(methods); length > 0 {
		refOpts.methodsCompat = make([]*config.MethodConfig, length)
		for i, method := range methods {
			refOpts.methodsCompat[i] = compatMethodConfig(method)
			if err := refOpts.methodsCompat[i].Init(); err != nil {
				return err
			}
		}
	}

	// init cluster
	// TODO: use constant replace failover
	if refConf.Cluster == "" {
		refConf.Cluster = "failover"
	}

	// init registries
	if len(refOpts.registriesCompat) > 0 {
		regs := refOpts.registriesCompat
		if len(refConf.RegistryIDs) <= 0 {
			refConf.RegistryIDs = make([]string, len(regs))
			for key := range regs {
				refConf.RegistryIDs = append(refConf.RegistryIDs, key)
			}
		}
		refConf.RegistryIDs = commonCfg.TranslateIds(refConf.RegistryIDs)

		newRegs := make(map[string]*config.RegistryConfig)
		for _, id := range refConf.RegistryIDs {
			if reg, ok := regs[id]; ok {
				newRegs[id] = reg
			}
		}
		refOpts.registriesCompat = newRegs
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

	return commonCfg.Verify(refOpts)
}

type ReferenceOption func(*ReferenceOptions)

// ---------- For user ----------

func WithCheck() ReferenceOption {
	return func(opts *ReferenceOptions) {
		check := true
		opts.Reference.Check = &check
	}
}

func WithURL(url string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.URL = url
	}
}

func WithFilter(filter string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Filter = filter
	}
}

// WithInterface sets the interface name for the service reference.
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

func WithRegistryIDs(registryIDs ...string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(registryIDs) > 0 {
			opts.Reference.RegistryIDs = registryIDs
		}
	}
}

// ========== Cluster Strategy ==========

func WithClusterAvailable() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAvailable
	}
}

func WithClusterBroadcast() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithClusterFailBack() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailback
	}
}

func WithClusterFailFast() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailfast
	}
}

func WithClusterFailOver() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailover
	}
}

func WithClusterFailSafe() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithClusterForking() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyForking
	}
}

func WithClusterZoneAware() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithClusterAdaptiveService() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAdaptiveService
	}
}

func WithCluster(cluster string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Cluster = cluster
	}
}

// ========== LoadBalance Strategy ==========

func WithLoadBalanceConsistentHashing() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithLoadBalanceLeastActive() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithLoadBalanceRandom() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithLoadBalanceRoundRobin() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithLoadBalanceP2C() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithLoadBalance(lb string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = lb
	}
}

func WithRetries(retries int) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Retries = strconv.Itoa(retries)
	}
}

func WithGroup(group string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Group = group
	}
}

func WithVersion(version string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Version = version
	}
}

func WithSerializationJSON() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Serialization = constant.JSONSerialization
	}
}

func WithSerialization(serialization string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Serialization = serialization
	}
}

func WithProvidedBy(providedBy string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ProvidedBy = providedBy
	}
}

func WithAsync() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Async = true
	}
}

func WithParams(params map[string]string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(params) <= 0 {
			return
		}
		opts.Reference.Params = params
	}
}

func WithGeneric() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Generic = "true"
	}
}

func WithSticky() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Sticky = true
	}
}

// TODO: remove this function after old triple removed
func WithIDL(IDLMode string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.IDLMode = IDLMode
	}
}

// ========== Protocol to consume ==========

func WithProtocolDubbo() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.DubboProtocol
	}
}

func WithProtocolTriple() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.TriProtocol
	}
}

func WithProtocolJsonRPC() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.JSONRPCProtocol
	}
}

func WithProtocol(protocol string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = protocol
	}
}

func WithRequestTimeout(timeout time.Duration) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.RequestTimeout = timeout.String()
	}
}

func WithForceTag() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ForceTag = true
	}
}

func WithMeshProviderPort(port int) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.MeshProviderPort = port
	}
}

func WithMethod(opts ...config.MethodOption) ReferenceOption {
	regOpts := config.NewMethodOptions(opts...)

	return func(opts *ReferenceOptions) {
		if len(opts.Reference.MethodsConfig) == 0 {
			opts.Reference.MethodsConfig = make([]*global.MethodConfig, 0)
		}
		opts.Reference.MethodsConfig = append(opts.Reference.MethodsConfig, regOpts.Method)
	}
}

func WithParam(k, v string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if opts.Reference.Params == nil {
			opts.Reference.Params = make(map[string]string)
		}
		opts.Reference.Params[k] = v
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

func setApplicationCompat(app *config.ApplicationConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.applicationCompat = app
	}
}

func setRegistriesCompat(regs map[string]*config.RegistryConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.registriesCompat = regs
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

type ClientOptions struct {
	Consumer    *global.ConsumerConfig
	Application *global.ApplicationConfig
	Registries  map[string]*global.RegistryConfig
	Shutdown    *global.ShutdownConfig
	Metrics     *global.MetricsConfig
	Otel        *global.OtelConfig
	TLS         *global.TLSConfig

	overallReference  *global.ReferenceConfig
	applicationCompat *config.ApplicationConfig
	registriesCompat  map[string]*config.RegistryConfig
}

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Consumer:         global.DefaultConsumerConfig(),
		Registries:       make(map[string]*global.RegistryConfig),
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
	// init application
	application := cliOpts.Application
	if application != nil {
		cliOpts.applicationCompat = compatApplicationConfig(application)
		if err := cliOpts.applicationCompat.Init(); err != nil {
			return err
		}
	}

	// init registries
	regs := cliOpts.Registries
	if regs != nil {
		cliOpts.registriesCompat = make(map[string]*config.RegistryConfig)
		if len(consumerConf.RegistryIDs) <= 0 {
			consumerConf.RegistryIDs = make([]string, len(regs))
			for key := range regs {
				consumerConf.RegistryIDs = append(consumerConf.RegistryIDs, key)
			}
		}
		consumerConf.RegistryIDs = commonCfg.TranslateIds(consumerConf.RegistryIDs)

		for _, id := range consumerConf.RegistryIDs {
			if reg, ok := regs[id]; ok {
				cliOpts.registriesCompat[id] = compatRegistryConfig(reg)
				if err := cliOpts.registriesCompat[id].Init(); err != nil {
					return err
				}
			}
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

	// init graceful_shutdown
	graceful_shutdown.Init(graceful_shutdown.SetShutdownConfig(cliOpts.Shutdown))
	return nil
}

type ClientOption func(*ClientOptions)

func WithClientNoCheck() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Check = false
	}
}

func WithClientURL(url string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.URL = url
	}
}

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithClientFilter(filter string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithClientRegistryIDs(registryIDs ...string) ClientOption {
	return func(opts *ClientOptions) {
		if len(registryIDs) > 0 {
			opts.Consumer.RegistryIDs = registryIDs
		}
	}
}

func WithClientRegistry(opts ...registry.Option) ClientOption {
	regOpts := registry.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		cliOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

func WithClientShutdown(opts ...graceful_shutdown.Option) ClientOption {
	sdOpts := graceful_shutdown.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		cliOpts.Shutdown = sdOpts.Shutdown
	}
}

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

func WithClientClusterAvailable() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyAvailable
	}
}

func WithClientClusterBroadcast() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithClientClusterFailBack() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailback
	}
}

func WithClientClusterFailFast() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailfast
	}
}

func WithClientClusterFailOver() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailover
	}
}

func WithClientClusterFailSafe() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithClientClusterForking() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyForking
	}
}

func WithClientClusterZoneAware() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithClientClusterAdaptiveService() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = constant.ClusterKeyAdaptiveService
	}
}

func WithClientClusterStrategy(strategy string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Cluster = strategy
	}
}

// Deprecated：use triple.WithKeepAliveInterval()
// TODO: remove KeepAliveInterval and KeepAliveInterval in version 4.0.0
//
// If there is no other traffic on the connection, the ping will be sent, only works for 'tri' protocol with http2.
// A minimum value of 10s will be used instead to invoid 'too many pings'.If not set, default value is 10s.
func WithKeepAliveInterval(keepAliveInterval time.Duration) ClientOption {
	if keepAliveInterval < constant.MinKeepAliveInterval {
		keepAliveInterval = constant.MinKeepAliveInterval
	}
	return func(opts *ClientOptions) {
		opts.overallReference.KeepAliveInterval = keepAliveInterval.String()
	}
}

// Deprecated：use triple.WithKeepAliveTimeout()
// TODO: remove KeepAliveInterval and KeepAliveInterval in version 4.0.0
//
// WithKeepAliveTimeout is timeout after which the connection will be closed, only works for 'tri' protocol with http2
// If not set, default value is 20s.
func WithKeepAliveTimeout(keepAliveTimeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.KeepAliveTimeout = keepAliveTimeout.String()
	}
}

// ========== LoadBalance Strategy ==========

func WithClientLoadBalanceConsistentHashing() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithClientLoadBalanceLeastActive() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithClientLoadBalanceRandom() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithClientLoadBalanceRoundRobin() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithClientLoadBalanceP2C() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithClientLoadBalance(lb string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Loadbalance = lb
	}
}

func WithClientRetries(retries int) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Retries = strconv.Itoa(retries)
	}
}

// is this needed?
func WithClientGroup(group string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Group = group
	}
}

// is this needed?
func WithClientVersion(version string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Version = version
	}
}

func WithClientSerializationJSON() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Serialization = constant.JSONSerialization
	}
}

func WithClientSerialization(ser string) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Serialization = ser
	}
}

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

func WithClientParams(params map[string]string) ClientOption {
	return func(opts *ClientOptions) {
		if len(params) <= 0 {
			return
		}
		opts.overallReference.Params = params
	}
}

func WithClientParam(k, v string) ClientOption {
	return func(opts *ClientOptions) {
		if opts.overallReference.Params == nil {
			opts.overallReference.Params = make(map[string]string, 8)
		}
		opts.overallReference.Params[k] = v
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

func WithClientSticky() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.Sticky = true
	}
}

// ========== Protocol to consume ==========

func WithClientProtocolDubbo() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.DubboProtocol
	}
}

func WithClientProtocolTriple() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.TriProtocol
	}
}

func WithClientProtocolJsonRPC() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.JSONRPCProtocol
	}
}

func WithClientProtocol(opts ...protocol.ClientOption) ClientOption {
	proOpts := protocol.NewClientOptions(opts...)

	return func(srvOpts *ClientOptions) {
		if srvOpts.overallReference.ProtocolClientConfig == nil {
			srvOpts.overallReference.ProtocolClientConfig = new(global.ClientProtocolConfig)
		}
		srvOpts.overallReference.ProtocolClientConfig = proOpts.ProtocolClient
	}
}

func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.RequestTimeout = timeout.String()
	}
}

func WithClientForceTag() ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.ForceTag = true
	}
}

func WithClientMeshProviderPort(port int) ClientOption {
	return func(opts *ClientOptions) {
		opts.overallReference.MeshProviderPort = port
	}
}

func SetClientRegistries(regs map[string]*global.RegistryConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Registries = regs
	}
}

func SetApplication(application *global.ApplicationConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Application = application
	}
}

func SetClientConsumer(consumer *global.ConsumerConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer = consumer
	}
}

func SetClientShutdown(shutdown *global.ShutdownConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Shutdown = shutdown
	}
}

func SetClientMetrics(metrics *global.MetricsConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Metrics = metrics
	}
}

func SetClientOtel(otel *global.OtelConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Otel = otel
	}
}

func SetClientTLS(tls *global.TLSConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.TLS = tls
	}
}

// todo: need to be consistent with MethodConfig
type CallOptions struct {
	RequestTimeout string
	Retries        string
}

type CallOption func(*CallOptions)

func newDefaultCallOptions() *CallOptions {
	return &CallOptions{}
}

// WithCallRequestTimeout the maximum waiting time for one specific call, only works for 'tri' and 'dubbo' protocol
func WithCallRequestTimeout(timeout time.Duration) CallOption {
	return func(opts *CallOptions) {
		opts.RequestTimeout = timeout.String()
	}
}

// WithCallRetries the maximum retry times on request failure for one specific call, only works for 'tri' and 'dubbo' protocol
func WithCallRetries(retries int) CallOption {
	return func(opts *CallOptions) {
		opts.Retries = strconv.Itoa(retries)
	}
}

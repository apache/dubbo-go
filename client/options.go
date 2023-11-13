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
	"reflect"
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
	"dubbo.apache.org/dubbo-go/v3/common/dubboutil"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/proxy"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type ReferenceOptions struct {
	Reference  *global.ReferenceConfig
	cliOpts    *ClientOptions
	Registries map[string]*global.RegistryConfig

	pxy          *proxy.Proxy
	id           string
	invoker      protocol.Invoker
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
	}
}

func (refOpts *ReferenceOptions) init(cli *Client, opts ...ReferenceOption) error {
	for _, opt := range opts {
		opt(refOpts)
	}
	if err := defaults.Set(refOpts); err != nil {
		return err
	}

	refOpts.cliOpts = cli.cliOpts
	dubboutil.CopyFields(reflect.ValueOf(refOpts.cliOpts.Consumer).Elem(), reflect.ValueOf(refOpts.Reference).Elem())

	ref := refOpts.Reference

	// init method
	methods := ref.Methods
	if length := len(methods); length > 0 {
		refOpts.methodsCompat = make([]*config.MethodConfig, length)
		for i, method := range methods {
			refOpts.methodsCompat[i] = compatMethodConfig(method)
			if err := refOpts.methodsCompat[i].Init(); err != nil {
				return err
			}
		}
	}

	// init application
	application := refOpts.cliOpts.Application
	if application != nil {
		refOpts.applicationCompat = compatApplicationConfig(application)
		if err := refOpts.applicationCompat.Init(); err != nil {
			return err
		}
		refOpts.metaDataType = refOpts.applicationCompat.MetadataType
		if ref.Group == "" {
			ref.Group = refOpts.applicationCompat.Group
		}
		if ref.Version == "" {
			ref.Version = refOpts.applicationCompat.Version
		}
	}
	// init cluster
	if ref.Cluster == "" {
		ref.Cluster = "failover"
	}

	// todo(DMwangnima): move to registry package
	// init registries
	var emptyRegIDsFlag bool
	if ref.RegistryIDs == nil || len(ref.RegistryIDs) <= 0 {
		emptyRegIDsFlag = true
	}
	regs := refOpts.Registries
	if regs != nil {
		refOpts.registriesCompat = make(map[string]*config.RegistryConfig)
		for key, reg := range regs {
			refOpts.registriesCompat[key] = compatRegistryConfig(reg)
			if err := refOpts.registriesCompat[key].Init(); err != nil {
				return err
			}
			if emptyRegIDsFlag {
				ref.RegistryIDs = append(ref.RegistryIDs, key)
			}
		}
	}
	ref.RegistryIDs = commonCfg.TranslateIds(ref.RegistryIDs)

	// init graceful_shutdown
	graceful_shutdown.Init(graceful_shutdown.SetShutdown_Config(refOpts.cliOpts.Shutdown))

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

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithFilter(filter string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithRegistryIDs(registryIDs []string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		if len(registryIDs) > 0 {
			opts.Reference.RegistryIDs = registryIDs
		}
	}
}

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

func WithLoadBalanceXDSRingHash() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyLeastActive
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

func WithJSON() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Serialization = constant.JSONSerialization
	}
}

func WithProvidedBy(providedBy string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ProvidedBy = providedBy
	}
}

// todo(DMwangnima): implement this functionality
//func WithAsync() ReferenceOption {
//	return func(opts *ReferenceOptions) {
//		opts.Reference.Async = true
//	}
//}

func WithParams(params map[string]string) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Params = params
	}
}

// todo(DMwangnima): implement this functionality
//func WithGeneric(generic bool) ReferenceOption {
//	return func(opts *ReferenceOptions) {
//		if generic {
//			opts.Reference.Generic = "true"
//		} else {
//			opts.Reference.Generic = "false"
//		}
//	}
//}

func WithSticky(sticky bool) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Sticky = sticky
	}
}

// ========== Protocol to consume ==========

func WithProtocolDubbo() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = constant.Dubbo
	}
}

func WithProtocolTriple() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = "tri"
	}
}

func WithProtocolJsonRPC() ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.Protocol = "jsonrpc"
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

func WithForce(force bool) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.ForceTag = force
	}
}

func WithMeshProviderPort(port int) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference.MeshProviderPort = port
	}
}

// ---------- For framework ----------
// These functions should not be invoked by users

func SetRegistries(regs map[string]*global.RegistryConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Registries = regs
	}
}

func SetReference(reference *global.ReferenceConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference = reference
	}
}

type ClientOptions struct {
	Consumer    *global.ConsumerConfig
	Application *global.ApplicationConfig
	Registries  map[string]*global.RegistryConfig
	Shutdown    *global.ShutdownConfig
}

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Consumer:    global.DefaultConsumerConfig(),
		Registries:  make(map[string]*global.RegistryConfig),
		Application: global.DefaultApplicationConfig(),
		Shutdown:    global.DefaultShutdownConfig(),
	}
}

func (cliOpts *ClientOptions) init(opts ...ClientOption) error {
	for _, opt := range opts {
		opt(cliOpts)
	}
	if err := defaults.Set(cliOpts); err != nil {
		return err
	}
	return nil
}

type ClientOption func(*ClientOptions)

func WithClientCheck() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Check = true
	}
}

func WithClientURL(url string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.URL = url
	}
}

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithClientFilter(filter string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithClientRegistryIDs(registryIDs []string) ClientOption {
	return func(opts *ClientOptions) {
		if len(registryIDs) > 0 {
			opts.Consumer.RegistryIDs = registryIDs
		}
	}
}

func WithClientRegistry(opts ...registry.Option) ClientOption {
	regOpts := registry.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		if cliOpts.Registries == nil {
			cliOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		cliOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

//func WithClientShutdown(opts ...graceful_shutdown.Option) ClientOption {
//	sdOpts := graceful_shutdown.NewOptions(opts...)
//
//	return func(cliOpts *ClientOptions) {
//		cliOpts.Shutdown = sdOpts.Shutdown
//	}
//}

// ========== Cluster Strategy ==========

func WithClientClusterAvailable() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyAvailable
	}
}

func WithClientClusterBroadcast() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithClientClusterFailBack() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailback
	}
}

func WithClientClusterFailFast() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailfast
	}
}

func WithClientClusterFailOver() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailover
	}
}

func WithClientClusterFailSafe() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithClientClusterForking() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyForking
	}
}

func WithClientClusterZoneAware() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithClientClusterAdaptiveService() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// ========== LoadBalance Strategy ==========

func WithClientLoadBalanceConsistentHashing() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithClientLoadBalanceLeastActive() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithClientLoadBalanceRandom() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithClientLoadBalanceRoundRobin() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithClientLoadBalanceP2C() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithClientLoadBalanceXDSRingHash() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithClientRetries(retries int) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Retries = strconv.Itoa(retries)
	}
}

func WithClientGroup(group string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Group = group
	}
}

func WithClientVersion(version string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Version = version
	}
}

func WithClientJSON() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Serialization = constant.JSONSerialization
	}
}

func WithClientProvidedBy(providedBy string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.ProvidedBy = providedBy
	}
}

// todo(DMwangnima): implement this functionality
//func WithAsync() ClientOption {
//	return func(opts *ClientOptions) {
//		opts.Consumer.Async = true
//	}
//}

func WithClientParams(params map[string]string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Params = params
	}
}

// todo(DMwangnima): implement this functionality
//func WithClientGeneric(generic bool) ClientOption {
//	return func(opts *ClientOptions) {
//		if generic {
//			opts.Consumer.Generic = "true"
//		} else {
//			opts.Consumer.Generic = "false"
//		}
//	}
//}

func WithClientSticky(sticky bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Sticky = sticky
	}
}

// ========== Protocol to consume ==========

func WithClientProtocolDubbo() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.Dubbo
	}
}

func WithClientProtocolTriple() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = "tri"
	}
}

func WithClientProtocolJsonRPC() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = "jsonrpc"
	}
}

func WithClientProtocol(protocol string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = protocol
	}
}

func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.RequestTimeout = timeout.String()
	}
}

func WithClientForce(force bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.ForceTag = force
	}
}

func WithClientMeshProviderPort(port int) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.MeshProviderPort = port
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

func SetConsumer(consumer *global.ConsumerConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer = consumer
	}
}

func SetShutdown(shutdown *global.ShutdownConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Shutdown = shutdown
	}
}

// todo: need to be consistent with MethodConfig
type CallOptions struct {
	RequestTimeout string
	Retries        string
}

type CallOption func(*CallOptions)

func newDefaultCallOptions() *CallOptions {
	return &CallOptions{
		RequestTimeout: "",
		Retries:        "",
	}
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

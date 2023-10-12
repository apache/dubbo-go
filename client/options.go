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
	"dubbo.apache.org/dubbo-go/v3/proxy"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type ClientOptions struct {
	Application *global.ApplicationConfig
	Consumer    *global.ConsumerConfig
	Reference   *global.ReferenceConfig
	Registries  map[string]*global.RegistryConfig
	Shutdown    *global.ShutdownConfig

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

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Application: global.DefaultApplicationConfig(),
		Consumer:    global.DefaultConsumerConfig(),
		Reference:   global.DefaultReferenceConfig(),
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

	ref := cliOpts.Reference

	// init method
	methods := ref.Methods
	if length := len(methods); length > 0 {
		cliOpts.methodsCompat = make([]*config.MethodConfig, length)
		for i, method := range methods {
			cliOpts.methodsCompat[i] = compatMethodConfig(method)
			if err := cliOpts.methodsCompat[i].Init(); err != nil {
				return err
			}
		}

	}

	// init application
	application := cliOpts.Application
	if application != nil {
		cliOpts.applicationCompat = compatApplicationConfig(application)
		if err := cliOpts.applicationCompat.Init(); err != nil {
			return err
		}
		cliOpts.metaDataType = cliOpts.applicationCompat.MetadataType
		if ref.Group == "" {
			ref.Group = cliOpts.applicationCompat.Group
		}
		if ref.Version == "" {
			ref.Version = cliOpts.applicationCompat.Version
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
	regs := cliOpts.Registries
	if regs != nil {
		cliOpts.registriesCompat = make(map[string]*config.RegistryConfig)
		for key, reg := range regs {
			cliOpts.registriesCompat[key] = compatRegistryConfig(reg)
			if err := cliOpts.registriesCompat[key].Init(); err != nil {
				return err
			}
			if emptyRegIDsFlag {
				ref.RegistryIDs = append(ref.RegistryIDs, key)
			}
		}
	}
	ref.RegistryIDs = commonCfg.TranslateIds(ref.RegistryIDs)

	// init graceful_shutdown
	graceful_shutdown.Init(graceful_shutdown.WithShutdown_Config(cliOpts.Shutdown))

	return commonCfg.Verify(cliOpts)
}

type ClientOption func(*ClientOptions)

// ---------- For user ----------

func WithCheck() ClientOption {
	return func(opts *ClientOptions) {
		check := true
		opts.Reference.Check = &check
	}
}

func WithURL(url string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.URL = url
	}
}

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithFilter(filter string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithRegistryIDs(registryIDs []string) ClientOption {
	return func(opts *ClientOptions) {
		if len(registryIDs) > 0 {
			opts.Reference.RegistryIDs = registryIDs
		}
	}
}

func WithRegistry(opts ...registry.Option) ClientOption {
	regOpts := registry.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		if cliOpts.Registries == nil {
			cliOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		cliOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

func WithShutdown(opts ...graceful_shutdown.Option) ClientOption {
	sdOpts := graceful_shutdown.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		cliOpts.Shutdown = sdOpts.Shutdown
	}
}

// ========== Cluster Strategy ==========

func WithClusterAvailable() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAvailable
	}
}

func WithClusterBroadcast() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithClusterFailBack() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailback
	}
}

func WithClusterFailFast() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailfast
	}
}

func WithClusterFailOver() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailover
	}
}

func WithClusterFailSafe() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithClusterForking() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyForking
	}
}

func WithClusterZoneAware() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithClusterAdaptiveService() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// ========== LoadBalance Strategy ==========

func WithLoadBalanceConsistentHashing() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithLoadBalanceLeastActive() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithLoadBalanceRandom() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithLoadBalanceRoundRobin() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithLoadBalanceP2C() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithLoadBalanceXDSRingHash() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithRetries(retries int) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Retries = strconv.Itoa(retries)
	}
}

func WithGroup(group string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Group = group
	}
}

func WithVersion(version string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Version = version
	}
}

func WithJSON() ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Serialization = constant.JSONSerialization
	}
}

func WithProvidedBy(providedBy string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.ProvidedBy = providedBy
	}
}

// todo(DMwangnima): implement this functionality
//func WithAsync() ClientOption {
//	return func(opts *ClientOptions) {
//		opts.Reference.Async = true
//	}
//}

func WithParams(params map[string]string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Params = params
	}
}

// todo(DMwangnima): implement this functionality
//func WithGeneric(generic bool) ClientOption {
//	return func(opts *ClientOptions) {
//		if generic {
//			opts.Reference.Generic = "true"
//		} else {
//			opts.Reference.Generic = "false"
//		}
//	}
//}

func WithSticky(sticky bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Sticky = sticky
	}
}

func WithRequestTimeout(timeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.RequestTimeout = timeout.String()
	}
}

func WithForce(force bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.ForceTag = force
	}
}

func WithTracingKey(tracingKey string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.TracingKey = tracingKey
	}
}

func WithMeshProviderPort(port int) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.MeshProviderPort = port
	}
}

// ---------- For framework ----------
// These functions should not be invoked by users

func SetRegistries(regs map[string]*global.RegistryConfig) ClientOption {
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

func SetReference(reference *global.ReferenceConfig) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference = reference
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

func WithCallRequestTimeout(timeout string) CallOption {
	return func(opts *CallOptions) {
		opts.RequestTimeout = timeout
	}
}

func WithCallRetries(retries string) CallOption {
	return func(opts *CallOptions) {
		opts.Retries = retries
	}
}

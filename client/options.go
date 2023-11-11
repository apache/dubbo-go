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

type ReferenceOptions struct {
	Reference   *global.ReferenceConfig
	Application *global.ApplicationConfig
	Registries  map[string]*global.RegistryConfig
	Shutdown    *global.ShutdownConfig
	cliOpts     *ClientOptions

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
		Application: global.DefaultApplicationConfig(),
		Reference:   global.DefaultReferenceConfig(),
		Shutdown:    global.DefaultShutdownConfig(),
	}
}

func (refOpts *ReferenceOptions) init(cli *Client, opts ...ReferenceOption) error {
	for _, opt := range opts {
		opt(refOpts)
	}
	if err := defaults.Set(refOpts); err != nil {
		return err
	}

	refOpts.setRefWithCon(cli.cliOpts.Consumer)

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
	application := refOpts.Application
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
	if regs == nil {
		regs = cli.cliOpts.Registries
	}
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
	graceful_shutdown.Init(graceful_shutdown.SetShutdown_Config(refOpts.Shutdown))

	return commonCfg.Verify(refOpts)
}

func (refOpts *ReferenceOptions) setRefWithCon(consumer *global.ConsumerConfig) {
	// set fields of refOpts.Reference with consumer as default value
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

	return func(cliOpts *ReferenceOptions) {
		if cliOpts.Registries == nil {
			cliOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		cliOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

func WithShutdown(opts ...graceful_shutdown.Option) ReferenceOption {
	sdOpts := graceful_shutdown.NewOptions(opts...)

	return func(cliOpts *ReferenceOptions) {
		cliOpts.Shutdown = sdOpts.Shutdown
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

func SetApplication(application *global.ApplicationConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Application = application
	}
}

func SetReference(reference *global.ReferenceConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
		opts.Reference = reference
	}
}

func SetShutdown(shutdown *global.ShutdownConfig) ReferenceOption {
	return func(opts *ReferenceOptions) {
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

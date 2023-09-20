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
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/proxy"
)

type ClientOptions struct {
	Application *global.ApplicationConfig
	Consumer    *global.ConsumerConfig
	Reference   *global.ReferenceConfig
	Registries  map[string]*global.RegistryConfig

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
		Reference: global.DefaultReferenceConfig(),
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
	regs := cliOpts.Registries
	if regs != nil {
		cliOpts.registriesCompat = make(map[string]*config.RegistryConfig)
		for key, reg := range regs {
			cliOpts.registriesCompat[key] = compatRegistryConfig(reg)
			if err := cliOpts.registriesCompat[key].Init(); err != nil {
				return err
			}
		}
	}
	ref.RegistryIDs = commonCfg.TranslateIds(ref.RegistryIDs)

	return commonCfg.Verify(cliOpts)
}

type ClientOption func(*ClientOptions)

// ---------- For user ----------

func WithCheck(check bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Check = &check
	}
}

func WithURL(url string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.URL = url
	}
}

func WithFilter(filter string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Filter = filter
	}
}

func WithProtocol(protocol string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Protocol = protocol
	}
}

func WithRegistryIDs(registryIDs []string) ClientOption {
	return func(opts *ClientOptions) {
		if len(registryIDs) > 0 {
			opts.Reference.RegistryIDs = registryIDs
		}
	}
}

func WithCluster(cluster string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Cluster = cluster
	}
}

func WithLoadBalance(loadBalance string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Loadbalance = loadBalance
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

func WithSerialization(serialization string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Serialization = serialization
	}
}

func WithProviderBy(providedBy string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.ProvidedBy = providedBy
	}
}

func WithAsync(async bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Async = async
	}
}

func WithParams(params map[string]string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Params = params
	}
}

func WithGeneric(generic string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Generic = generic
	}
}

func WithSticky(sticky bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.Sticky = sticky
	}
}

func WithRequestTimeout(timeout string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Reference.RequestTimeout = timeout
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

func WithRegistryConfig(key string, opts ...global.RegistryOption) ClientOption {
	regCfg := new(global.RegistryConfig)
	for _, opt := range opts {
		opt(regCfg)
	}

	return func(opts *ClientOptions) {
		if opts.Registries == nil {
			opts.Registries = make(map[string]*global.RegistryConfig)
		}
		opts.Registries[key] = regCfg
	}
}

func WithApplicationConfig(opts ...global.ApplicationOption) ClientOption {
	appCfg := new(global.ApplicationConfig)
	for _, opt := range opts {
		opt(appCfg)
	}

	return func(opts *ClientOptions) {
		opts.Application = appCfg
	}
}

func WithConsumerConfig(opts ...global.ConsumerOption) ClientOption {
	conCfg := new(global.ConsumerConfig)
	for _, opt := range opts {
		opt(conCfg)
	}

	return func(opts *ClientOptions) {
		opts.Consumer = conCfg
	}
}

func WithReferenceConfig(opts ...global.ReferenceOption) ClientOption {
	refCfg := new(global.ReferenceConfig)
	for _, opt := range opts {
		opt(refCfg)
	}

	return func(opts *ClientOptions) {
		opts.Reference = refCfg
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

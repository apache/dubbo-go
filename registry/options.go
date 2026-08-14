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

package registry

import (
	"strings"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// Options wraps the registry configuration. ID distinguishes this
// registry from others when multiple registries are configured.
type Options struct {
	Registry *global.RegistryConfig

	ID string
}

func defaultOptions() *Options {
	return &Options{
		Registry: global.DefaultRegistryConfig(),
	}
}

// NewOptions builds Options from the defaults and the given options.
// A registry protocol must be specified, otherwise it panics.
func NewOptions(opts ...Option) *Options {
	defOpts := defaultOptions()
	for _, opt := range opts {
		opt(defOpts)
	}

	if defOpts.Registry.Protocol == "" {
		panic("Please specify registry, eg. WithZookeeper()")
	}
	if defOpts.ID == "" {
		defOpts.ID = defOpts.Registry.Protocol
	}

	return defOpts
}

// Option configures the registry options.
type Option func(*Options)

// WithEtcdV3 uses etcd v3 as the registry backend.
func WithEtcdV3() Option {
	return func(opts *Options) {
		opts.Registry.Protocol = constant.EtcdV3Key
	}
}

// WithNacos uses Nacos as the registry backend.
func WithNacos() Option {
	return func(opts *Options) {
		opts.Registry.Protocol = constant.NacosKey
	}
}

// WithPolaris uses Polaris as the registry backend.
func WithPolaris() Option {
	return func(opts *Options) {
		opts.Registry.Protocol = constant.PolarisKey
	}
}

// WithZookeeper uses ZooKeeper as the registry backend.
func WithZookeeper() Option {
	return func(opts *Options) {
		opts.Registry.Protocol = constant.ZookeeperKey
	}
}

// WithRegistry sets the registry backend by protocol name,
// e.g. WithRegistry("zookeeper").
func WithRegistry(r string) Option {
	return func(opts *Options) {
		opts.Registry.Protocol = r
	}
}

// WithID specifies the id of registry.Options. Then you could configure client.WithRegistryIDs and
// server.WithServer_RegistryIDs to specify which registry you need to use in multi-registries scenario.
func WithID(id string) Option {
	return func(opts *Options) {
		opts.ID = id
	}
}

// WithTimeout sets the timeout of operations against the registry.
func WithTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Registry.Timeout = timeout.String()
	}
}

// WithGroup sets the registry group. It is often used to isolate
// environments, like dev and prod, that share one registry.
func WithGroup(group string) Option {
	return func(opts *Options) {
		opts.Registry.Group = group
	}
}

// WithNamespace sets the namespace of the registry. Only some
// registries support it, e.g. Nacos.
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Registry.Namespace = namespace
	}
}

// WithTTL sets the TTL of registered instances, after which the
// registry considers them expired.
func WithTTL(ttl time.Duration) Option {
	return func(opts *Options) {
		opts.Registry.TTL = ttl.String()
	}
}

// WithAddress sets the address of the registry. When the address
// carries a scheme, like nacos://127.0.0.1:8848, the protocol is
// derived from it as well.
func WithAddress(address string) Option {
	return func(opts *Options) {
		if i := strings.Index(address, "://"); i > 0 {
			opts.Registry.Protocol = address[0:i]
		}
		opts.Registry.Address = address
	}
}

// WithUsername sets the username used to authenticate with the registry.
func WithUsername(name string) Option {
	return func(opts *Options) {
		opts.Registry.Username = name
	}
}

// WithPassword sets the password used to authenticate with the registry.
func WithPassword(password string) Option {
	return func(opts *Options) {
		opts.Registry.Password = password
	}
}

// WithSimplified enables the simplified registration mode, which
// registers less metadata to the registry.
func WithSimplified() Option {
	return func(opts *Options) {
		opts.Registry.Simplified = true
	}
}

// WithPreferred marks the registry as preferred, so it is always
// used first when subscribing to multiple registries.
func WithPreferred() Option {
	return func(opts *Options) {
		opts.Registry.Preferred = true
	}
}

// WithZone sets the zone of the registry, usually to isolate
// traffic by region.
func WithZone(zone string) Option {
	return func(opts *Options) {
		opts.Registry.Zone = zone
	}
}

// WithWeight sets the weight of the registry, which affects the
// traffic distribution among registries. It is ignored when a
// preferred registry is configured.
func WithWeight(weight int64) Option {
	return func(opts *Options) {
		opts.Registry.Weight = weight
	}
}

// WithParams sets extra params of the registry, which are passed
// through to the registry implementation.
func WithParams(params map[string]string) Option {
	return func(opts *Options) {
		opts.Registry.Params = params
	}
}

// WithRegisterServiceAndInterface registers both services and
// interfaces with the registry.
func WithRegisterServiceAndInterface() Option {
	return func(opts *Options) {
		opts.Registry.RegistryType = constant.RegistryTypeAll
	}
}

// WithRegisterInterface registers interfaces only.
func WithRegisterInterface() Option {
	return func(opts *Options) {
		opts.Registry.RegistryType = constant.RegistryTypeInterface
	}
}

// WithRegisterService registers services only.
func WithRegisterService() Option {
	return func(opts *Options) {
		opts.Registry.RegistryType = constant.RegistryTypeService
	}
}

// WithoutUseAsMetaReport disables the registry being used as the
// metadata report.
func WithoutUseAsMetaReport() Option {
	return func(opts *Options) {
		opts.Registry.UseAsMetaReport = "false"
	}
}

// WithoutUseAsConfigCenter disables the registry being used as the
// config center.
func WithoutUseAsConfigCenter() Option {
	return func(opts *Options) {
		opts.Registry.UseAsConfigCenter = "false"
	}
}

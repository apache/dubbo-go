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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Registry *global.RegistryConfig
}

func DefaultOptions() *Options {
	return &Options{Registry: global.DefaultRegistryConfig()}
}

type Option func(*Options)

func WithEtcdV3() Option {
	return func(opts *Options) {
		// todo(DMwangnima): move etcdv3 to constant
		opts.Registry.Protocol = "etcdv3"
	}
}

func WithNacos() Option {
	return func(opts *Options) {
		opts.Registry.Protocol = constant.NacosKey
	}
}

func WithPolaris() Option {
	return func(opts *Options) {
		opts.Registry.Protocol = constant.PolarisKey
	}
}

func WithXDS() Option {
	return func(opts *Options) {
		opts.Registry.Protocol = constant.XDSRegistryKey
	}
}

func WithZookeeper() Option {
	return func(opts *Options) {
		// todo(DMwangnima): move zookeeper to constant
		opts.Registry.Protocol = "zookeeper"
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Registry.Timeout = timeout.String()
	}
}

func WithGroup(group string) Option {
	return func(opts *Options) {
		opts.Registry.Group = group
	}
}

func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Registry.Namespace = namespace
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(opts *Options) {
		opts.Registry.TTL = ttl.String()
	}
}

func WithAddress(address string) Option {
	return func(opts *Options) {
		opts.Registry.Address = address
	}
}

func WithUsername(name string) Option {
	return func(opts *Options) {
		opts.Registry.Username = name
	}
}

func WithPassword(password string) Option {
	return func(opts *Options) {
		opts.Registry.Password = password
	}
}

func WithSimplified() Option {
	return func(opts *Options) {
		opts.Registry.Simplified = true
	}
}

func WithPreferred() Option {
	return func(opts *Options) {
		opts.Registry.Preferred = true
	}
}

func WithZone(zone string) Option {
	return func(opts *Options) {
		opts.Registry.Zone = zone
	}
}

func WithWeight(weight int64) Option {
	return func(opts *Options) {
		opts.Registry.Weight = weight
	}
}

func WithParams(params map[string]string) Option {
	return func(opts *Options) {
		opts.Registry.Params = params
	}
}

func WithRegisterServiceAndInterface() Option {
	return func(opts *Options) {
		opts.Registry.RegistryType = constant.RegistryTypeAll
	}
}

func WithRegisterInterface() Option {
	return func(opts *Options) {
		opts.Registry.RegistryType = constant.RegistryTypeInterface
	}
}

func WithoutUseAsMetaReport() Option {
	return func(opts *Options) {
		opts.Registry.UseAsMetaReport = false
	}
}

func WithoutUseAsConfigCenter() Option {
	return func(opts *Options) {
		opts.Registry.UseAsConfigCenter = false
	}
}

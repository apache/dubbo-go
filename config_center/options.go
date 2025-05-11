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

package config_center

import (
	"strconv"
	"strings"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/constant/file"
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Center *global.CenterConfig
}

func defaultOptions() *Options {
	return &Options{Center: global.DefaultCenterConfig()}
}

func NewOptions(opts ...Option) *Options {
	centerOptions := defaultOptions()
	for _, opt := range opts {
		opt(centerOptions)
	}
	return centerOptions
}

type Option func(*Options)

func WithZookeeper() Option {
	return func(opts *Options) {
		opts.Center.Protocol = constant.ZookeeperKey
	}
}

func WithNacos() Option {
	return func(opts *Options) {
		opts.Center.Protocol = constant.NacosKey
	}
}

func WithApollo() Option {
	return func(opts *Options) {
		opts.Center.Protocol = constant.ApolloKey
	}
}

func WithConfigCenter(cc string) Option {
	return func(opts *Options) {
		opts.Center.Protocol = cc
	}
}

func WithAddress(address string) Option {
	return func(opts *Options) {
		if i := strings.Index(address, "://"); i > 0 {
			opts.Center.Protocol = address[0:i]
		}
		opts.Center.Address = address
	}
}

func WithDataID(id string) Option {
	return func(opts *Options) {
		opts.Center.DataId = id
	}
}

func WithCluster(cluster string) Option {
	return func(opts *Options) {
		opts.Center.Cluster = cluster
	}
}

func WithGroup(group string) Option {
	return func(opts *Options) {
		opts.Center.Group = group
	}
}

func WithUsername(username string) Option {
	return func(opts *Options) {
		opts.Center.Username = username
	}
}

func WithPassword(password string) Option {
	return func(opts *Options) {
		opts.Center.Password = password
	}
}

func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Center.Namespace = namespace
	}
}

func WithAppID(id string) Option {
	return func(opts *Options) {
		opts.Center.AppID = id
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Center.Timeout = strconv.Itoa(int(timeout.Milliseconds()))
	}
}

func WithParams(params map[string]string) Option {
	return func(opts *Options) {
		opts.Center.Params = params
	}
}

func WithFile() Option {
	return func(opts *Options) {
		opts.Center.Protocol = constant.FileKey
	}
}

func WithFileExtJson() Option {
	return func(opts *Options) {
		opts.Center.FileExtension = string(file.JSON)
	}
}

func WithFileExtToml() Option {
	return func(opts *Options) {
		opts.Center.FileExtension = string(file.TOML)
	}
}

func WithFileExtYaml() Option {
	return func(opts *Options) {
		opts.Center.FileExtension = string(file.YAML)
	}
}

func WithFileExtYml() Option {
	return func(opts *Options) {
		opts.Center.FileExtension = string(file.YML)
	}
}

func WithFileExtProperties() Option {
	return func(opts *Options) {
		opts.Center.FileExtension = string(file.PROPERTIES)
	}
}

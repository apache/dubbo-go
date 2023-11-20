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

package metadata

import (
	"strconv"
	"strings"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Metadata *global.MetadataReportConfig
}

func defaultOptions() *Options {
	return &Options{Metadata: global.DefaultMetadataReportConfig()}
}

func NewOptions(opts ...Option) *Options {
	metaOptions := defaultOptions()
	for _, opt := range opts {
		opt(metaOptions)
	}
	return metaOptions
}

type Option func(*Options)

func WithZookeeper() Option {
	return func(opts *Options) {
		opts.Metadata.Protocol = constant.ZookeeperKey
	}
}

func WithNacos() Option {
	return func(opts *Options) {
		opts.Metadata.Protocol = constant.NacosKey
	}
}

func WithEtcdV3() Option {
	return func(opts *Options) {
		opts.Metadata.Protocol = constant.EtcdV3Key
	}
}

func WithAddress(address string) Option {
	return func(opts *Options) {
		if i := strings.Index(address, "://"); i > 0 {
			opts.Metadata.Protocol = address[0:i]
		}
		opts.Metadata.Address = address
	}
}

func WithUsername(username string) Option {
	return func(opts *Options) {
		opts.Metadata.Username = username
	}
}

func WithPassword(password string) Option {
	return func(opts *Options) {
		opts.Metadata.Password = password
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Metadata.Timeout = strconv.Itoa(int(timeout.Milliseconds()))
	}
}

func WithGroup(group string) Option {
	return func(opts *Options) {
		opts.Metadata.Group = group
	}
}

func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Metadata.Namespace = namespace
	}
}

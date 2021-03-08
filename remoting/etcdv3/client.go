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

package etcdv3

import (
	"time"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

const (
	// ConnDelay connection delay
	ConnDelay = 3
	// MaxFailTimes max failure times
	MaxFailTimes = 15
	// RegistryETCDV3Client client name
	RegistryETCDV3Client = "etcd registry"
	// metadataETCDV3Client client name
	MetadataETCDV3Client = "etcd metadata"
)

// nolint
type Options struct {
	name      string
	endpoints []string
	client    *gxetcd.Client
	timeout   time.Duration
	heartbeat int // heartbeat second
}

// Option will define a function of handling Options
type Option func(*Options)

// WithEndpoints sets etcd client endpoints
func WithEndpoints(endpoints ...string) Option {
	return func(opt *Options) {
		opt.endpoints = endpoints
	}
}

// WithName sets etcd client name
func WithName(name string) Option {
	return func(opt *Options) {
		opt.name = name
	}
}

// WithTimeout sets etcd client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(opt *Options) {
		opt.timeout = timeout
	}
}

// WithHeartbeat sets etcd client heartbeat
func WithHeartbeat(heartbeat int) Option {
	return func(opt *Options) {
		opt.heartbeat = heartbeat
	}
}

// ValidateClient validates client and sets options
func ValidateClient(container clientFacade, opts ...Option) error {
	options := &Options{
		heartbeat: 1, // default heartbeat
	}
	for _, opt := range opts {
		opt(options)
	}

	lock := container.ClientLock()
	lock.Lock()
	defer lock.Unlock()

	// new Client
	if container.Client() == nil {
		newClient, err := gxetcd.NewClient(options.name, options.endpoints, options.timeout, options.heartbeat)
		if err != nil {
			logger.Warnf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
				options.name, options.endpoints, options.timeout, err)
			return perrors.WithMessagef(err, "new client (address:%+v)", options.endpoints)
		}
		container.SetClient(newClient)
	}

	// Client lose connection with etcd server
	if container.Client().GetRawClient() == nil {
		newClient, err := gxetcd.NewClient(options.name, options.endpoints, options.timeout, options.heartbeat)
		if err != nil {
			logger.Warnf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
				options.name, options.endpoints, options.timeout, err)
			return perrors.WithMessagef(err, "new client (address:%+v)", options.endpoints)
		}
		container.SetClient(newClient)
	}

	return nil
}

//  nolint
func NewServiceDiscoveryClient(opts ...Option) *gxetcd.Client {
	options := &Options{
		heartbeat: 1, // default heartbeat
	}
	for _, opt := range opts {
		opt(options)
	}

	newClient, err := gxetcd.NewClient(options.name, options.endpoints, options.timeout, options.heartbeat)
	if err != nil {
		logger.Errorf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
			options.name, options.endpoints, options.timeout, err)
	}
	return newClient
}

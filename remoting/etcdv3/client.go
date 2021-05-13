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
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

// ValidateClient validates client and sets options
func ValidateClient(container clientFacade, opts ...gxetcd.Option) error {
	options := &gxetcd.Options{}
	for _, opt := range opts {
		opt(options)
	}
	lock := container.ClientLock()
	lock.Lock()
	defer lock.Unlock()

	// new Client
	if container.Client() == nil {
		newClient, err := gxetcd.NewClient(options.Name, options.Endpoints, options.Timeout, options.Heartbeat)
		if err != nil {
			logger.Warnf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
				options.Name, options.Endpoints, options.Timeout, err)
			return perrors.WithMessagef(err, "new client (address:%+v)", options.Endpoints)
		}
		container.SetClient(newClient)
	}

	// Client lose connection with etcd server
	if container.Client().GetRawClient() == nil {
		newClient, err := gxetcd.NewClient(options.Name, options.Endpoints, options.Timeout, options.Heartbeat)
		if err != nil {
			logger.Warnf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
				options.Name, options.Endpoints, options.Timeout, err)
			return perrors.WithMessagef(err, "new client (address:%+v)", options.Endpoints)
		}
		container.SetClient(newClient)
	}

	return nil
}

//  nolint
func NewServiceDiscoveryClient(opts ...gxetcd.Option) *gxetcd.Client {
	options := &gxetcd.Options{
		Heartbeat: 1, // default heartbeat
	}
	for _, opt := range opts {
		opt(options)
	}

	newClient, err := gxetcd.NewClient(options.Name, options.Endpoints, options.Timeout, options.Heartbeat)
	if err != nil {
		logger.Errorf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
			options.Name, options.Endpoints, options.Timeout, err)
	}
	return newClient
}

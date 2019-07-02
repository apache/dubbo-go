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

package consul

import (
	"strconv"
)

import (
	perrors "github.com/pkg/errors"
	consul "github.com/hashicorp/consul/api"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
)

func init() {
	extension.SetRegistry("consul", newConsulRegistry)
}

// Consul registry wraps the consul client, to
// register and subscribe service.
type consulRegistry struct {
	// Registry url.
	*common.URL

	// Consul client.
	client *consul.Client

	// Done field represents whether
	// consul registry is closed.
	done   chan struct{}
}

func newConsulRegistry(url *common.URL) (registry.Registry, error) {
	config := &consul.Config{Address: url.Location}
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}

	r := &consulRegistry{
		URL:    url,
		client: client,
		done:   make(chan struct{}, 1),
	}

	return r, nil
}

func (r *consulRegistry) Register(url common.URL) error {
	var err error

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if role == common.PROVIDER {
		err = r.register(url)
		if err != nil {
			return perrors.WithStack(err)
		}
	}
	return nil
}

func (r *consulRegistry) register(url common.URL) error {
	service, err := buildService(url)
	if err != nil {
		return err
	}
	return r.client.Agent().ServiceRegister(service)
}

func (r *consulRegistry) Subscribe(url common.URL) (registry.Listener, error) {
	var (
		listener registry.Listener
		err 	 error
	)

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if role == common.CONSUMER {
		listener, err = r.subscribe(url)
		if err != nil {
			return nil, err
		}
	}
	return listener, nil
}

func (r *consulRegistry) subscribe(url common.URL) (registry.Listener, error) {
	listener, err := newConsulListener(url)
	return listener, err
}

func (r *consulRegistry) GetUrl() common.URL {
	return *r.URL
}

func (r *consulRegistry) IsAvailable() bool {
	select {
	case <- r.done:
		return false
	default:
		return true
	}
}

func (r *consulRegistry) Destroy() {
	close(r.done)
}

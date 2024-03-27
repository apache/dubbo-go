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

package dubbo

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// TestIndependentConfig tests the configurations of the `instance`, `client`, and `server` are independent.
func TestIndependentConfig(t *testing.T) {
	// instance configuration
	ins, err := NewInstance(
		WithName("dubbo_test"),
		WithRegistry(
			registry.WithZookeeper(),
			registry.WithAddress("127.0.0.1:2181"),
		),
	)
	if err != nil {
		panic(err)
	}

	// client configuration, ensure that the `instance` configuration can be passed to the `client`.
	_, err = ins.NewClient(
		func(options *client.ClientOptions) {
			assert.Equal(t, "dubbo_test", options.Application.Name)
			options.Application.Name = "dubbo_test_client"
			assert.Equal(t, "dubbo_test_client", options.Application.Name)

			assert.Equal(t, "127.0.0.1:2181", options.Registries[constant.ZookeeperKey].Address)
			options.Registries[constant.ZookeeperKey].Address = "127.0.0.1:2182"
			assert.Equal(t, "127.0.0.1:2182", options.Registries[constant.ZookeeperKey].Address)
		},
	)
	if err != nil {
		panic(err)
	}

	// server configuration, ensure that the `instance` configuration can be passed to the `server`, and the
	// `instance` configuration is not affected by the `client` configuration.
	_, err = ins.NewServer(
		func(options *server.ServerOptions) {
			assert.Equal(t, "dubbo_test", options.Application.Name)
			options.Application.Name = "dubbo_test_server"
			assert.Equal(t, "dubbo_test_server", options.Application.Name)

			assert.Equal(t, "127.0.0.1:2181", options.Registries[constant.ZookeeperKey].Address)
			options.Registries[constant.ZookeeperKey].Address = "127.0.0.1:2183"
			assert.Equal(t, "127.0.0.1:2183", options.Registries[constant.ZookeeperKey].Address)
		},
	)
	if err != nil {
		panic(err)
	}

	// check client configuration again, ensure that the `instance` and `client` configuration is not affected
	// by the `server` configuration.
	_, err = ins.NewClient(
		func(options *client.ClientOptions) {
			assert.Equal(t, "dubbo_test", options.Application.Name)
			options.Application.Name = "dubbo_test_client"
			assert.Equal(t, "dubbo_test_client", options.Application.Name)

			assert.Equal(t, "127.0.0.1:2181", options.Registries[constant.ZookeeperKey].Address)
			options.Registries[constant.ZookeeperKey].Address = "127.0.0.1:2184"
			assert.Equal(t, "127.0.0.1:2184", options.Registries[constant.ZookeeperKey].Address)
		},
	)
	if err != nil {
		panic(err)
	}
}

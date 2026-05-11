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
	"maps"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	legacyconfig "dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/server"
)

type testRPCService struct {
	ref string
}

func (s *testRPCService) Reference() string {
	return s.ref
}

func cloneClientDefinitions(src map[string]*client.ClientDefinition) map[string]*client.ClientDefinition {
	return maps.Clone(src)
}

func cloneServiceDefinitions(src map[string]*server.ServiceDefinition) map[string]*server.ServiceDefinition {
	return maps.Clone(src)
}

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

func TestInstanceInitKeepsGlobalOnlyConfig(t *testing.T) {
	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.Shutdown = global.DefaultShutdownConfig()
		opts.Shutdown.ClosingInvokerExpireTime = "7s"
	})
	require.NoError(t, err)

	_, err = ins.NewServer(func(options *server.ServerOptions) {
		require.NotNil(t, options.Shutdown)
		assert.Equal(t, "7s", options.Shutdown.ClosingInvokerExpireTime)
	})
	require.NoError(t, err)
}

func TestInstanceInitKeepsGlobalOnlyConfigWithConfigCenter(t *testing.T) {
	resetDynamicConfiguration(t)

	const configCenterProtocol = "mock-global-init"
	extension.SetConfigCenterFactory(configCenterProtocol, func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{
			Content: `
dubbo:
  application:
    name: remote-app
`,
		}
	})

	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.ConfigCenter = &global.CenterConfig{
			Protocol:      configCenterProtocol,
			Address:       "127.0.0.1:8848",
			DataId:        "dubbo.yaml",
			Group:         "dubbo",
			FileExtension: "yaml",
		}
		opts.Shutdown = global.DefaultShutdownConfig()
		opts.Shutdown.ClosingInvokerExpireTime = "7s"
		opts.Protocols = map[string]*global.ProtocolConfig{
			constant.TriProtocol: {
				Name: constant.TriProtocol,
				Port: defaultTripleProtocolPort,
				TripleConfig: &global.TripleConfig{
					Cors: &global.CorsConfig{
						AllowOrigins: []string{"https://example.com"},
					},
					OpenAPI: &global.OpenAPIConfig{
						Enabled: true,
						Path:    "/dubbo/openapi/",
					},
				},
			},
		}
	})
	require.NoError(t, err)

	assert.Equal(t, "remote-app", ins.insOpts.Application.Name)
	assert.Equal(t, "7s", ins.insOpts.Shutdown.ClosingInvokerExpireTime)
	tri := ins.insOpts.Protocols[constant.TriProtocol]
	require.NotNil(t, tri)
	require.NotNil(t, tri.TripleConfig)
	require.NotNil(t, tri.TripleConfig.Cors)
	require.NotNil(t, tri.TripleConfig.OpenAPI)
	assert.Equal(t, []string{"https://example.com"}, tri.TripleConfig.Cors.AllowOrigins)
	assert.Equal(t, "/dubbo/openapi", tri.TripleConfig.OpenAPI.Path)
}

func TestInstanceInitDefaultsGlobalProtocolBeforeNewServer(t *testing.T) {
	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.Protocols = map[string]*global.ProtocolConfig{
			constant.TriProtocol: {},
		}
	})
	require.NoError(t, err)

	tri := ins.insOpts.Protocols[constant.TriProtocol]
	require.NotNil(t, tri)
	assert.Equal(t, constant.TriProtocol, tri.Name)
	assert.Equal(t, defaultTripleProtocolPort, tri.Port)

	_, err = ins.NewServer(func(options *server.ServerOptions) {
		tri := options.Protocols[constant.TriProtocol]
		require.NotNil(t, tri)
		assert.Equal(t, constant.TriProtocol, tri.Name)
		assert.Equal(t, defaultTripleProtocolPort, tri.Port)
	})
	require.NoError(t, err)
}

func TestInstanceInitTranslatesGlobalRegistryAddress(t *testing.T) {
	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.Registries = map[string]*global.RegistryConfig{
			"zk": {
				Address:         "zookeeper://127.0.0.1:2181",
				UseAsMetaReport: "false",
			},
		}
	})
	require.NoError(t, err)

	reg := ins.insOpts.Registries["zk"]
	require.NotNil(t, reg)
	assert.Equal(t, constant.ZookeeperKey, reg.Protocol)
	assert.Equal(t, "127.0.0.1:2181", reg.Address)
}

func TestInstanceInitMirrorsTLSConfigToCompatRootConfig(t *testing.T) {
	restoreCompatRootConfig(t)

	_, err := NewInstance(func(opts *InstanceOptions) {
		opts.TLSConfig = &global.TLSConfig{
			CACertFile:    "ca.pem",
			TLSCertFile:   "cert.pem",
			TLSKeyFile:    "key.pem",
			TLSServerName: "dubbo.example",
		}
	})
	require.NoError(t, err)

	root := legacyconfig.GetRootConfig()
	require.NotNil(t, root)
	require.NotNil(t, root.TLSConfig)
	assert.Equal(t, "ca.pem", root.TLSConfig.CACertFile)
	assert.Equal(t, "cert.pem", root.TLSConfig.TLSCertFile)
	assert.Equal(t, "key.pem", root.TLSConfig.TLSKeyFile)
	assert.Equal(t, "dubbo.example", root.TLSConfig.TLSServerName)
}

func TestSetProviderServiceRegistersByReference(t *testing.T) {
	proLock.Lock()
	original := cloneServiceDefinitions(providerServices)
	providerServices = make(map[string]*server.ServiceDefinition)
	proLock.Unlock()

	defer func() {
		proLock.Lock()
		providerServices = original
		proLock.Unlock()
	}()

	svc := &testRPCService{ref: "provider.test.Service"}
	SetProviderService(svc)

	proLock.RLock()
	defer proLock.RUnlock()
	def, ok := providerServices[svc.Reference()]
	require.True(t, ok)
	require.NotNil(t, def)
	assert.Equal(t, svc, def.Handler)
}

func TestGetConsumerConnectionFromConsumerServices(t *testing.T) {
	conLock.Lock()
	original := cloneClientDefinitions(consumerServices)
	consumerServices = make(map[string]*client.ClientDefinition)
	conLock.Unlock()

	defer func() {
		conLock.Lock()
		consumerServices = original
		conLock.Unlock()
	}()

	svc := &testRPCService{ref: "consumer.test.Service"}
	SetConsumerService(svc)

	conn, err := GetConsumerConnection(svc.Reference())
	require.Error(t, err)
	require.Nil(t, conn)

	expectedConn := &client.Connection{}
	conLock.Lock()
	consumerServices[svc.Reference()].SetConnection(expectedConn)
	conLock.Unlock()

	conn, err = GetConsumerConnection(svc.Reference())
	require.NoError(t, err)
	assert.Equal(t, expectedConn, conn)
}

func resetDynamicConfiguration(t *testing.T) {
	t.Helper()
	env := commonCfg.GetEnvInstance()
	original := env.GetDynamicConfiguration()
	env.SetDynamicConfiguration(nil)
	t.Cleanup(func() {
		env.SetDynamicConfiguration(original)
	})
}

func restoreCompatRootConfig(t *testing.T) {
	t.Helper()
	original := legacyconfig.GetRootConfig()
	if original == nil {
		return
	}
	originalValue := *original
	t.Cleanup(func() {
		legacyconfig.SetRootConfig(originalValue)
	})
}

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
				Port: constant.DefaultTripleProtocolPort,
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
	defaultOpenAPI := global.DefaultOpenAPIConfig()
	assert.Equal(t, defaultOpenAPI.InfoTitle, tri.TripleConfig.OpenAPI.InfoTitle)
	assert.Equal(t, defaultOpenAPI.InfoVersion, tri.TripleConfig.OpenAPI.InfoVersion)
	assert.Equal(t, defaultOpenAPI.DefaultConsumesMediaTypes, tri.TripleConfig.OpenAPI.DefaultConsumesMediaTypes)
	assert.Equal(t, defaultOpenAPI.DefaultProducesMediaTypes, tri.TripleConfig.OpenAPI.DefaultProducesMediaTypes)
	assert.Equal(t, defaultOpenAPI.DefaultHttpStatusCodes, tri.TripleConfig.OpenAPI.DefaultHttpStatusCodes)
	require.NotNil(t, tri.TripleConfig.OpenAPI.Settings)

	_, err = ins.NewServer(func(options *server.ServerOptions) {
		require.NotNil(t, options.Shutdown)
		assert.Equal(t, "7s", options.Shutdown.ClosingInvokerExpireTime)

		tri := options.Protocols[constant.TriProtocol]
		require.NotNil(t, tri)
		require.NotNil(t, tri.TripleConfig)
		require.NotNil(t, tri.TripleConfig.Cors)
		require.NotNil(t, tri.TripleConfig.OpenAPI)
		assert.Equal(t, []string{"https://example.com"}, tri.TripleConfig.Cors.AllowOrigins)
		assert.Equal(t, "/dubbo/openapi", tri.TripleConfig.OpenAPI.Path)
		assert.Equal(t, defaultOpenAPI.InfoTitle, tri.TripleConfig.OpenAPI.InfoTitle)
		assert.Equal(t, defaultOpenAPI.InfoVersion, tri.TripleConfig.OpenAPI.InfoVersion)
		assert.Equal(t, defaultOpenAPI.DefaultConsumesMediaTypes, tri.TripleConfig.OpenAPI.DefaultConsumesMediaTypes)
		assert.Equal(t, defaultOpenAPI.DefaultProducesMediaTypes, tri.TripleConfig.OpenAPI.DefaultProducesMediaTypes)
		assert.Equal(t, defaultOpenAPI.DefaultHttpStatusCodes, tri.TripleConfig.OpenAPI.DefaultHttpStatusCodes)
		require.NotNil(t, tri.TripleConfig.OpenAPI.Settings)
	})
	require.NoError(t, err)
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
	assert.Equal(t, constant.DefaultTripleProtocolPort, tri.Port)

	_, err = ins.NewServer(func(options *server.ServerOptions) {
		tri := options.Protocols[constant.TriProtocol]
		require.NotNil(t, tri)
		assert.Equal(t, constant.TriProtocol, tri.Name)
		assert.Equal(t, constant.DefaultTripleProtocolPort, tri.Port)
	})
	require.NoError(t, err)
}

func TestInstanceInitAddsDefaultGlobalProtocolWhenEmpty(t *testing.T) {
	ins, err := NewInstance()
	require.NoError(t, err)

	tri := ins.insOpts.Protocols[constant.TriProtocol]
	require.NotNil(t, tri)
	assert.Equal(t, constant.TriProtocol, tri.Name)
	assert.Equal(t, constant.DefaultTripleProtocolPort, tri.Port)
	assert.Equal(t, "4mib", tri.MaxServerRecvMsgSize)
}

func TestInstanceInitTranslatesGlobalRegistryAddress(t *testing.T) {
	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.Registries = map[string]*global.RegistryConfig{
			"zk": {
				Address:         "zookeeper://127.0.0.1:2181?unused=true",
				UseAsMetaReport: "false",
			},
		}
	})
	require.NoError(t, err)

	reg := ins.insOpts.Registries["zk"]
	require.NotNil(t, reg)
	assert.Equal(t, constant.ZookeeperKey, reg.Protocol)
	assert.Equal(t, "127.0.0.1:2181", reg.Address)

	_, err = ins.NewClient(func(options *client.ClientOptions) {
		reg := options.Registries["zk"]
		require.NotNil(t, reg)
		assert.Equal(t, constant.ZookeeperKey, reg.Protocol)
		assert.Equal(t, "127.0.0.1:2181", reg.Address)
	})
	require.NoError(t, err)

	_, err = ins.NewServer(func(options *server.ServerOptions) {
		reg := options.Registries["zk"]
		require.NotNil(t, reg)
		assert.Equal(t, constant.ZookeeperKey, reg.Protocol)
		assert.Equal(t, "127.0.0.1:2181", reg.Address)
	})
	require.NoError(t, err)
}

func TestInstanceInitRejectsDuplicateGlobalRegistryAddress(t *testing.T) {
	_, err := NewInstance(func(opts *InstanceOptions) {
		opts.Registries = map[string]*global.RegistryConfig{
			"zk-a": {
				Protocol: "zookeeper",
				Address:  "127.0.0.1:2181",
			},
			"zk-b": {
				Protocol: "zookeeper",
				Address:  "127.0.0.1:2181",
			},
		}
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate registry address")
}

func TestInstanceInitPropagatesTLSConfig(t *testing.T) {
	assertTLS := func(t *testing.T, cfg *global.TLSConfig) {
		t.Helper()
		require.NotNil(t, cfg)
		assert.Equal(t, "ca.pem", cfg.CACertFile)
		assert.Equal(t, "cert.pem", cfg.TLSCertFile)
		assert.Equal(t, "key.pem", cfg.TLSKeyFile)
		assert.Equal(t, "dubbo.example", cfg.TLSServerName)
	}

	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.TLSConfig = &global.TLSConfig{
			CACertFile:    "ca.pem",
			TLSCertFile:   "cert.pem",
			TLSKeyFile:    "key.pem",
			TLSServerName: "dubbo.example",
		}
	})
	require.NoError(t, err)
	assertTLS(t, ins.insOpts.TLSConfig)

	_, err = ins.NewClient(func(options *client.ClientOptions) {
		assertTLS(t, options.TLS)
	})
	require.NoError(t, err)

	_, err = ins.NewServer(func(options *server.ServerOptions) {
		assertTLS(t, options.TLS)
	})
	require.NoError(t, err)
}

func TestInstanceInitUsesGlobalMetricsDefaults(t *testing.T) {
	enabled := true
	rc := defaultInstanceOptions()
	rc.Metrics = &global.MetricsConfig{Enable: &enabled}

	require.NoError(t, rc.finalizeGlobalOptionsWithRuntimeActivation(false))

	require.NotNil(t, rc.Metrics.EnableMetadata)
	require.NotNil(t, rc.Metrics.EnableRegistry)
	require.NotNil(t, rc.Metrics.EnableConfigCenter)
	require.NotNil(t, rc.Metrics.Prometheus)
	require.NotNil(t, rc.Metrics.Prometheus.Exporter)
	require.NotNil(t, rc.Metrics.Prometheus.Exporter.Enabled)
	assert.False(t, *rc.Metrics.EnableMetadata)
	assert.False(t, *rc.Metrics.EnableRegistry)
	assert.False(t, *rc.Metrics.EnableConfigCenter)
	assert.True(t, *rc.Metrics.Prometheus.Exporter.Enabled)
}

func TestInstanceInitPreservesGlobalTracingConfig(t *testing.T) {
	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.Tracing = map[string]*global.TracingConfig{
			"jaeger": {
				Address: "127.0.0.1:6831",
			},
		}
	})
	require.NoError(t, err)

	assert.Equal(t, "jaeger", ins.insOpts.Provider.TracingKey)
	assert.Equal(t, "jaeger", ins.insOpts.Consumer.TracingKey)

	tracing := ins.insOpts.Tracing["jaeger"]
	require.NotNil(t, tracing)
	assert.Equal(t, "jaeger", tracing.Name)
	assert.Equal(t, "127.0.0.1:6831", tracing.Address)
	require.NotNil(t, tracing.UseAgent)
	assert.False(t, *tracing.UseAgent)
}

func TestInstanceProcessAppliesDynamicUpdatesToLiveOptions(t *testing.T) {
	ins, err := NewInstance(func(opts *InstanceOptions) {
		opts.Application = &global.ApplicationConfig{
			Name:    "stable-app",
			Version: "1.0.0",
		}
		opts.Consumer = global.DefaultConsumerConfig()
		opts.Consumer.RequestTimeout = "3s"
		opts.Registries = map[string]*global.RegistryConfig{
			"zk": {
				Protocol:        "zookeeper",
				Address:         "127.0.0.1:2181",
				Timeout:         "5s",
				UseAsMetaReport: "false",
			},
		}
		opts.Shutdown = global.DefaultShutdownConfig()
		opts.Shutdown.ClosingInvokerExpireTime = "7s"
		opts.TLSConfig = &global.TLSConfig{
			CACertFile:    "ca.pem",
			TLSCertFile:   "cert.pem",
			TLSKeyFile:    "key.pem",
			TLSServerName: "dubbo.example",
		}
	})
	require.NoError(t, err)

	ins.insOpts.Process(&config_center.ConfigChangeEvent{Value: `
dubbo:
  application:
    name: dynamic-app
  registries:
    zk:
      timeout: 9s
  consumer:
    request-timeout: 11s
  shutdown:
    closing-invoker-expire-time: 2s
`})

	assert.Equal(t, "stable-app", ins.insOpts.Application.Name)
	assert.Equal(t, "1.0.0", ins.insOpts.Application.Version)
	assert.Equal(t, "9s", ins.insOpts.Registries["zk"].Timeout)
	assert.Equal(t, "11s", ins.insOpts.Consumer.RequestTimeout)
	assert.Equal(t, "7s", ins.insOpts.Shutdown.ClosingInvokerExpireTime)
	require.NotNil(t, ins.insOpts.TLSConfig)
	assert.Equal(t, "dubbo.example", ins.insOpts.TLSConfig.TLSServerName)
}

func TestInstanceProcessDoesNotMutateLiveOptionsOnInvalidPayload(t *testing.T) {
	ins, err := NewInstance(WithName("stable-app"))
	require.NoError(t, err)

	ins.insOpts.Process(&config_center.ConfigChangeEvent{Value: 123})

	assert.Equal(t, "stable-app", ins.insOpts.Application.Name)
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

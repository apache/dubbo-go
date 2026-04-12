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

package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
)

const (
	compatProtoID    = "tri-compat"
	compatRegID      = "nacos-compat"
	compatSvcID      = "com.example.CompatProviderService"
	compatRefID      = "com.example.CompatReferenceService"
)

func TestCompatApp(t *testing.T) {
	root := compatRoot(t)

	require.NotNil(t, root.Application)
	assert.Equal(t, "compat-app", root.Application.Name)
	assert.Equal(t, "compat-group", root.Application.Group)
	assert.Equal(t, "1.0.0", root.Application.Version)
	assert.Equal(t, "gray", root.Application.Tag)
}

func TestCompatProto(t *testing.T) {
	root := compatRoot(t)

	require.Len(t, root.Protocols, 1)
	protocolCfg := root.Protocols[compatProtoID]
	require.NotNil(t, protocolCfg)
	assert.Equal(t, "tri", protocolCfg.Name)
	assert.Equal(t, "127.0.0.1", protocolCfg.Ip)
	assert.Equal(t, "20000", protocolCfg.Port)
	assert.Equal(t, "8mib", protocolCfg.MaxServerSendMsgSize)
	assert.Equal(t, "6mib", protocolCfg.MaxServerRecvMsgSize)
	require.NotNil(t, protocolCfg.TripleConfig)
	assert.Equal(t, "30s", protocolCfg.TripleConfig.KeepAliveInterval)
	assert.Equal(t, "10s", protocolCfg.TripleConfig.KeepAliveTimeout)
	require.NotNil(t, protocolCfg.TripleConfig.Http3)
	assert.True(t, protocolCfg.TripleConfig.Http3.Enable)
	assert.False(t, protocolCfg.TripleConfig.Http3.Negotiation)
}

func TestCompatRegistry(t *testing.T) {
	root := compatRoot(t)

	require.Len(t, root.Registries, 1)
	registryCfg := root.Registries[compatRegID]
	require.NotNil(t, registryCfg)
	assert.Equal(t, "nacos", registryCfg.Protocol)
	assert.Equal(t, "127.0.0.1:8848", registryCfg.Address)
	assert.Equal(t, "compat-ns", registryCfg.Namespace)
	assert.Equal(t, "true", registryCfg.UseAsConfigCenter)
}

func TestCompatProvider(t *testing.T) {
	root := compatRoot(t)

	require.NotNil(t, root.Provider)
	assert.Equal(t, "echo", root.Provider.Filter)
	assert.Equal(t, []string{compatRegID}, root.Provider.RegistryIDs)
	assert.Equal(t, []string{compatProtoID}, root.Provider.ProtocolIDs)
	assert.Contains(t, root.Provider.Services, compatSvcID)
	providerSvc := root.Provider.Services[compatSvcID]
	require.NotNil(t, providerSvc)
	assert.Equal(t, compatSvcID, providerSvc.Interface)
	assert.Equal(t, "provider-group", providerSvc.Group)
	assert.Equal(t, "provider-v1", providerSvc.Version)
	assert.Equal(t, "failover", providerSvc.Cluster)
	assert.Equal(t, "random", providerSvc.Loadbalance)
	require.Len(t, providerSvc.Methods, 1)
	assert.Equal(t, "SayHello", providerSvc.Methods[0].Name)
	assert.Equal(t, "2s", providerSvc.Methods[0].RequestTimeout)
}

func TestCompatConsumer(t *testing.T) {
	root := compatRoot(t)

	require.NotNil(t, root.Consumer)
	assert.Equal(t, "cshutdown", root.Consumer.Filter)
	assert.Equal(t, "tri", root.Consumer.Protocol)
	assert.Equal(t, "5s", root.Consumer.RequestTimeout)
	assert.Contains(t, root.Consumer.References, compatRefID)
	refCfg := root.Consumer.References[compatRefID]
	require.NotNil(t, refCfg)
	assert.Equal(t, compatRefID, refCfg.InterfaceName)
	assert.Equal(t, "tri://127.0.0.1:20000", refCfg.URL)
	assert.Equal(t, "ref-group", refCfg.Group)
	assert.Equal(t, "ref-v1", refCfg.Version)
	assert.Equal(t, "tri", refCfg.Protocol)
	require.Len(t, refCfg.MethodsConfig, 1)
	assert.Equal(t, "Query", refCfg.MethodsConfig[0].Name)
	assert.Equal(t, "1s", refCfg.MethodsConfig[0].RequestTimeout)
}

func TestCompatMetricsOtel(t *testing.T) {
	root := compatRoot(t)

	require.NotNil(t, root.Metrics)
	require.NotNil(t, root.Metrics.Enable)
	assert.True(t, *root.Metrics.Enable)
	assert.Equal(t, "prometheus", root.Metrics.Protocol)
	require.NotNil(t, root.Metrics.Probe)
	assert.Equal(t, "22333", root.Metrics.Probe.Port)

	require.NotNil(t, root.Otel)
	require.NotNil(t, root.Otel.TraceConfig)
	require.NotNil(t, root.Otel.TraceConfig.Enable)
	assert.False(t, *root.Otel.TraceConfig.Enable)
	assert.Equal(t, "stdout", root.Otel.TraceConfig.Exporter)
	assert.Equal(t, "http://127.0.0.1:4318", root.Otel.TraceConfig.Endpoint)
}

func TestCompatShutdown(t *testing.T) {
	root := compatRoot(t)

	require.NotNil(t, root.Shutdown)
	assert.Equal(t, "60s", root.Shutdown.Timeout)
	assert.Equal(t, "3s", root.Shutdown.StepTimeout)
	assert.Equal(t, "2s", root.Shutdown.ConsumerUpdateWaitTime)
	require.NotNil(t, root.Shutdown.InternalSignal)
	assert.True(t, *root.Shutdown.InternalSignal)
}

// TestCompatEmptyCollections verifies nil-tolerant behavior
// for optional sections and empty collections in compat conversion.
func TestCompatEmptyCollections(t *testing.T) {
	_, err := dubbo.NewInstance(func(opts *dubbo.InstanceOptions) {
		opts.Application.Name = "compat-empty"
		opts.Protocols = compatProtocols("20001")
		opts.Registries = map[string]*global.RegistryConfig{}
		opts.Router = nil
	})
	require.NoError(t, err)

	root := config.GetRootConfig()
	require.NotNil(t, root)
	assert.NotNil(t, root.Protocols)
	require.Len(t, root.Protocols, 1)
	assert.NotNil(t, root.Registries)
	assert.Empty(t, root.Registries)
	assert.NotNil(t, root.Router)
	assert.Empty(t, root.Router)
}

func compatRoot(t *testing.T, opts ...dubbo.InstanceOption) *config.RootConfig {
	t.Helper()

	allOpts := make([]dubbo.InstanceOption, 0, len(opts)+1)
	allOpts = append(allOpts, compatFixture())
	allOpts = append(allOpts, opts...)

	_, err := dubbo.NewInstance(allOpts...)
	require.NoError(t, err)

	root := config.GetRootConfig()
	require.NotNil(t, root)
	return root
}

func compatFixture() dubbo.InstanceOption {
	return func(opts *dubbo.InstanceOptions) {
		opts.Application = compatApplication()
		opts.Protocols = compatProtocols("20000")
		opts.Registries = compatRegistries()
		opts.Provider = compatProvider()
		opts.Consumer = compatConsumer()
		opts.Metrics = compatMetrics()
		opts.Otel = compatOtel()
		opts.Shutdown = compatShutdown()
		opts.Router = compatRouter()
		opts.Custom = compatCustom()
		opts.Profiles = compatProfiles()
	}
}

func compatApplication() *global.ApplicationConfig {
	return &global.ApplicationConfig{
		Name:    "compat-app",
		Group:   "compat-group",
		Version: "1.0.0",
		Tag:     "gray",
	}
}

func compatProtocols(port string) map[string]*global.ProtocolConfig {
	return map[string]*global.ProtocolConfig{
		compatProtoID: {
			Name:                 "tri",
			Ip:                   "127.0.0.1",
			Port:                 port,
			MaxServerSendMsgSize: "8mib",
			MaxServerRecvMsgSize: "6mib",
			TripleConfig: &global.TripleConfig{
				KeepAliveInterval: "30s",
				KeepAliveTimeout:  "10s",
				Http3: &global.Http3Config{
					Enable:      true,
					Negotiation: false,
				},
			},
		},
	}
}

func compatRegistries() map[string]*global.RegistryConfig {
	return map[string]*global.RegistryConfig{
		compatRegID: {
			Protocol:          "nacos",
			Address:           "127.0.0.1:8848",
			Namespace:         "compat-ns",
			UseAsConfigCenter: "true",
		},
	}
}

func compatProvider() *global.ProviderConfig {
	return &global.ProviderConfig{
		Filter:      "echo",
		RegistryIDs: []string{compatRegID},
		ProtocolIDs: []string{compatProtoID},
		Services: map[string]*global.ServiceConfig{
			compatSvcID: {
				Interface:   compatSvcID,
				Group:       "provider-group",
				Version:     "provider-v1",
				Cluster:     "failover",
				Loadbalance: "random",
				Methods: []*global.MethodConfig{
					{
						Name:           "SayHello",
						RequestTimeout: "2s",
					},
				},
			},
		},
	}
}

func compatConsumer() *global.ConsumerConfig {
	referenceCheck := true
	return &global.ConsumerConfig{
		Filter:         "cshutdown",
		Protocol:       "tri",
		RequestTimeout: "5s",
		Check:          true,
		References: map[string]*global.ReferenceConfig{
			compatRefID: {
				InterfaceName: compatRefID,
				Check:         &referenceCheck,
				URL:           "tri://127.0.0.1:20000",
				Protocol:      "tri",
				Group:         "ref-group",
				Version:       "ref-v1",
				MethodsConfig: []*global.MethodConfig{
					{
						Name:           "Query",
						RequestTimeout: "1s",
					},
				},
			},
		},
	}
}

func compatMetrics() *global.MetricsConfig {
	metricEnabled := true
	return &global.MetricsConfig{
		Enable:   &metricEnabled,
		Protocol: "prometheus",
		Probe: &global.ProbeConfig{
			Port: "22333",
		},
	}
}

func compatOtel() *global.OtelConfig {
	traceEnabled := false
	return &global.OtelConfig{
		TracingConfig: &global.OtelTraceConfig{
			Enable:   &traceEnabled,
			Exporter: "stdout",
			Endpoint: "http://127.0.0.1:4318",
		},
	}
}

func compatShutdown() *global.ShutdownConfig {
	internalSignal := true
	return &global.ShutdownConfig{
		Timeout:                "60s",
		StepTimeout:            "3s",
		ConsumerUpdateWaitTime: "2s",
		InternalSignal:         &internalSignal,
	}
}

func compatRouter() []*global.RouterConfig {
	return []*global.RouterConfig{}
}

func compatCustom() *global.CustomConfig {
	return &global.CustomConfig{
		ConfigMap: map[string]any{"k": "v"},
	}
}

func compatProfiles() *global.ProfilesConfig {
	return &global.ProfilesConfig{
		Active: "test",
	}
}

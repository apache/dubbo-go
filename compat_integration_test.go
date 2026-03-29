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
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

const (
	newConfigAPICompatProtocolID    = "tri-compat"
	newConfigAPICompatRegistryID    = "nacos-compat"
	newConfigAPICompatProviderSvcID = "com.example.CompatProviderService"
	newConfigAPICompatReferenceID   = "com.example.CompatReferenceService"
)

// TestNewConfigAPI_CompatRoundTripPreservesKeyFields verifies that
// InstanceOptions -> RootConfig -> InstanceOptions keeps key fields consistent.
func TestNewConfigAPI_CompatRoundTripPreservesKeyFields(t *testing.T) {
	src := buildNewConfigAPICompatFixture()

	root := compatRootConfig(src)
	require.NotNil(t, root)

	dst := defaultInstanceOptions()
	compatInstanceOptions(root, dst)

	require.NotNil(t, dst.Application)
	assert.Equal(t, src.Application.Name, dst.Application.Name)
	assert.Equal(t, src.Application.Group, dst.Application.Group)
	assert.Equal(t, src.Application.Version, dst.Application.Version)
	assert.Equal(t, src.Application.Tag, dst.Application.Tag)

	require.Len(t, dst.Protocols, 1)
	srcProtocol := src.Protocols[newConfigAPICompatProtocolID]
	dstProtocol := dst.Protocols[newConfigAPICompatProtocolID]
	require.NotNil(t, dstProtocol)
	assert.Equal(t, srcProtocol.Name, dstProtocol.Name)
	assert.Equal(t, srcProtocol.Ip, dstProtocol.Ip)
	assert.Equal(t, srcProtocol.Port, dstProtocol.Port)
	assert.Equal(t, srcProtocol.MaxServerSendMsgSize, dstProtocol.MaxServerSendMsgSize)
	assert.Equal(t, srcProtocol.MaxServerRecvMsgSize, dstProtocol.MaxServerRecvMsgSize)
	require.NotNil(t, dstProtocol.TripleConfig)
	assert.Equal(t, srcProtocol.TripleConfig.KeepAliveInterval, dstProtocol.TripleConfig.KeepAliveInterval)
	assert.Equal(t, srcProtocol.TripleConfig.KeepAliveTimeout, dstProtocol.TripleConfig.KeepAliveTimeout)
	require.NotNil(t, dstProtocol.TripleConfig.Http3)
	assert.Equal(t, srcProtocol.TripleConfig.Http3.Enable, dstProtocol.TripleConfig.Http3.Enable)
	assert.Equal(t, srcProtocol.TripleConfig.Http3.Negotiation, dstProtocol.TripleConfig.Http3.Negotiation)

	require.Len(t, dst.Registries, 1)
	srcRegistry := src.Registries[newConfigAPICompatRegistryID]
	dstRegistry := dst.Registries[newConfigAPICompatRegistryID]
	require.NotNil(t, dstRegistry)
	assert.Equal(t, srcRegistry.Protocol, dstRegistry.Protocol)
	assert.Equal(t, srcRegistry.Address, dstRegistry.Address)
	assert.Equal(t, srcRegistry.Namespace, dstRegistry.Namespace)
	assert.Equal(t, srcRegistry.UseAsConfigCenter, dstRegistry.UseAsConfigCenter)

	require.NotNil(t, dst.Provider)
	assert.Equal(t, src.Provider.Filter, dst.Provider.Filter)
	assert.Equal(t, src.Provider.RegistryIDs, dst.Provider.RegistryIDs)
	assert.Equal(t, src.Provider.ProtocolIDs, dst.Provider.ProtocolIDs)
	require.Len(t, dst.Provider.Services, 1)
	srcSvc := src.Provider.Services[newConfigAPICompatProviderSvcID]
	dstSvc := dst.Provider.Services[newConfigAPICompatProviderSvcID]
	require.NotNil(t, dstSvc)
	assert.Equal(t, srcSvc.Interface, dstSvc.Interface)
	assert.Equal(t, srcSvc.Group, dstSvc.Group)
	assert.Equal(t, srcSvc.Version, dstSvc.Version)
	assert.Equal(t, srcSvc.Cluster, dstSvc.Cluster)
	assert.Equal(t, srcSvc.Loadbalance, dstSvc.Loadbalance)
	require.Len(t, dstSvc.Methods, 1)
	assert.Equal(t, srcSvc.Methods[0].Name, dstSvc.Methods[0].Name)
	assert.Equal(t, srcSvc.Methods[0].RequestTimeout, dstSvc.Methods[0].RequestTimeout)

	require.NotNil(t, dst.Consumer)
	assert.Equal(t, src.Consumer.Filter, dst.Consumer.Filter)
	assert.Equal(t, src.Consumer.Protocol, dst.Consumer.Protocol)
	assert.Equal(t, src.Consumer.RequestTimeout, dst.Consumer.RequestTimeout)
	require.Len(t, dst.Consumer.References, 1)
	srcRef := src.Consumer.References[newConfigAPICompatReferenceID]
	dstRef := dst.Consumer.References[newConfigAPICompatReferenceID]
	require.NotNil(t, dstRef)
	assert.Equal(t, srcRef.InterfaceName, dstRef.InterfaceName)
	assert.Equal(t, srcRef.URL, dstRef.URL)
	assert.Equal(t, srcRef.Group, dstRef.Group)
	assert.Equal(t, srcRef.Version, dstRef.Version)
	assert.Equal(t, srcRef.Protocol, dstRef.Protocol)
	require.Len(t, dstRef.MethodsConfig, 1)
	assert.Equal(t, srcRef.MethodsConfig[0].Name, dstRef.MethodsConfig[0].Name)
	assert.Equal(t, srcRef.MethodsConfig[0].RequestTimeout, dstRef.MethodsConfig[0].RequestTimeout)

	require.NotNil(t, dst.Metrics)
	require.NotNil(t, src.Metrics.Enable)
	require.NotNil(t, dst.Metrics.Enable)
	assert.Equal(t, *src.Metrics.Enable, *dst.Metrics.Enable)
	assert.Equal(t, src.Metrics.Protocol, dst.Metrics.Protocol)
	require.NotNil(t, src.Metrics.Probe)
	require.NotNil(t, dst.Metrics.Probe)
	assert.Equal(t, src.Metrics.Probe.Port, dst.Metrics.Probe.Port)

	require.NotNil(t, dst.Otel)
	require.NotNil(t, src.Otel.TracingConfig)
	require.NotNil(t, dst.Otel.TracingConfig)
	require.NotNil(t, src.Otel.TracingConfig.Enable)
	require.NotNil(t, dst.Otel.TracingConfig.Enable)
	assert.Equal(t, *src.Otel.TracingConfig.Enable, *dst.Otel.TracingConfig.Enable)
	assert.Equal(t, src.Otel.TracingConfig.Exporter, dst.Otel.TracingConfig.Exporter)
	assert.Equal(t, src.Otel.TracingConfig.Endpoint, dst.Otel.TracingConfig.Endpoint)

	require.NotNil(t, dst.Shutdown)
	assert.Equal(t, src.Shutdown.Timeout, dst.Shutdown.Timeout)
	assert.Equal(t, src.Shutdown.StepTimeout, dst.Shutdown.StepTimeout)
	assert.Equal(t, src.Shutdown.ConsumerUpdateWaitTime, dst.Shutdown.ConsumerUpdateWaitTime)
	require.NotNil(t, src.Shutdown.InternalSignal)
	require.NotNil(t, dst.Shutdown.InternalSignal)
	assert.Equal(t, *src.Shutdown.InternalSignal, *dst.Shutdown.InternalSignal)
}

// TestNewConfigAPI_CompatRoundTripHandlesNilOptionalFields verifies nil-tolerant
// behavior for optional sections and empty collections in compat conversion.
func TestNewConfigAPI_CompatRoundTripHandlesNilOptionalFields(t *testing.T) {
	src := &InstanceOptions{
		Protocols:  map[string]*global.ProtocolConfig{},
		Registries: map[string]*global.RegistryConfig{},
	}

	root := compatRootConfig(src)
	require.NotNil(t, root)

	dst := &InstanceOptions{}
	require.NotPanics(t, func() {
		compatInstanceOptions(root, dst)
	})

	assert.Nil(t, dst.Application)
	assert.NotNil(t, dst.Protocols)
	assert.Empty(t, dst.Protocols)
	assert.NotNil(t, dst.Registries)
	assert.Empty(t, dst.Registries)
	assert.NotNil(t, dst.Router)
	assert.Empty(t, dst.Router)
}

func buildNewConfigAPICompatFixture() *InstanceOptions {
	metricEnabled := true
	traceEnabled := true
	internalSignal := true
	referenceCheck := true

	return &InstanceOptions{
		Application: &global.ApplicationConfig{
			Name:    "compat-app",
			Group:   "compat-group",
			Version: "1.0.0",
			Tag:     "gray",
		},
		Protocols: map[string]*global.ProtocolConfig{
			newConfigAPICompatProtocolID: {
				Name:                 "tri",
				Ip:                   "127.0.0.1",
				Port:                 "20000",
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
		},
		Registries: map[string]*global.RegistryConfig{
			newConfigAPICompatRegistryID: {
				Protocol:          "nacos",
				Address:           "127.0.0.1:8848",
				Namespace:         "compat-ns",
				UseAsConfigCenter: "true",
			},
		},
		Provider: &global.ProviderConfig{
			Filter:      "echo",
			RegistryIDs: []string{newConfigAPICompatRegistryID},
			ProtocolIDs: []string{newConfigAPICompatProtocolID},
			Services: map[string]*global.ServiceConfig{
				newConfigAPICompatProviderSvcID: {
					Interface:   newConfigAPICompatProviderSvcID,
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
		},
		Consumer: &global.ConsumerConfig{
			Filter:         "cshutdown",
			Protocol:       "tri",
			RequestTimeout: "5s",
			Check:          true,
			References: map[string]*global.ReferenceConfig{
				newConfigAPICompatReferenceID: {
					InterfaceName: newConfigAPICompatReferenceID,
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
		},
		Metrics: &global.MetricsConfig{
			Enable:   &metricEnabled,
			Protocol: "prometheus",
			Probe: &global.ProbeConfig{
				Port: "22333",
			},
		},
		Otel: &global.OtelConfig{
			TracingConfig: &global.OtelTraceConfig{
				Enable:   &traceEnabled,
				Exporter: "stdout",
				Endpoint: "http://127.0.0.1:4318",
			},
		},
		Shutdown: &global.ShutdownConfig{
			Timeout:                "60s",
			StepTimeout:            "3s",
			ConsumerUpdateWaitTime: "2s",
			InternalSignal:         &internalSignal,
		},
		Router: []*global.RouterConfig{},
		Custom: &global.CustomConfig{
			ConfigMap: map[string]any{"k": "v"},
		},
		Profiles: &global.ProfilesConfig{
			Active: "test",
		},
	}
}

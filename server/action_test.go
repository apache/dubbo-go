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

package server

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// Test Prefix method
func TestPrefix(t *testing.T) {
	svcOpts := &ServiceOptions{
		Id: "com.example.TestService",
	}

	prefix := svcOpts.Prefix()
	assert.Equal(t, "dubbo.service.com.example.TestService", prefix)
}

// Test InitExported
func TestInitExported(t *testing.T) {
	svcOpts := &ServiceOptions{
		exported: atomic.NewBool(true),
	}

	svcOpts.InitExported()
	assert.False(t, svcOpts.exported.Load())
}

// Test IsExport returns false initially
func TestIsExportFalse(t *testing.T) {
	svcOpts := &ServiceOptions{
		exported: atomic.NewBool(false),
	}

	assert.False(t, svcOpts.IsExport())
}

// Test IsExport returns true when exported
func TestIsExportTrue(t *testing.T) {
	svcOpts := &ServiceOptions{
		exported: atomic.NewBool(true),
	}

	assert.True(t, svcOpts.IsExport())
}

// Test check with valid TpsLimiter
func TestCheckValidTpsLimiter(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:  "com.example.Service",
			TpsLimiter: "",
		},
	}

	err := svcOpts.check()
	assert.NoError(t, err)
}

// Test check with valid TpsLimitRate
func TestCheckValidTpsLimitRate(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:    "com.example.Service",
			TpsLimitRate: "1000",
		},
	}

	err := svcOpts.check()
	assert.NoError(t, err)
}

// Test check with invalid TpsLimitRate
func TestCheckInvalidTpsLimitRate(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:    "com.example.Service",
			TpsLimitRate: "invalid",
		},
	}

	err := svcOpts.check()
	assert.Error(t, err)
}

// Test check with negative TpsLimitRate
func TestCheckNegativeTpsLimitRate(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:    "com.example.Service",
			TpsLimitRate: "-1",
		},
	}

	err := svcOpts.check()
	assert.Error(t, err)
}

// Test check with valid TpsLimitInterval
func TestCheckValidTpsLimitInterval(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:        "com.example.Service",
			TpsLimitInterval: "1000",
		},
	}

	err := svcOpts.check()
	assert.NoError(t, err)
}

// Test check with invalid TpsLimitInterval
func TestCheckInvalidTpsLimitInterval(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:        "com.example.Service",
			TpsLimitInterval: "invalid",
		},
	}

	err := svcOpts.check()
	assert.Error(t, err)
}

// Test check with negative TpsLimitInterval
func TestCheckNegativeTpsLimitInterval(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:        "com.example.Service",
			TpsLimitInterval: "-1",
		},
	}

	err := svcOpts.check()
	assert.Error(t, err)
}

// Test Implement
func TestImplement(t *testing.T) {
	svcOpts := &ServiceOptions{}
	mockService := &mockRPCService{}

	svcOpts.Implement(mockService)

	assert.Equal(t, mockService, svcOpts.rpcService)
}

// Test Unexport when not exported
func TestUnexportNotExported(t *testing.T) {
	svcOpts := &ServiceOptions{
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
		exporters:  []base.Exporter{},
	}

	svcOpts.Unexport()
	assert.False(t, svcOpts.exported.Load())
	assert.False(t, svcOpts.unexported.Load())
}

// Test Unexport when already unexported
func TestUnexportAlreadyUnexported(t *testing.T) {
	svcOpts := &ServiceOptions{
		unexported: atomic.NewBool(true),
		exported:   atomic.NewBool(false),
		exporters:  []base.Exporter{},
	}

	svcOpts.Unexport()
	assert.True(t, svcOpts.unexported.Load())
}

// Test removeDuplicateElement
func TestRemoveDuplicateElement(t *testing.T) {
	items := []string{"a", "b", "a", "c", "b", ""}
	result := removeDuplicateElement(items)

	assert.Len(t, result, 3)
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")
	assert.Contains(t, result, "c")
}

// Test removeDuplicateElement with empty slice
func TestRemoveDuplicateElementEmpty(t *testing.T) {
	items := []string{}
	result := removeDuplicateElement(items)

	assert.Empty(t, result)
}

// Test removeDuplicateElement with only empty strings
func TestRemoveDuplicateElementOnlyEmpty(t *testing.T) {
	items := []string{"", "", ""}
	result := removeDuplicateElement(items)

	assert.Empty(t, result)
}

// Test removeDuplicateElement with mixed content
func TestRemoveDuplicateElementMixed(t *testing.T) {
	items := []string{"reg1", "reg2", "reg1", "", "reg3", ""}
	result := removeDuplicateElement(items)

	assert.Len(t, result, 3)
	assert.Contains(t, result, "reg1")
	assert.Contains(t, result, "reg2")
	assert.Contains(t, result, "reg3")
}

// Test getRegistryIds
func TestGetRegistryIds(t *testing.T) {
	registries := map[string]*global.RegistryConfig{
		"reg1": {Protocol: "zookeeper"},
		"reg2": {Protocol: "nacos"},
		"reg3": {Protocol: "etcd"},
	}

	ids := getRegistryIds(registries)

	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "reg1")
	assert.Contains(t, ids, "reg2")
	assert.Contains(t, ids, "reg3")
}

// Test getRegistryIds with empty map
func TestGetRegistryIdsEmpty(t *testing.T) {
	registries := map[string]*global.RegistryConfig{}

	ids := getRegistryIds(registries)

	assert.Empty(t, ids)
}

// Test GetExportedUrls when not exported
func TestGetExportedUrlsNotExported(t *testing.T) {
	svcOpts := &ServiceOptions{
		exported:  atomic.NewBool(false),
		exporters: []base.Exporter{},
	}

	urls := svcOpts.GetExportedUrls()
	assert.Nil(t, urls)
}

// Test GetExportedUrls when exported with no exporters
func TestGetExportedUrlsExportedEmpty(t *testing.T) {
	svcOpts := &ServiceOptions{
		exported:  atomic.NewBool(true),
		exporters: []base.Exporter{},
	}

	urls := svcOpts.GetExportedUrls()
	assert.Empty(t, urls)
}

// Test getUrlMap basic functionality
func TestGetUrlMapBasic(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:     "com.example.Service",
			Cluster:       constant.ClusterKeyFailover,
			Loadbalance:   constant.LoadBalanceKeyRoundRobin,
			Warmup:        "60",
			Retries:       "3",
			Serialization: constant.JSONSerialization,
			Filter:        "",
			Params:        map[string]string{},
			NotRegister:   false,
			Tag:           "v1",
			Group:         "test-group",
			Version:       "1.0.0",
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name:         "test-app",
			Organization: "test-org",
			Module:       "test-module",
			Owner:        "test-owner",
			Environment:  "test-env",
			Version:      "1.0.0",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{Enable: nil},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{Enable: nil},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.NotNil(t, urlMap)
	assert.Equal(t, "com.example.Service", urlMap.Get(constant.InterfaceKey))
	assert.Equal(t, constant.ClusterKeyFailover, urlMap.Get(constant.ClusterKey))
	assert.Equal(t, constant.LoadBalanceKeyRoundRobin, urlMap.Get(constant.LoadbalanceKey))
	assert.Equal(t, "60", urlMap.Get(constant.WarmupKey))
	assert.Equal(t, "3", urlMap.Get(constant.RetriesKey))
	assert.Equal(t, constant.JSONSerialization, urlMap.Get(constant.SerializationKey))
}

// Test getUrlMap with custom params
func TestGetUrlMapWithParams(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface: "com.example.Service",
			Params: map[string]string{
				"customKey": "customValue",
			},
			Cluster:       constant.ClusterKeyFailover,
			Loadbalance:   constant.LoadBalanceKeyRoundRobin,
			Warmup:        "60",
			Retries:       "3",
			Serialization: constant.JSONSerialization,
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name:         "test-app",
			Organization: "test-org",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.NotNil(t, urlMap)
	assert.Equal(t, "customValue", urlMap.Get("customKey"))
}

// Test getUrlMap with group and version
func TestGetUrlMapWithGroupAndVersion(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:     "com.example.Service",
			Group:         "test-group",
			Version:       "1.0.0",
			Cluster:       constant.ClusterKeyFailover,
			Loadbalance:   constant.LoadBalanceKeyRoundRobin,
			Warmup:        "60",
			Retries:       "3",
			Serialization: constant.JSONSerialization,
			Params:        map[string]string{},
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name: "test-app",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.Equal(t, "test-group", urlMap.Get(constant.GroupKey))
	assert.Equal(t, "1.0.0", urlMap.Get(constant.VersionKey))
}

// Test getUrlMap with methods
func TestGetUrlMapWithMethods(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:     "com.example.Service",
			Cluster:       constant.ClusterKeyFailover,
			Loadbalance:   constant.LoadBalanceKeyRoundRobin,
			Warmup:        "60",
			Retries:       "3",
			Serialization: constant.JSONSerialization,
			Params:        map[string]string{},
			Methods: []*global.MethodConfig{
				{
					Name:        "testMethod",
					LoadBalance: "random",
					Retries:     "5",
					Weight:      200,
				},
			},
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name: "test-app",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.Equal(t, "random", urlMap.Get("methods.testMethod."+constant.LoadbalanceKey))
	assert.Equal(t, "5", urlMap.Get("methods.testMethod."+constant.RetriesKey))
	assert.Equal(t, "200", urlMap.Get("methods.testMethod."+constant.WeightKey))
}

// Test loadProtocol with matching IDs
func TestLoadProtocol(t *testing.T) {
	protocols := map[string]*global.ProtocolConfig{
		"dubbo": {
			Name: "dubbo",
			Port: "20880",
		},
		"triple": {
			Name: "triple",
			Port: "50051",
		},
	}

	protocolIDs := []string{"dubbo", "triple"}
	result := loadProtocol(protocolIDs, protocols)

	assert.Len(t, result, 2)
}

// Test loadProtocol with partial matching
func TestLoadProtocolPartialMatch(t *testing.T) {
	protocols := map[string]*global.ProtocolConfig{
		"dubbo": {
			Name: "dubbo",
			Port: "20880",
		},
		"triple": {
			Name: "triple",
			Port: "50051",
		},
	}

	protocolIDs := []string{"dubbo"}
	result := loadProtocol(protocolIDs, protocols)

	assert.Len(t, result, 1)
	assert.Equal(t, "dubbo", result[0].Name)
}

// Test loadProtocol with no matching IDs
func TestLoadProtocolNoMatch(t *testing.T) {
	protocols := map[string]*global.ProtocolConfig{
		"dubbo": {
			Name: "dubbo",
			Port: "20880",
		},
	}

	protocolIDs := []string{"non-existent"}
	result := loadProtocol(protocolIDs, protocols)

	assert.Empty(t, result)
}

// Test setRegistrySubURL
func TestSetRegistrySubURL(t *testing.T) {
	ivkURL, _ := common.NewURL("dubbo://localhost:20880/com.example.Service")
	ivkURL.AddParam(constant.RegistryKey, "zookeeper")
	ivkURL.AddParam(constant.RegistryTypeKey, "service_discovery")

	regURL, _ := common.NewURL("registry://zookeeper:2181")
	regURL.AddParam(constant.RegistryKey, "zookeeper")
	regURL.AddParam(constant.RegistryTypeKey, "service_discovery")

	setRegistrySubURL(ivkURL, regURL)

	assert.Equal(t, "zookeeper", ivkURL.GetParam(constant.RegistryKey, ""))
	assert.Equal(t, "service_discovery", ivkURL.GetParam(constant.RegistryTypeKey, ""))
	assert.NotNil(t, regURL.SubURL)
}

// Test Unexport when exported
func TestUnexportWhenExported(t *testing.T) {
	svcOpts := &ServiceOptions{
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(true),
		exporters:  []base.Exporter{},
	}

	svcOpts.Unexport()
	assert.False(t, svcOpts.exported.Load())
	assert.True(t, svcOpts.unexported.Load())
}

// Test check with multiple validation errors
func TestCheckMultipleErrors(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:        "com.example.Service",
			TpsLimitRate:     "invalid",
			TpsLimitInterval: "also-invalid",
		},
	}

	err := svcOpts.check()
	assert.Error(t, err)
}

// Test getUrlMap with TPS limit config
func TestGetUrlMapWithTpsLimit(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:               "com.example.Service",
			TpsLimiter:              "default",
			TpsLimitRate:            "1000",
			TpsLimitStrategy:        "adaptive",
			TpsLimitRejectedHandler: "abort",
			TpsLimitInterval:        "100",
			Cluster:                 constant.ClusterKeyFailover,
			Loadbalance:             constant.LoadBalanceKeyRoundRobin,
			Warmup:                  "60",
			Retries:                 "3",
			Serialization:           constant.JSONSerialization,
			Params:                  map[string]string{},
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name: "test-app",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.Equal(t, "default", urlMap.Get(constant.TPSLimiterKey))
	assert.Equal(t, "1000", urlMap.Get(constant.TPSLimitRateKey))
	assert.Equal(t, "adaptive", urlMap.Get(constant.TPSLimitStrategyKey))
	assert.Equal(t, "abort", urlMap.Get(constant.TPSRejectedExecutionHandlerKey))
}

// Test getUrlMap with execute limit config
func TestGetUrlMapWithExecuteLimit(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:                   "com.example.Service",
			ExecuteLimit:                "200",
			ExecuteLimitRejectedHandler: "abort",
			Cluster:                     constant.ClusterKeyFailover,
			Loadbalance:                 constant.LoadBalanceKeyRoundRobin,
			Warmup:                      "60",
			Retries:                     "3",
			Serialization:               constant.JSONSerialization,
			Params:                      map[string]string{},
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name: "test-app",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.Equal(t, "200", urlMap.Get(constant.ExecuteLimitKey))
	assert.Equal(t, "abort", urlMap.Get(constant.ExecuteRejectedExecutionHandlerKey))
}

// Test getUrlMap with auth and param sign
func TestGetUrlMapWithAuthAndParamSign(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:     "com.example.Service",
			Auth:          "default",
			ParamSign:     "md5",
			Cluster:       constant.ClusterKeyFailover,
			Loadbalance:   constant.LoadBalanceKeyRoundRobin,
			Warmup:        "60",
			Retries:       "3",
			Serialization: constant.JSONSerialization,
			Params:        map[string]string{},
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name: "test-app",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.Equal(t, "default", urlMap.Get(constant.ServiceAuthKey))
	assert.Equal(t, "md5", urlMap.Get(constant.ParameterSignatureEnableKey))
}

// Test getUrlMap with access log
func TestGetUrlMapWithAccessLog(t *testing.T) {
	svcOpts := &ServiceOptions{
		Service: &global.ServiceConfig{
			Interface:     "com.example.Service",
			AccessLog:     "/var/log/access.log",
			Cluster:       constant.ClusterKeyFailover,
			Loadbalance:   constant.LoadBalanceKeyRoundRobin,
			Warmup:        "60",
			Retries:       "3",
			Serialization: constant.JSONSerialization,
			Params:        map[string]string{},
		},
		Provider: &global.ProviderConfig{},
		applicationCompat: &config.ApplicationConfig{
			Name: "test-app",
		},
		srvOpts: &ServerOptions{
			Metrics: &global.MetricsConfig{},
			Otel: &global.OtelConfig{
				TracingConfig: &global.OtelTraceConfig{},
			},
		},
	}

	urlMap := svcOpts.getUrlMap()
	assert.Equal(t, "/var/log/access.log", urlMap.Get(constant.AccessLogFilterKey))
}

// Test postProcessConfig
func TestPostProcessConfig(t *testing.T) {
	svcOpts := &ServiceOptions{}
	url, _ := common.NewURL("dubbo://localhost:20880/test")

	svcOpts.postProcessConfig(url)
	assert.NotNil(t, url)
}

// Mock RPCService for testing
type mockRPCService struct{}

func (m *mockRPCService) Invoke(methodName string, params []any, results []any) error {
	return nil
}

func (m *mockRPCService) Reference() string {
	return "com.example.MockService"
}

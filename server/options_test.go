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
	"time"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test defaultServerOptions
func TestDefaultServerOptions(t *testing.T) {
	opts := defaultServerOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Application)
	assert.NotNil(t, opts.Provider)
	assert.NotNil(t, opts.Shutdown)
	assert.NotNil(t, opts.Metrics)
	assert.NotNil(t, opts.Otel)
	assert.NotNil(t, opts.TLS)
}

// Test ServerOptions.init with no options
func TestServerOptionsInit(t *testing.T) {
	opts := defaultServerOptions()
	err := opts.init()
	require.NoError(t, err)
	assert.NotNil(t, opts.Provider)
}

// Test ServerOptions.init with options
func TestServerOptionsInitWithOptions(t *testing.T) {
	opts := defaultServerOptions()
	testOpt := WithServerGroup("test-group")
	err := opts.init(testOpt)
	require.NoError(t, err)
	assert.Equal(t, "test-group", opts.Provider.Group)
}

// Test WithServerLoadBalanceConsistentHashing
func TestWithServerLoadBalanceConsistentHashing(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerLoadBalanceConsistentHashing()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyConsistentHashing, opts.Provider.Loadbalance)
}

// Test WithServerLoadBalanceLeastActive
func TestWithServerLoadBalanceLeastActive(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerLoadBalanceLeastActive()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyLeastActive, opts.Provider.Loadbalance)
}

// Test WithServerLoadBalanceRandom
func TestWithServerLoadBalanceRandom(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerLoadBalanceRandom()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyRandom, opts.Provider.Loadbalance)
}

// Test WithServerLoadBalanceRoundRobin
func TestWithServerLoadBalanceRoundRobin(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerLoadBalanceRoundRobin()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyRoundRobin, opts.Provider.Loadbalance)
}

// Test WithServerLoadBalanceP2C
func TestWithServerLoadBalanceP2C(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerLoadBalanceP2C()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyP2C, opts.Provider.Loadbalance)
}

// Test WithServerLoadBalance
func TestWithServerLoadBalance(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerLoadBalance("custom-lb")
	opt(opts)
	assert.Equal(t, "custom-lb", opts.Provider.Loadbalance)
}

// Test WithServerWarmUp
func TestWithServerWarmUp(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerWarmUp(60 * time.Second)
	opt(opts)
	assert.Equal(t, "60", opts.Provider.Warmup)
}

// Test WithServerClusterAvailable
func TestWithServerClusterAvailable(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterAvailable()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyAvailable, opts.Provider.Cluster)
}

// Test WithServerClusterBroadcast
func TestWithServerClusterBroadcast(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterBroadcast()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyBroadcast, opts.Provider.Cluster)
}

// Test WithServerClusterFailBack
func TestWithServerClusterFailBack(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterFailBack()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailback, opts.Provider.Cluster)
}

// Test WithServerClusterFailFast
func TestWithServerClusterFailFast(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterFailFast()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailfast, opts.Provider.Cluster)
}

// Test WithServerClusterFailOver
func TestWithServerClusterFailOver(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterFailOver()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailover, opts.Provider.Cluster)
}

// Test WithServerClusterFailSafe
func TestWithServerClusterFailSafe(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterFailSafe()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailsafe, opts.Provider.Cluster)
}

// Test WithServerClusterForking
func TestWithServerClusterForking(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterForking()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyForking, opts.Provider.Cluster)
}

// Test WithServerClusterZoneAware
func TestWithServerClusterZoneAware(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterZoneAware()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyZoneAware, opts.Provider.Cluster)
}

// Test WithServerClusterAdaptiveService
func TestWithServerClusterAdaptiveService(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerClusterAdaptiveService()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyAdaptiveService, opts.Provider.Cluster)
}

// Test WithServerCluster
func TestWithServerCluster(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerCluster("custom-cluster")
	opt(opts)
	assert.Equal(t, "custom-cluster", opts.Provider.Cluster)
}

// Test WithServerGroup
func TestWithServerGroup(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerGroup("test-group")
	opt(opts)
	assert.Equal(t, "test-group", opts.Provider.Group)
}

// Test WithServerVersion
func TestWithServerVersion(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerVersion("1.0.0")
	opt(opts)
	assert.Equal(t, "1.0.0", opts.Provider.Version)
}

// Test WithServerJSON
func TestWithServerJSON(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerJSON()
	opt(opts)
	assert.Equal(t, constant.JSONSerialization, opts.Provider.Serialization)
}

// Test WithServerToken
func TestWithServerToken(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerToken("test-token")
	opt(opts)
	assert.Equal(t, "test-token", opts.Provider.Token)
}

// Test WithServerNotRegister
func TestWithServerNotRegister(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerNotRegister()
	opt(opts)
	assert.True(t, opts.Provider.NotRegister)
}

// Test WithServerWarmup
func TestWithServerWarmup(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerWarmup(30 * time.Second)
	opt(opts)
	assert.Equal(t, "30s", opts.Provider.Warmup)
}

// Test WithServerRetries
func TestWithServerRetries(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerRetries(3)
	opt(opts)
	assert.Equal(t, "3", opts.Provider.Retries)
}

// Test WithServerSerialization
func TestWithServerSerialization(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerSerialization("protobuf")
	opt(opts)
	assert.Equal(t, "protobuf", opts.Provider.Serialization)
}

// Test WithServerAccesslog
func TestWithServerAccesslog(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerAccesslog("/var/log/access.log")
	opt(opts)
	assert.Equal(t, "/var/log/access.log", opts.Provider.AccessLog)
}

// Test WithServerTpsLimiter
func TestWithServerTpsLimiter(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerTpsLimiter("default")
	opt(opts)
	assert.Equal(t, "default", opts.Provider.TpsLimiter)
}

// Test WithServerTpsLimitRate
func TestWithServerTpsLimitRate(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerTpsLimitRate(1000)
	opt(opts)
	assert.Equal(t, "1000", opts.Provider.TpsLimitRate)
}

// Test WithServerTpsLimitStrategy
func TestWithServerTpsLimitStrategy(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerTpsLimitStrategy("adaptive")
	opt(opts)
	assert.Equal(t, "adaptive", opts.Provider.TpsLimitStrategy)
}

// Test WithServerTpsLimitRejectedHandler
func TestWithServerTpsLimitRejectedHandler(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerTpsLimitRejectedHandler("abort")
	opt(opts)
	assert.Equal(t, "abort", opts.Provider.TpsLimitRejectedHandler)
}

// Test WithServerExecuteLimit
func TestWithServerExecuteLimit(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerExecuteLimit("100")
	opt(opts)
	assert.Equal(t, "100", opts.Provider.ExecuteLimit)
}

// Test WithServerExecuteLimitRejectedHandler
func TestWithServerExecuteLimitRejectedHandler(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerExecuteLimitRejectedHandler("abort")
	opt(opts)
	assert.Equal(t, "abort", opts.Provider.ExecuteLimitRejectedHandler)
}

// Test WithServerAuth
func TestWithServerAuth(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerAuth("default")
	opt(opts)
	assert.Equal(t, "default", opts.Provider.Auth)
}

// Test WithServerParamSign
func TestWithServerParamSign(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerParamSign("md5")
	opt(opts)
	assert.Equal(t, "md5", opts.Provider.ParamSign)
}

// Test WithServerTag
func TestWithServerTag(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerTag("v1")
	opt(opts)
	assert.Equal(t, "v1", opts.Provider.Tag)
}

// Test WithServerParam
func TestWithServerParam(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerParam("key1", "value1")
	opt(opts)
	assert.Equal(t, "value1", opts.Provider.Params["key1"])
}

// Test WithServerParam creates params map if nil
func TestWithServerParamCreateMap(t *testing.T) {
	opts := defaultServerOptions()
	opts.Provider.Params = nil
	opt := WithServerParam("key1", "value1")
	opt(opts)
	assert.NotNil(t, opts.Provider.Params)
	assert.Equal(t, "value1", opts.Provider.Params["key1"])
}

// Test WithServerFilter
func TestWithServerFilter(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerFilter("echo")
	opt(opts)
	assert.Equal(t, "echo", opts.Provider.Filter)
}

// Test WithServerRegistryIDs
func TestWithServerRegistryIDs(t *testing.T) {
	opts := defaultServerOptions()
	registryIDs := []string{"registry1", "registry2"}
	opt := WithServerRegistryIDs(registryIDs)
	opt(opts)
	assert.Equal(t, registryIDs, opts.Provider.RegistryIDs)
}

// Test WithServerProtocolIDs
func TestWithServerProtocolIDs(t *testing.T) {
	opts := defaultServerOptions()
	protocolIDs := []string{"dubbo", "triple"}
	opt := WithServerProtocolIDs(protocolIDs)
	opt(opts)
	assert.Equal(t, protocolIDs, opts.Provider.ProtocolIDs)
}

// Test WithServerAdaptiveService
func TestWithServerAdaptiveService(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerAdaptiveService()
	opt(opts)
	assert.True(t, opts.Provider.AdaptiveService)
}

// Test WithServerAdaptiveServiceVerbose
func TestWithServerAdaptiveServiceVerbose(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerAdaptiveServiceVerbose()
	opt(opts)
	assert.True(t, opts.Provider.AdaptiveServiceVerbose)
}

// Test SetServerApplication
func TestSetServerApplication(t *testing.T) {
	opts := defaultServerOptions()
	appCfg := &global.ApplicationConfig{Name: "test-app"}
	opt := SetServerApplication(appCfg)
	opt(opts)
	assert.Equal(t, appCfg, opts.Application)
}

// Test SetServerRegistries
func TestSetServerRegistries(t *testing.T) {
	opts := defaultServerOptions()
	regs := map[string]*global.RegistryConfig{
		"reg1": {Protocol: "zookeeper"},
	}
	opt := SetServerRegistries(regs)
	opt(opts)
	assert.Equal(t, regs, opts.Registries)
}

// Test SetServerProtocols
func TestSetServerProtocols(t *testing.T) {
	opts := defaultServerOptions()
	pros := map[string]*global.ProtocolConfig{
		"dubbo": {Name: "dubbo"},
	}
	opt := SetServerProtocols(pros)
	opt(opts)
	assert.Equal(t, pros, opts.Protocols)
}

// Test SetServerShutdown
func TestSetServerShutdown(t *testing.T) {
	opts := defaultServerOptions()
	shutdown := &global.ShutdownConfig{Timeout: "10s"}
	opt := SetServerShutdown(shutdown)
	opt(opts)
	assert.Equal(t, shutdown, opts.Shutdown)
}

// Test SetServerMetrics
func TestSetServerMetrics(t *testing.T) {
	opts := defaultServerOptions()
	metrics := &global.MetricsConfig{}
	opt := SetServerMetrics(metrics)
	opt(opts)
	assert.Equal(t, metrics, opts.Metrics)
}

// Test SetServerOtel
func TestSetServerOtel(t *testing.T) {
	opts := defaultServerOptions()
	otel := &global.OtelConfig{}
	opt := SetServerOtel(otel)
	opt(opts)
	assert.Equal(t, otel, opts.Otel)
}

// Test SetServerTLS
func TestSetServerTLS(t *testing.T) {
	opts := defaultServerOptions()
	tlsCfg := &global.TLSConfig{}
	opt := SetServerTLS(tlsCfg)
	opt(opts)
	assert.Equal(t, tlsCfg, opts.TLS)
}

// Test SetServerProvider
func TestSetServerProvider(t *testing.T) {
	opts := defaultServerOptions()
	provider := &global.ProviderConfig{}
	opt := SetServerProvider(provider)
	opt(opts)
	assert.Equal(t, provider, opts.Provider)
}

// ==================== ServiceOptions Tests ====================

// Test defaultServiceOptions
func TestDefaultServiceOptions(t *testing.T) {
	opts := defaultServiceOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Service)
	assert.NotNil(t, opts.Application)
	assert.False(t, opts.unexported.Load())
	assert.False(t, opts.exported.Load())
	assert.True(t, opts.needExport)
}

// Test WithInterface
func TestWithInterface(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithInterface("com.example.Service")
	opt(opts)
	assert.Equal(t, "com.example.Service", opts.Service.Interface)
}

// Test WithRegistryIDs
func TestWithRegistryIDs(t *testing.T) {
	opts := defaultServiceOptions()
	registryIDs := []string{"registry1"}
	opt := WithRegistryIDs(registryIDs)
	opt(opts)
	assert.NotEqual(t, registryIDs, opts.Service.RegistryIDs)
}

// Test WithFilter
func TestWithFilter(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithFilter("echo,access_log")
	opt(opts)
	assert.Equal(t, "echo,access_log", opts.Service.Filter)
}

// Test WithLoadBalanceConsistentHashing
func TestWithLoadBalanceConsistentHashing(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithLoadBalanceConsistentHashing()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyConsistentHashing, opts.Service.Loadbalance)
}

// Test WithLoadBalanceLeastActive
func TestWithLoadBalanceLeastActive(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithLoadBalanceLeastActive()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyLeastActive, opts.Service.Loadbalance)
}

// Test WithLoadBalanceRandom
func TestWithLoadBalanceRandom(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithLoadBalanceRandom()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyRandom, opts.Service.Loadbalance)
}

// Test WithLoadBalanceRoundRobin
func TestWithLoadBalanceRoundRobin(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithLoadBalanceRoundRobin()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyRoundRobin, opts.Service.Loadbalance)
}

// Test WithLoadBalanceP2C
func TestWithLoadBalanceP2C(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithLoadBalanceP2C()
	opt(opts)
	assert.Equal(t, constant.LoadBalanceKeyP2C, opts.Service.Loadbalance)
}

// Test WithLoadBalance
func TestWithLoadBalance(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithLoadBalance("custom-lb")
	opt(opts)
	assert.Equal(t, "custom-lb", opts.Service.Loadbalance)
}

// Test WithWarmUp
func TestWithWarmUp(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithWarmUp(60 * time.Second)
	opt(opts)
	assert.Equal(t, "60", opts.Service.Warmup)
}

// Test WithClusterAvailable
func TestWithClusterAvailable(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterAvailable()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyAvailable, opts.Service.Cluster)
}

// Test WithClusterBroadcast
func TestWithClusterBroadcast(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterBroadcast()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyBroadcast, opts.Service.Cluster)
}

// Test WithClusterFailBack
func TestWithClusterFailBack(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterFailBack()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailback, opts.Service.Cluster)
}

// Test WithClusterFailFast
func TestWithClusterFailFast(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterFailFast()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailfast, opts.Service.Cluster)
}

// Test WithClusterFailOver
func TestWithClusterFailOver(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterFailOver()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailover, opts.Service.Cluster)
}

// Test WithClusterFailSafe
func TestWithClusterFailSafe(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterFailSafe()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyFailsafe, opts.Service.Cluster)
}

// Test WithClusterForking
func TestWithClusterForking(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterForking()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyForking, opts.Service.Cluster)
}

// Test WithClusterZoneAware
func TestWithClusterZoneAware(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterZoneAware()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyZoneAware, opts.Service.Cluster)
}

// Test WithClusterAdaptiveService
func TestWithClusterAdaptiveService(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithClusterAdaptiveService()
	opt(opts)
	assert.Equal(t, constant.ClusterKeyAdaptiveService, opts.Service.Cluster)
}

// Test WithCluster
func TestWithCluster(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithCluster("custom-cluster")
	opt(opts)
	assert.Equal(t, "custom-cluster", opts.Service.Cluster)
}

// Test WithGroup
func TestWithGroup(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithGroup("test-group")
	opt(opts)
	assert.Equal(t, "test-group", opts.Service.Group)
}

// Test WithVersion
func TestWithVersion(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithVersion("1.0.0")
	opt(opts)
	assert.Equal(t, "1.0.0", opts.Service.Version)
}

// Test WithJSON
func TestWithJSON(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithJSON()
	opt(opts)
	assert.Equal(t, constant.JSONSerialization, opts.Service.Serialization)
}

// Test WithToken
func TestWithToken(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithToken("test-token")
	opt(opts)
	assert.Equal(t, "test-token", opts.Service.Token)
}

// Test WithNotRegister
func TestWithNotRegister(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithNotRegister()
	opt(opts)
	assert.True(t, opts.Service.NotRegister)
}

// Test WithWarmup
func TestWithWarmup(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithWarmup(30 * time.Second)
	opt(opts)
	assert.Equal(t, "30s", opts.Service.Warmup)
}

// Test WithRetries
func TestWithRetries(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithRetries(3)
	opt(opts)
	assert.Equal(t, "3", opts.Service.Retries)
}

// Test WithSerialization
func TestWithSerialization(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithSerialization("protobuf")
	opt(opts)
	assert.Equal(t, "protobuf", opts.Service.Serialization)
}

// Test WithAccesslog
func TestWithAccesslog(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithAccesslog("/var/log/access.log")
	opt(opts)
	assert.Equal(t, "/var/log/access.log", opts.Service.AccessLog)
}

// Test WithTpsLimiter
func TestWithTpsLimiter(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithTpsLimiter("default")
	opt(opts)
	assert.Equal(t, "default", opts.Service.TpsLimiter)
}

// Test WithTpsLimitRate
func TestWithTpsLimitRate(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithTpsLimitRate(1000)
	opt(opts)
	assert.Equal(t, "1000", opts.Service.TpsLimitRate)
}

// Test WithTpsLimitStrategy
func TestWithTpsLimitStrategy(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithTpsLimitStrategy("adaptive")
	opt(opts)
	assert.Equal(t, "adaptive", opts.Service.TpsLimitStrategy)
}

// Test WithTpsLimitRejectedHandler
func TestWithTpsLimitRejectedHandler(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithTpsLimitRejectedHandler("abort")
	opt(opts)
	assert.Equal(t, "abort", opts.Service.TpsLimitRejectedHandler)
}

// Test WithExecuteLimit
func TestWithExecuteLimit(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithExecuteLimit("100")
	opt(opts)
	assert.Equal(t, "100", opts.Service.ExecuteLimit)
}

// Test WithExecuteLimitRejectedHandler
func TestWithExecuteLimitRejectedHandler(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithExecuteLimitRejectedHandler("abort")
	opt(opts)
	assert.Equal(t, "abort", opts.Service.ExecuteLimitRejectedHandler)
}

// Test WithAuth
func TestWithAuth(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithAuth("default")
	opt(opts)
	assert.Equal(t, "default", opts.Service.Auth)
}

// Test WithParamSign
func TestWithParamSign(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithParamSign("md5")
	opt(opts)
	assert.Equal(t, "md5", opts.Service.ParamSign)
}

// Test WithTag
func TestWithTag(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithTag("v1")
	opt(opts)
	assert.Equal(t, "v1", opts.Service.Tag)
}

// Test WithParam
func TestWithParam(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithParam("key1", "value1")
	opt(opts)
	assert.Equal(t, "value1", opts.Service.Params["key1"])
}

// Test WithParam creates params map if nil
func TestWithParamCreateMap(t *testing.T) {
	opts := defaultServiceOptions()
	opts.Service.Params = nil
	opt := WithParam("key1", "value1")
	opt(opts)
	assert.NotNil(t, opts.Service.Params)
	assert.Equal(t, "value1", opts.Service.Params["key1"])
}

// Test WithIDLMode
func TestWithIDLMode(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithIDLMode(constant.IDL)
	opt(opts)
	assert.Equal(t, constant.IDL, opts.IDLMode)
}

// Test SetApplication
func TestSetApplication(t *testing.T) {
	opts := defaultServiceOptions()
	appCfg := &global.ApplicationConfig{Name: "test-app"}
	opt := SetApplication(appCfg)
	opt(opts)
	assert.Equal(t, appCfg, opts.Application)
}

// Test SetProvider
func TestSetProvider(t *testing.T) {
	opts := defaultServiceOptions()
	provider := &global.ProviderConfig{}
	opt := SetProvider(provider)
	opt(opts)
	assert.Equal(t, provider, opts.Provider)
}

// Test SetService
func TestSetService(t *testing.T) {
	opts := defaultServiceOptions()
	service := &global.ServiceConfig{Interface: "com.example.Service"}
	opt := SetService(service)
	opt(opts)
	assert.Equal(t, service, opts.Service)
}

// Test SetRegistries
func TestSetRegistries(t *testing.T) {
	opts := defaultServiceOptions()
	regs := map[string]*global.RegistryConfig{
		"reg1": {Protocol: "zookeeper"},
	}
	opt := SetRegistries(regs)
	opt(opts)
	assert.Equal(t, regs, opts.Registries)
}

// Test SetProtocols
func TestSetProtocols(t *testing.T) {
	opts := defaultServiceOptions()
	pros := map[string]*global.ProtocolConfig{
		"dubbo": {Name: "dubbo"},
	}
	opt := SetProtocols(pros)
	opt(opts)
	assert.Equal(t, pros, opts.Protocols)
}

// Test WithServerRegistry
func TestWithServerRegistry(t *testing.T) {
	opts := defaultServerOptions()

	// Use registry.WithZookeeper() to create a proper registry option
	opt := WithServerRegistry(registry.WithZookeeper(), registry.WithID("test-registry"))
	opt(opts)

	assert.NotNil(t, opts.Registries)
	assert.Contains(t, opts.Registries, "test-registry")
	assert.Equal(t, "zookeeper", opts.Registries["test-registry"].Protocol)
}

// Test WithServerProtocol
func TestWithServerProtocol(t *testing.T) {
	opts := defaultServerOptions()

	// Use protocol options to create a proper protocol configuration
	opt := WithServerProtocol(protocol.WithDubbo(), protocol.WithID("test-protocol"))
	opt(opts)

	assert.NotNil(t, opts.Protocols)
	assert.Contains(t, opts.Protocols, "test-protocol")
	assert.Equal(t, "dubbo", opts.Protocols["test-protocol"].Name)
}

// Test WithServerFilterConf
func TestWithServerFilterConf(t *testing.T) {
	opts := defaultServerOptions()
	conf := map[string]any{"key": "value"}
	opt := WithServerFilterConf(conf)
	opt(opts)
	assert.Equal(t, conf, opts.Provider.FilterConf)
}

// Test WithServerTLSOption
func TestWithServerTLSOption(t *testing.T) {
	opts := defaultServerOptions()
	opt := WithServerTLSOption()
	opt(opts)
	assert.NotNil(t, opts.TLS)
}

// Test WithProtocol for ServiceOptions
func TestWithProtocol(t *testing.T) {
	opts := defaultServiceOptions()

	// Use protocol options to create a proper protocol configuration
	opt := WithProtocol(protocol.WithTriple(), protocol.WithID("test-protocol"))
	opt(opts)

	assert.NotNil(t, opts.Protocols)
	assert.Contains(t, opts.Protocols, "test-protocol")
	assert.Equal(t, "tri", opts.Protocols["test-protocol"].Name)
}

// Test WithRegistry for ServiceOptions
func TestWithRegistry(t *testing.T) {
	opts := defaultServiceOptions()

	// Use registry.WithNacos() to create a proper registry option
	opt := WithRegistry(registry.WithNacos(), registry.WithID("test-registry"))
	opt(opts)

	assert.NotNil(t, opts.Registries)
	assert.Contains(t, opts.Registries, "test-registry")
	assert.Equal(t, "nacos", opts.Registries["test-registry"].Protocol)
}

// Test WithMethod
func TestWithMethod(t *testing.T) {
	opts := defaultServiceOptions()
	opt := WithMethod()
	opt(opts)
	assert.NotNil(t, opts.Service.Methods)
	// Verify that a method was actually added
	assert.Len(t, opts.Service.Methods, 1)
}

// Test WithProtocolIDs for ServiceOptions
func TestWithProtocolIDs(t *testing.T) {
	opts := defaultServiceOptions()
	protocolIDs := []string{"dubbo"}
	opt := WithProtocolIDs(protocolIDs)
	opt(opts)
	assert.NotEqual(t, protocolIDs, opts.Service.ProtocolIDs)
}

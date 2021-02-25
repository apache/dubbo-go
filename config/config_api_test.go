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

package config

import (
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultServiceConfig(t *testing.T) {
	serviceConfig := NewServiceConfigByAPI(
		WithServiceCluster("test-cluster"),
		WithServiceInterface("test-interface"),
		WithServiceLoadBalance("test-loadbalance"),
		WithServiceMethod("test-method1", "test-retries1", "test-lb1"),
		WithServiceMethod("test-method2", "test-retries2", "test-lb2"),
		WithServiceMethod("test-method3", "test-retries3", "test-lb3"),
		WithServiceProtocol("test-protocol"),
		WithServiceRegistry("test-registry"),
		WithServiceWarmUpTime("test-warmup"),
	)
	assert.Equal(t, serviceConfig.Cluster, "test-cluster")
	assert.Equal(t, serviceConfig.InterfaceName, "test-interface")
	assert.Equal(t, serviceConfig.Loadbalance, "test-loadbalance")
	for i, v := range serviceConfig.Methods {
		backFix := strconv.Itoa(i + 1)
		assert.Equal(t, v.Name, "test-method"+backFix)
		assert.Equal(t, v.Retries, "test-retries"+backFix)
		assert.Equal(t, v.LoadBalance, "test-lb"+backFix)
	}
	assert.Equal(t, serviceConfig.Protocol, "test-protocol")
	assert.Equal(t, serviceConfig.Registry, "test-registry")
	assert.Equal(t, serviceConfig.Warmup, "test-warmup")
}

func TestNewReferenceConfigByAPI(t *testing.T) {
	refConfig := NewReferenceConfigByAPI(
		WithReferenceCluster("test-cluster"),
		WithReferenceInterface("test-interface"),
		WithReferenceMethod("test-method1", "test-retries1", "test-lb1"),
		WithReferenceMethod("test-method2", "test-retries2", "test-lb2"),
		WithReferenceMethod("test-method3", "test-retries3", "test-lb3"),
		WithReferenceProtocol("test-protocol"),
		WithReferenceRegistry("test-registry"),
	)
	assert.Equal(t, refConfig.Cluster, "test-cluster")
	assert.Equal(t, refConfig.InterfaceName, "test-interface")
	for i, v := range refConfig.Methods {
		backFix := strconv.Itoa(i + 1)
		assert.Equal(t, v.Name, "test-method"+backFix)
		assert.Equal(t, v.Retries, "test-retries"+backFix)
		assert.Equal(t, v.LoadBalance, "test-lb"+backFix)
	}
	assert.Equal(t, refConfig.Protocol, "test-protocol")
	assert.Equal(t, refConfig.Registry, "test-registry")
}

func TestNewRegistryConfig(t *testing.T) {
	regConfig := NewRegistryConfig(
		WithRegistryTimeOut("test-timeout"),
		WithRegistryProtocol("test-protocol"),
		WithRegistryGroup("test-group"),
		WithRegistryAddress("test-address"),
		WithRegistrySimplified(true),
		WithRegistryUserName("test-username"),
		WithRegistryPassword("test-password"),
	)
	assert.Equal(t, regConfig.TimeoutStr, "test-timeout")
	assert.Equal(t, regConfig.Protocol, "test-protocol")
	assert.Equal(t, regConfig.Group, "test-group")
	assert.Equal(t, regConfig.Address, "test-address")
	assert.Equal(t, regConfig.Simplified, true)
	assert.Equal(t, regConfig.Username, "test-username")
	assert.Equal(t, regConfig.Password, "test-password")
}

func TestNewConsumerConfig(t *testing.T) {
	referConfig := NewReferenceConfigByAPI(
		WithReferenceCluster("test-cluster"),
		WithReferenceInterface("test-interface"),
		WithReferenceMethod("test-method1", "test-retries1", "test-lb1"),
		WithReferenceMethod("test-method2", "test-retries2", "test-lb2"),
		WithReferenceMethod("test-method3", "test-retries3", "test-lb3"),
		WithReferenceProtocol("test-protocol"),
		WithReferenceRegistry("test-registry"),
	)
	defaultZKRegistry := NewDefaultRegistryConfig("zookeeper")
	assert.Equal(t, defaultZKRegistry.Address, defaultZKAddr)
	assert.Equal(t, defaultZKRegistry.Protocol, "zookeeper")
	assert.Equal(t, defaultZKRegistry.TimeoutStr, defaultRegistryTimeout)

	testConsumerConfig := NewConsumerConfig(
		WithConsumerConfigCheck(true),
		WithConsumerConnTimeout(time.Minute),
		WithConsumerRequestTimeout(time.Hour),
		WithConsumerReferenceConfig("UserProvider", referConfig),
		WithConsumerRegistryConfig("demoZK", defaultZKRegistry),
	)

	assert.Equal(t, *testConsumerConfig.Check, true)
	assert.Equal(t, testConsumerConfig.ConnectTimeout, time.Minute)
	assert.Equal(t, testConsumerConfig.RequestTimeout, time.Hour)
	assert.Equal(t, testConsumerConfig.Registries["demoZK"], defaultZKRegistry)
	assert.Equal(t, testConsumerConfig.References["UserProvider"], referConfig)
}

func TestNewProviderConfig(t *testing.T) {
	serviceConfig := NewServiceConfigByAPI(
		WithServiceCluster("test-cluster"),
		WithServiceInterface("test-interface"),
		WithServiceLoadBalance("test-loadbalance"),
		WithServiceMethod("test-method1", "test-retries1", "test-lb1"),
		WithServiceMethod("test-method2", "test-retries2", "test-lb2"),
		WithServiceMethod("test-method3", "test-retries3", "test-lb3"),
		WithServiceProtocol("test-protocol"),
		WithServiceRegistry("test-registry"),
		WithServiceWarmUpTime("test-warmup"),
	)

	defaultConsulRegistry := NewDefaultRegistryConfig("consul")
	assert.Equal(t, defaultConsulRegistry.Address, defaultConsulAddr)
	assert.Equal(t, defaultConsulRegistry.Protocol, "consul")
	assert.Equal(t, defaultConsulRegistry.TimeoutStr, defaultRegistryTimeout)

	defaultNacosRegistry := NewDefaultRegistryConfig("nacos")
	assert.Equal(t, defaultNacosRegistry.Address, defaultNacosAddr)
	assert.Equal(t, defaultNacosRegistry.Protocol, "nacos")
	assert.Equal(t, defaultNacosRegistry.TimeoutStr, defaultRegistryTimeout)

	testProviderConfig := NewProviderConfig(
		WithProviderServices("UserProvider", serviceConfig),
		WithProviderProtocol("dubbo", "dubbo", "20000"),
		WithProviderRegistry("demoConsul", defaultConsulRegistry),
		WithProviderRegistry("demoNacos", defaultNacosRegistry),
	)

	assert.NotNil(t, testProviderConfig.Services)
	for k, v := range testProviderConfig.Services {
		assert.Equal(t, k, "UserProvider")
		assert.Equal(t, v, serviceConfig)
	}
	assert.NotNil(t, testProviderConfig.Registries)
	i := 0
	for k, v := range testProviderConfig.Registries {
		if i == 0 {
			assert.Equal(t, k, "demoConsul")
			assert.Equal(t, v, defaultConsulRegistry)
			i++
		} else {
			assert.Equal(t, k, "demoNacos")
			assert.Equal(t, v, defaultNacosRegistry)
		}
	}

	assert.NotNil(t, testProviderConfig.Protocols)
	assert.Equal(t, testProviderConfig.Protocols["dubbo"].Name, "dubbo")
	assert.Equal(t, testProviderConfig.Protocols["dubbo"].Port, "20000")
}

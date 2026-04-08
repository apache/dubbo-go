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
	"fmt"
	"sync"
	"testing"
)

import (
	"github.com/dubbogo/gost/encoding/yaml"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

func TestGoConfigProcess(t *testing.T) {
	rc := &RootConfigBuilder{rootConfig: newEmptyRootConfig()}
	r := &RegistryConfig{Protocol: "zookeeper", Timeout: "10s", Address: "127.0.0.1:2181"}
	rc.AddRegistry("demoZK", r)
	SetRootConfig(*rc.rootConfig)

	// test koan.UnmarshalWithConf error
	b := "dubbo:\n  registries:\n    demoZK:\n      protocol: zookeeper\n      timeout: 11s\n      address: 127.0.0.1:2181\n      simplified: abc123"
	c2 := &config_center.ConfigChangeEvent{Key: "test", Value: b}
	GetRootConfig().Process(c2)
	assert.Equal(t, "10s", GetRootConfig().Registries["demoZK"].Timeout)

	// test update registry time out
	bs, _ := yaml.LoadYMLConfig("./testdata/root_config_test.yml")
	c := &config_center.ConfigChangeEvent{Key: "test", Value: string(bs)}
	GetRootConfig().Process(c)
	assert.Equal(t, "11s", GetRootConfig().Registries["demoZK"].Timeout)
	assert.Equal(t, "6s", GetRootConfig().Consumer.RequestTimeout)

}

func TestNewRootConfigBuilder(t *testing.T) {
	registryConfig := NewRegistryConfigWithProtocolDefaultPort("nacos")
	protocolConfig := NewProtocolConfigBuilder().
		SetName("dubbo").
		SetPort("20000").
		Build()
	newConfig := NewRootConfigBuilder().
		SetConfigCenter(NewConfigCenterConfigBuilder().Build()).
		SetMetadataReport(NewMetadataReportConfigBuilder().Build()).
		AddProtocol("dubbo", protocolConfig).
		AddRegistry("nacos", registryConfig).
		SetProtocols(map[string]*ProtocolConfig{"dubbo": protocolConfig}).
		SetRegistries(map[string]*RegistryConfig{"nacos": registryConfig}).
		SetProvider(NewProviderConfigBuilder().Build()).
		SetConsumer(NewConsumerConfigBuilder().Build()).
		SetMetric(NewMetricConfigBuilder().Build()).
		SetLogger(NewLoggerConfigBuilder().Build()).
		SetShutdown(NewShutDownConfigBuilder().Build()).
		SetShutDown(NewShutDownConfigBuilder().Build()).
		SetRouter([]*RouterConfig{}).
		SetEventDispatcherType("direct").
		SetCacheFile("abc=123").
		Build()

	SetRootConfig(*newConfig)

	assert.Equal(t, constant.Dubbo, newConfig.Prefix())
	ids := newConfig.getRegistryIds()
	assert.Equal(t, "nacos", ids[0])

	down := GetShutDown()
	assert.NotNil(t, down)

	application := GetApplicationConfig()
	assert.NotNil(t, application)

	registerPOJO()
	config := GetRootConfig()
	assert.Equal(t, newConfig, config)
}

func TestRootConfigConcurrentSetAndGet(t *testing.T) {
	SetRootConfig(*NewRootConfigBuilder().Build())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			cfg := NewRootConfigBuilder().Build()
			cfg.Application.Name = fmt.Sprintf("app-%d", idx)
			SetRootConfig(*cfg)
		}(i)
		go func() {
			defer wg.Done()
			_ = GetRootConfig()
		}()
	}
	wg.Wait()
	finalCfg := GetRootConfig()
	assert.NotNil(t, finalCfg)
	assert.NotEmpty(t, finalCfg.Application.Name)
}

func TestRootConfigProcessCopyOnWrite(t *testing.T) {
	base := NewRootConfigBuilder().Build()
	base.Registries["demoZK"] = &RegistryConfig{
		Protocol: "zookeeper",
		Timeout:  "10s",
		Address:  "127.0.0.1:2181",
	}
	base.Consumer.RequestTimeout = "3s"
	SetRootConfig(*base)

	oldSnapshot := GetRootConfig()
	eventValue := "dubbo:\n  registries:\n    demoZK:\n      protocol: zookeeper\n      timeout: 11s\n      address: 127.0.0.1:2181\n  consumer:\n    request-timeout: 6s\n"
	oldSnapshot.Process(&config_center.ConfigChangeEvent{Key: "test", Value: eventValue})

	newSnapshot := GetRootConfig()
	assert.NotSame(t, oldSnapshot, newSnapshot)
	assert.Equal(t, "10s", oldSnapshot.Registries["demoZK"].Timeout)
	assert.Equal(t, "3s", oldSnapshot.Consumer.RequestTimeout)
	assert.Equal(t, "11s", newSnapshot.Registries["demoZK"].Timeout)
	assert.Equal(t, "6s", newSnapshot.Consumer.RequestTimeout)
}

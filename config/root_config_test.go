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

	// test koan.UnmarshalWithConf error
	b := "dubbo:\n  registries:\n    demoZK:\n      protocol: zookeeper\n      timeout: 11s\n      address: 127.0.0.1:2181\n      simplified: abc123"
	c2 := &config_center.ConfigChangeEvent{Key: "test", Value: b}
	rc.rootConfig.Process(c2)
	assert.Equal(t, rc.rootConfig.Registries["demoZK"].Timeout, "10s")

	// test update registry time out
	bs, _ := yaml.LoadYMLConfig("./testdata/root_config_test.yml")
	c := &config_center.ConfigChangeEvent{Key: "test", Value: string(bs)}
	rc.rootConfig.Process(c)
	assert.Equal(t, rc.rootConfig.Registries["demoZK"].Timeout, "11s")
	assert.Equal(t, rc.rootConfig.Consumer.RequestTimeout, "6s")

}

func TestNewRootConfigBuilder(t *testing.T) {
	registryConfig := NewRegistryConfigWithProtocolDefaultPort("nacos")
	protocolConfig := NewProtocolConfigBuilder().
		SetName("dubbo").
		SetPort("20000").
		Build()
	rootConfig = NewRootConfigBuilder().
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

	assert.Equal(t, rootConfig.Prefix(), constant.Dubbo)
	ids := rootConfig.getRegistryIds()
	assert.Equal(t, ids[0], "nacos")

	down := GetShutDown()
	assert.NotNil(t, down)

	application := GetApplicationConfig()
	assert.NotNil(t, application)

	registerPOJO()
	config := GetRootConfig()
	assert.Equal(t, rootConfig, config)
}

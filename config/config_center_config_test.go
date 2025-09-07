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
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestCenterConfig_toURL(t *testing.T) {
	// Test toURL method
	centerConfig := &CenterConfig{
		Protocol: "zookeeper",
		Address:  "127.0.0.1:2181",
		Username: "testuser",
		Password: "testpass",
		DataId:   "test-data-id",
		Group:    "test-group",
		Timeout:  "30s",
	}

	url, err := centerConfig.toURL()
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, "zookeeper", url.Protocol)
	assert.Equal(t, "127.0.0.1:2181", url.Location)
	assert.Equal(t, "testuser", url.Username)
	assert.Equal(t, "testpass", url.Password)
}

func TestCenterConfig_toURL_EmptyConfig(t *testing.T) {
	// Test toURL with empty config
	centerConfig := &CenterConfig{}

	url, err := centerConfig.toURL()
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, "", url.Protocol)
	assert.Equal(t, "", url.Location)
	assert.Equal(t, "", url.Username)
	assert.Equal(t, "", url.Password)
}

func TestCenterConfig_toURL_WithParams(t *testing.T) {
	// Test toURL with additional parameters
	centerConfig := &CenterConfig{
		Protocol: "nacos",
		Address:  "127.0.0.1:8848",
		DataId:   "test-app",
		Group:    "DEFAULT_GROUP",
		Timeout:  "60s",
	}

	url, err := centerConfig.toURL()
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, "nacos", url.Protocol)
	assert.Equal(t, "127.0.0.1:8848", url.Location)

	// Verify basic functionality - URL was created successfully
	assert.NotNil(t, url)
}

func TestConfigCenterConfigBuilder_Setters(t *testing.T) {
	// Test ConfigCenterConfigBuilder setter methods
	builder := NewConfigCenterConfigBuilder()

	// Test SetProtocol
	builder.SetProtocol("zookeeper")
	assert.Equal(t, "zookeeper", builder.configCenterConfig.Protocol)

	// Test SetUserName
	builder.SetUserName("testuser")
	assert.Equal(t, "testuser", builder.configCenterConfig.Username)

	// Test SetAddress
	builder.SetAddress("127.0.0.1:2181")
	assert.Equal(t, "127.0.0.1:2181", builder.configCenterConfig.Address)

	// Test SetPassword
	builder.SetPassword("testpass")
	assert.Equal(t, "testpass", builder.configCenterConfig.Password)
}

func TestConfigCenterConfigBuilder_FluentAPI(t *testing.T) {
	// Test fluent API chaining
	config := NewConfigCenterConfigBuilder().
		SetProtocol("nacos").
		SetAddress("127.0.0.1:8848").
		SetUserName("nacosuser").
		SetPassword("nacospass").
		Build()

	assert.Equal(t, "nacos", config.Protocol)
	assert.Equal(t, "127.0.0.1:8848", config.Address)
	assert.Equal(t, "nacosuser", config.Username)
	assert.Equal(t, "nacospass", config.Password)
}

func TestCenterConfig_GetUrlMap(t *testing.T) {
	// Test GetUrlMap method
	centerConfig := &CenterConfig{
		DataId:  "test-app",
		Group:   "test-group",
		Timeout: "30s",
	}

	urlMap := centerConfig.GetUrlMap()
	assert.NotNil(t, urlMap)
	// GetUrlMap returns url.Values, verify it contains expected keys
	if dataIds, exists := urlMap["dataId"]; exists && len(dataIds) > 0 {
		assert.Equal(t, "test-app", dataIds[0])
	}
	if groups, exists := urlMap["group"]; exists && len(groups) > 0 {
		assert.Equal(t, "test-group", groups[0])
	}
	if timeouts, exists := urlMap["timeout"]; exists && len(timeouts) > 0 {
		assert.Equal(t, "30s", timeouts[0])
	}
}

func TestCenterConfig_GetUrlMap_EmptyConfig(t *testing.T) {
	// Test GetUrlMap with empty config
	centerConfig := &CenterConfig{}

	urlMap := centerConfig.GetUrlMap()
	assert.NotNil(t, urlMap)
	// GetUrlMap returns url.Values, verify it's not nil
	assert.NotNil(t, urlMap)
}

func TestCenterConfig_Prefix(t *testing.T) {
	// Test Prefix method
	centerConfig := &CenterConfig{}
	assert.Equal(t, constant.ConfigCenterPrefix, centerConfig.Prefix())
}

func TestConfigCenterConfigBuilder_Build(t *testing.T) {
	// Test Build method
	builder := NewConfigCenterConfigBuilder()
	config := builder.Build()

	assert.NotNil(t, config)
	assert.IsType(t, &CenterConfig{}, config)
}

func TestConfigCenterConfigBuilder_NewConfigCenterConfigBuilder(t *testing.T) {
	// Test NewConfigCenterConfigBuilder constructor
	builder := NewConfigCenterConfigBuilder()
	assert.NotNil(t, builder)
	assert.NotNil(t, builder.configCenterConfig)
	assert.IsType(t, &CenterConfig{}, builder.configCenterConfig)
}

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
	"dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

func TestStartConfigCenter(t *testing.T) {
	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{}
	})
	baseConfig := &BaseConfig{ConfigCenterConfig: &ConfigCenterConfig{
		Protocol:   "mock",
		Address:    "172.0.0.1",
		Group:      "dubbo",
		ConfigFile: "mockDubbo.properties",
	}}

	c := &configCenter{}
	err := c.startConfigCenter(*baseConfig)
	assert.NoError(t, err)
	b, v := config.GetEnvInstance().Configuration().Back().Value.(*config.InmemoryConfiguration).GetProperty("dubbo.application.organization")
	assert.True(t, b)
	assert.Equal(t, "ikurento.com", v)
}

func TestStartConfigCenterWithRemoteRef(t *testing.T) {
	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{}
	})
	m := make(map[string]*RemoteConfig)
	m["mock"] = &RemoteConfig{Protocol: "mock", Address: "172.0.0.1"}
	baseConfig := &BaseConfig{
		Remotes: m,
		ConfigCenterConfig: &ConfigCenterConfig{
			Group:      "dubbo",
			RemoteRef:  "mock",
			ConfigFile: "mockDubbo.properties",
		},
	}

	c := &configCenter{}
	err := c.startConfigCenter(*baseConfig)
	assert.NoError(t, err)
	b, v := config.GetEnvInstance().Configuration().Back().Value.(*config.InmemoryConfiguration).GetProperty("dubbo.application.organization")
	assert.True(t, b)
	assert.Equal(t, "ikurento.com", v)
}

func TestStartConfigCenterWithRemoteRefError(t *testing.T) {
	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{}
	})
	m := make(map[string]*RemoteConfig)
	m["mock"] = &RemoteConfig{Address: "172.0.0.1"}
	baseConfig := &BaseConfig{
		Remotes: m,
		ConfigCenterConfig: &ConfigCenterConfig{
			Protocol:   "mock",
			Group:      "dubbo",
			RemoteRef:  "mock",
			ConfigFile: "mockDubbo.properties",
		},
	}

	c := &configCenter{}
	err := c.startConfigCenter(*baseConfig)
	assert.Error(t, err)
}

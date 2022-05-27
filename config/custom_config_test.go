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
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestCustomInit(t *testing.T) {
	t.Run("empty use default", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/custom/empty.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		customConfig := rootConfig.Custom
		assert.NotNil(t, customConfig)
		assert.Equal(t, customConfig.ConfigMap, map[string]interface{}(nil))
		assert.Equal(t, customConfig.GetDefineValue("test", "test"), "test")
		assert.Equal(t, GetDefineValue("test", "test"), "test")
	})

	t.Run("use config", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/custom/custom.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		customConfig := rootConfig.Custom
		assert.NotNil(t, customConfig)
		assert.Equal(t, customConfig.ConfigMap, map[string]interface{}{"test-config": true})
		assert.Equal(t, customConfig.GetDefineValue("test-config", false), true)
		assert.Equal(t, customConfig.GetDefineValue("test-no-config", false), false)
		assert.Equal(t, GetDefineValue("test-config", false), true)
		assert.Equal(t, GetDefineValue("test-no-config", false), false)
	})

	t.Run("config builder", func(t *testing.T) {
		customConfigBuilder := NewCustomConfigBuilder()
		customConfigBuilder.SetDefineConfig("test-build", true)
		customConfig := customConfigBuilder.Build()
		assert.NotNil(t, customConfig)
		assert.Equal(t, customConfig.GetDefineValue("test-build", false), true)
		assert.Equal(t, customConfig.GetDefineValue("test-no-build", false), false)
		// todo @(laurence) now we should guarantee rootConfig ptr can't be changed during test
		tempRootConfig := rootConfig
		rt := NewRootConfigBuilder().SetCustom(customConfig).Build()
		SetRootConfig(*rt)
		assert.Equal(t, GetDefineValue("test-build", false), true)
		assert.Equal(t, GetDefineValue("test-no-build", false), false)
		SetRootConfig(*tempRootConfig)
	})
}

func TestConfigUtils(t *testing.T) {
	config := NewRegistryConfigWithProtocolDefaultPort("nacos")

	id := clientNameID(config, config.Protocol, config.Address)

	assert.Equal(t, id, strings.Join([]string{constant.RegistryConfigPrefix, "nacos", "127.0.0.1:8848"}, "-"))

	ids := translateIds([]string{"nacos,zk"})
	assert.Equal(t, ids[0], "nacos")
	assert.Equal(t, ids[1], "zk")

	element := removeDuplicateElement([]string{"nacos", "nacos"})
	assert.Equal(t, len(element), 1)
}

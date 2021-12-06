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

func TestCustomInit(t *testing.T) {
	t.Run("empty use default", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/custom/empty.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		CustomConfig := rootConfig.Custom
		assert.NotNil(t, CustomConfig)
		assert.Equal(t, CustomConfig.Name, "user-config")
		assert.Equal(t, CustomConfig.Version, "v1.0")
		assert.Equal(t, CustomConfig.DefineConfig, map[string]interface{}(nil))
		assert.Equal(t, CustomConfig.GetDefineValue("test", "test"), "test")
		assert.Equal(t, GetDefineValue("test", "test"), "test")
	})

	t.Run("use config", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/custom/custom.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		CustomConfig := rootConfig.Custom
		assert.NotNil(t, CustomConfig)
		assert.Equal(t, CustomConfig.Name, "test")
		assert.Equal(t, CustomConfig.Version, "v2.0")
		assert.Equal(t, CustomConfig.DefineConfig, map[string]interface{}{"test-config": true})
		assert.Equal(t, CustomConfig.GetDefineValue("test-config", false), true)
		assert.Equal(t, CustomConfig.GetDefineValue("test-no-config", false), false)
		assert.Equal(t, GetDefineValue("test-config", false), true)
		assert.Equal(t, GetDefineValue("test-no-config", false), false)
	})

	t.Run("config builder", func(t *testing.T) {
		CustomConfigBuilder := NewCustomConfigBuilder()
		CustomConfigBuilder.SetName("builder")
		CustomConfigBuilder.SetVersion("v3.0")
		CustomConfigBuilder.SetDefineConfig("test-build", true)
		CustomConfig := CustomConfigBuilder.Build()
		assert.NotNil(t, CustomConfig)
		assert.Equal(t, CustomConfig.Name, "builder")
		assert.Equal(t, CustomConfig.Version, "v3.0")
		assert.Equal(t, CustomConfig.GetDefineValue("test-build", false), true)
		assert.Equal(t, CustomConfig.GetDefineValue("test-no-build", false), false)
		rt := NewRootConfigBuilder().Build()
		assert.Equal(t, rt.Custom.Name, "")
		assert.Equal(t, rt.Custom.Version, "")
		rt = NewRootConfigBuilder().SetCustom(CustomConfig).Build()
		rt.Init()
		assert.Equal(t, rt.Custom.Name, "builder")
		assert.Equal(t, rt.Custom.Version, "v3.0")
	})
}

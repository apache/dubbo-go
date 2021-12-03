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

func TestUserDefineInit(t *testing.T) {
	t.Run("empty use default", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/user_define/empty_log.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		UserDefineConfig := rootConfig.UserDefine
		assert.NotNil(t, UserDefineConfig)
		assert.Equal(t, UserDefineConfig.Name, "user-config")
		assert.Equal(t, UserDefineConfig.Version, "v1.0")
		assert.Equal(t, UserDefineConfig.DefineConfig, map[string]interface{}(nil))
		assert.Equal(t, UserDefineConfig.GetDefineValue("test", "test"), "test")
	})

	t.Run("use config", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/user_define/user_define.yaml"))
		assert.Nil(t, err)
		assert.NotNil(t, rootConfig)
		UserDefineConfig := rootConfig.UserDefine
		assert.NotNil(t, UserDefineConfig)
		assert.Equal(t, UserDefineConfig.Name, "test")
		assert.Equal(t, UserDefineConfig.Version, "v2.0")
		assert.Equal(t, UserDefineConfig.DefineConfig, map[string]interface{}{"test-config": true})
		assert.Equal(t, UserDefineConfig.GetDefineValue("test-config", false), true)
		assert.Equal(t, UserDefineConfig.GetDefineValue("test-no-config", false), false)
	})
}

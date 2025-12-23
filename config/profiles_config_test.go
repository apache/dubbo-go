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
	"github.com/knadh/koanf"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestProfilesConfig_Prefix(t *testing.T) {
	profiles := &ProfilesConfig{}
	assert.Equal(t, constant.ProfilesConfigPrefix, profiles.Prefix())
}

func TestLoaderConf_MergeConfig(t *testing.T) {
	rc := NewRootConfigBuilder().Build()
	conf := NewLoaderConf(WithPath("./testdata/config/active/application.yaml"))
	koan := GetConfigResolver(conf)
	koan = conf.MergeConfig(koan)

	err := koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"})
	require.NoError(t, err)

	registries := rc.Registries
	assert.NotNil(t, registries)
	assert.Equal(t, "10s", registries["nacos"].Timeout)
	assert.Equal(t, "nacos://127.0.0.1:8848", registries["nacos"].Address)

	protocols := rc.Protocols
	assert.NotNil(t, protocols)
	assert.Equal(t, "dubbo", protocols["dubbo"].Name)
	assert.Equal(t, "20000", protocols["dubbo"].Port)

	consumer := rc.Consumer
	assert.NotNil(t, consumer)
	assert.Equal(t, "dubbo", consumer.References["helloService"].Protocol)
	assert.Equal(t, "org.github.dubbo.HelloService", consumer.References["helloService"].InterfaceName)
}

func Test_getLegalActive(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		active := getLegalActive("")
		assert.Equal(t, "default", active)
	})

	t.Run("normal", func(t *testing.T) {
		active := getLegalActive("active")
		assert.Equal(t, "active", active)
	})
}

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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestProfilesConfig_Prefix(t *testing.T) {
	profiles := &ProfilesConfig{}
	assert.Equal(t, profiles.Prefix(), constant.ProfilesConfigPrefix)
}

func TestLoaderConf_MergeConfig(t *testing.T) {
	rc := NewRootConfigBuilder().Build()
	conf := NewLoaderConf(WithPath("./testdata/config/active/application.yaml"))
	koan := GetConfigResolver(conf)
	koan = conf.MergeConfig(koan)

	err := koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"})
	assert.Nil(t, err)

	registries := rc.Registries
	assert.NotNil(t, registries)
	assert.Equal(t, registries["nacos"].Timeout, "10s")
	assert.Equal(t, registries["nacos"].Address, "nacos://127.0.0.1:8848")

	protocols := rc.Protocols
	assert.NotNil(t, protocols)
	assert.Equal(t, protocols["dubbo"].Name, "dubbo")
	assert.Equal(t, protocols["dubbo"].Port, "20000")

	consumer := rc.Consumer
	assert.NotNil(t, consumer)
	assert.Equal(t, consumer.References["helloService"].Protocol, "dubbo")
	assert.Equal(t, consumer.References["helloService"].InterfaceName, "org.github.dubbo.HelloService")
}

func Test_getLegalActive(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		active := getLegalActive("")
		assert.Equal(t, active, "default")
	})

	t.Run("normal", func(t *testing.T) {
		active := getLegalActive("active")
		assert.Equal(t, active, "active")
	})
}

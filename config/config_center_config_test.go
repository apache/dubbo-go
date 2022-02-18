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
	_ "dubbo.apache.org/dubbo-go/v3/config_center/apollo"
)

func TestApolloConfigCenterConfig(t *testing.T) {

	err := Load(WithPath("./testdata/config/center/apollo.yaml"))
	assert.Nil(t, err)

	registries := rootConfig.Registries
	assert.NotNil(t, registries)
}
func TestConfigCenterConfig(t *testing.T) {
	var (
		config *CenterConfig
		err    error
	)

	t.Run("ConfigCenterConfigBuilder", func(t *testing.T) {
		config = NewConfigCenterConfigBuilder().
			SetProtocol("protocol").
			SetUserName("userName").
			SetAddress("address").
			SetPassword("password").
			SetNamespace("namespace").
			SetGroup("group").
			SetDataID("dataId").
			Build()

		err = config.check()
		assert.Nil(t, err)

		assert.Equal(t, config.Protocol, "protocol")
		assert.Equal(t, config.Username, "userName")
		assert.Equal(t, config.Address, "address")
		assert.Equal(t, config.Password, "password")
		assert.Equal(t, config.Namespace, "namespace")
		assert.Equal(t, config.Group, "group")
		assert.Equal(t, config.DataId, "dataId")
		assert.Equal(t, config.Prefix(), constant.ConfigCenterPrefix)
		//assert.Equal(t, config.NameId(),
		//	strings.Join([]string{constant.ConfigCenterPrefix, "protocol", "address"}, "-"))
	})

	t.Run("GetUrlMap", func(t *testing.T) {
		url := config.GetUrlMap()
		assert.Equal(t, url.Get(constant.ConfigNamespaceKey), config.Namespace)
		assert.Equal(t, url.Get(constant.ConfigClusterKey), config.Cluster)
	})

	t.Run("translateConfigAddress", func(t *testing.T) {
		address := config.translateConfigAddress()
		assert.Equal(t, address, "address")
	})

	t.Run("toUrl", func(t *testing.T) {
		url, err := config.toURL()
		assert.Nil(t, err)
		namespace := url.GetParam(constant.ConfigNamespaceKey, "namespace")
		assert.Equal(t, namespace, "namespace")
	})
}

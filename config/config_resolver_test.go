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

func TestResolvePlaceHolder(t *testing.T) {
	t.Run("test resolver", func(t *testing.T) {
		conf := NewLoaderConf(WithPath("./testdata/config/resolver/application.yaml"))
		koan := GetConfigResolver(conf)
		assert.Equal(t, koan.Get("dubbo.config-center.address"), koan.Get("dubbo.registries.nacos.address"))
		assert.Equal(t, koan.Get("localhost"), koan.Get("dubbo.protocols.dubbo.ip"))
		assert.Equal(t, "", koan.Get("dubbo.registries.nacos.group"))
		assert.Equal(t, "dev", koan.Get("dubbo.registries.zk.group"))

		rc := NewRootConfigBuilder().Build()
		err := koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"})
		assert.Nil(t, err)
		assert.Equal(t, rc.ConfigCenter.Address, rc.Registries["nacos"].Address)
		//not exist, default
		assert.Equal(t, "", rc.Registries["nacos"].Group)
		assert.Equal(t, "dev", rc.Registries["zk"].Group)

	})
}

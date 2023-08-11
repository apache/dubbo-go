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

// todo: think about how to expose functions of RegistryConfig
//func TestNewRegistryConfigBuilder(t *testing.T) {
//
//	config := NewRegistryConfigBuilder().
//		SetProtocol("nacos").
//		SetTimeout("10s").
//		SetGroup("group").
//		SetNamespace("public").
//		SetTTL("10s").
//		SetAddress("127.0.0.1:8848").
//		SetUsername("nacos").
//		SetPassword("123456").
//		SetSimplified(true).
//		SetPreferred(true).
//		SetZone("zone").
//		SetWeight(100).
//		SetParams(map[string]string{"timeout": "3s"}).
//		AddParam("timeout", "15s").
//		SetRegistryType("local").
//		Build()
//
//	config.DynamicUpdateProperties(config)
//
//	assert.Equal(t, config.Prefix(), constant.RegistryConfigPrefix)
//
//	values := config.getUrlMap(common.PROVIDER)
//	assert.Equal(t, values.Get("timeout"), "15s")
//
//	url, err := config.toMetadataReportUrl()
//	assert.NoError(t, err)
//	assert.Equal(t, url.GetParam("timeout", "3s"), "10s")
//
//	url, err = config.toURL(common.PROVIDER)
//	assert.NoError(t, err)
//	assert.Equal(t, url.GetParam("timeout", "3s"), "15s")
//
//	address := config.translateRegistryAddress()
//	assert.Equal(t, address, "127.0.0.1:8848")
//}
func TestNewRegistryConfig(t *testing.T) {
	config := NewRegistryConfig(
		WithRegistryProtocol("nacos"),
		WithRegistryAddress("127.0.0.1:8848"),
		WithRegistryTimeOut("10s"),
		WithRegistryGroup("group"),
		WithRegistryTTL("10s"),
		WithRegistryUserName("nacos"),
		WithRegistryPassword("123456"),
		WithRegistrySimplified(true),
		WithRegistryPreferred(true),
		WithRegistryWeight(100),
		WithRegistryParams(map[string]string{"timeout": "3s"}))

	assert.Equal(t, config.Timeout, "10s")
}

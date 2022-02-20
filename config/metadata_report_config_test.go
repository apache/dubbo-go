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

func TestNewMetadataReportConfigBuilder(t *testing.T) {
	config := NewMetadataReportConfigBuilder().
		SetProtocol("nacos").
		SetAddress("127.0.0.1:8848").
		SetUsername("nacos").
		SetPassword("123456").
		SetTimeout("10s").
		SetGroup("dubbo").
		Build()

	assert.Equal(t, config.IsValid(), true)
	assert.Equal(t, config.Prefix(), constant.MetadataReportPrefix)

	url, err := config.ToUrl()
	assert.NoError(t, err)
	assert.Equal(t, url.GetParam(constant.TimeoutKey, "3s"), "10s")
}

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
	"os"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

func TestGetRestConsumerServiceConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_CONSUMER_FILE_PATH, "./rest/config_reader/reader_impl/testdata/consumer_config.yml")
	assert.NoError(t, err)
	err = ConsumerRestConfigInit()
	assert.NoError(t, err)
	serviceConfig := GetRestConsumerServiceConfig("UserProvider")
	assert.NotEmpty(t, serviceConfig)
	assert.NotEmpty(t, serviceConfig.RestMethodConfigsMap)
	assert.NotEmpty(t, serviceConfig.RestMethodConfigsMap["GetUser"])
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].QueryParamsMap[1], "userid")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].HeadersMap[3], "age")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].PathParamsMap[4], "time")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].Body, 0)
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].Produces, "application/xml")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].Consumes, "application/xml")
	assert.Equal(t, serviceConfig.Client, "resty1")
}

func TestGetRestProviderServiceConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_PROVIDER_FILE_PATH, "./rest/config_reader/reader_impl/testdata/provider_config.yml")
	assert.NoError(t, err)
	err = ProviderRestConfigInit()
	assert.NoError(t, err)
	serviceConfig := GetRestProviderServiceConfig("UserProvider")
	assert.NotEmpty(t, serviceConfig)
	assert.NotEmpty(t, serviceConfig.RestMethodConfigsMap)
	assert.NotEmpty(t, serviceConfig.RestMethodConfigsMap["GetUser"])
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].QueryParamsMap[1], "userid")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].HeadersMap[3], "age")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].PathParamsMap[4], "time")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].Body, 0)
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].Produces, "application/xml")
	assert.Equal(t, serviceConfig.RestMethodConfigsMap["GetUser"].Consumes, "application/xml")
	assert.Equal(t, serviceConfig.Server, "go-restful1")

}

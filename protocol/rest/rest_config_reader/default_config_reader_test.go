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

package rest_config_reader

import (
	"os"
	"testing"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigReader_ReadConsumerConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_CONSUMER_FILE_PATH, "./testdata/consumer_config.yml")
	assert.NoError(t, err)
	reader := GetDefaultConfigReader()
	config := reader.ReadConsumerConfig()
	assert.NotEmpty(t, config)
}

func TestDefaultConfigReader_ReadProviderConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_PROVIDER_FILE_PATH, "./testdata/provider_config.yml")
	assert.NoError(t, err)
	reader := GetDefaultConfigReader()
	config := reader.ReadProviderConfig()
	assert.NotEmpty(t, config)
}

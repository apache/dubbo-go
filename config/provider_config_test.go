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

//import (
//	"dubbo.apache.org/dubbo-go/v3/config"
//	"dubbo.apache.org/dubbo-go/v3/config/consumer"
//	"path/filepath"
//	"testing"
//)
//
//import (
//	"github.com/stretchr/testify/assert"
//)
//
//func TestConsumerInit(t *testing.T) {
//	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
//	assert.NoError(t, err)
//	assert.NoError(t, consumer.ConsumerInit(conPath))
//	assert.Equal(t, "default", config.consumerConfig.ProxyFactory)
//	assert.Equal(t, "dubbo.properties", config.consumerConfig.ConfigCenterConfig.ConfigFile)
//	assert.Equal(t, "100ms", config.consumerConfig.Connect_Timeout)
//}
//
//func TestConsumerInitWithDefaultProtocol(t *testing.T) {
//	conPath, err := filepath.Abs("./testdata/consumer_config_withoutProtocol.yml")
//	assert.NoError(t, err)
//	assert.NoError(t, consumer.ConsumerInit(conPath))
//	assert.Equal(t, "dubbo", config.consumerConfig.References["UserProvider"].Protocol)
//}
//
//func TestProviderInitWithDefaultProtocol(t *testing.T) {
//	conPath, err := filepath.Abs("./testdata/provider_config_withoutProtocol.yml")
//	assert.NoError(t, err)
//	assert.NoError(t, ProviderInit(conPath))
//	assert.Equal(t, "dubbo", config.providerConfig.Services["UserProvider"].Protocol)
//}

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

package consul

import (
	"github.com/stretchr/testify/assert"
)

func (suite *consulRegistryTestSuite) testNewProviderRegistry() {
	providerRegistryUrl := newProviderRegistryUrl(registryHost, registryPort)
	providerRegistry, err := newConsulRegistry(providerRegistryUrl)
	assert.NoError(suite.t, err)
	suite.providerRegistry = providerRegistry
}

func (suite *consulRegistryTestSuite) testNewConsumerRegistry() {
	consumerRegistryUrl := newConsumerRegistryUrl(registryHost, registryPort)
	consumerRegistry, err := newConsulRegistry(consumerRegistryUrl)
	assert.NoError(suite.t, err)
	suite.consumerRegistry = consumerRegistry.(*consulRegistry)
}

func (suite *consulRegistryTestSuite) testRegister() {
	providerUrl := newProviderUrl(providerHost, providerPort, service, protocol)
	suite.providerUrl = providerUrl
	err := suite.providerRegistry.Register(providerUrl)
	assert.NoError(suite.t, err)
}

func (suite *consulRegistryTestSuite) testUnregister() {
	consulProviderRegistry, _ := suite.providerRegistry.(*consulRegistry)
	err := consulProviderRegistry.Unregister(suite.providerUrl)
	assert.NoError(suite.t, err)
}

func (suite *consulRegistryTestSuite) testSubscribe() {
	consumerUrl := newConsumerUrl(consumerHost, consumerPort, service, protocol)
	suite.consumerUrl = consumerUrl
	listener, err := suite.consumerRegistry.subscribe(&consumerUrl)
	assert.NoError(suite.t, err)
	suite.listener = listener
}

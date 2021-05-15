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

package configurator

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

// NewMockConfigurator creates a new mockConfigurator
func NewMockConfigurator(url *common.URL) config_center.Configurator {
	return &mockConfigurator{configuratorUrl: url}
}

type mockConfigurator struct {
	configuratorUrl *common.URL
}

// GetUrl gets a configuratorUrl
func (c *mockConfigurator) GetUrl() *common.URL {
	return c.configuratorUrl
}

// Configure sets up param CLUSTER_KEY and cluster for url
func (c *mockConfigurator) Configure(url *common.URL) {
	if cluster := c.GetUrl().GetParam(constant.CLUSTER_KEY, ""); cluster != "" {
		url.SetParam(constant.CLUSTER_KEY, cluster)
	}
}

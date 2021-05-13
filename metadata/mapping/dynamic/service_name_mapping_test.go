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

package dynamic

import (
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

func TestDynamicConfigurationServiceNameMapping(t *testing.T) {
	// mock data
	appName := "myApp"
	dc, err := (&config_center.MockDynamicConfigurationFactory{
		Content: appName,
	}).GetDynamicConfiguration(nil)
	assert.NoError(t, err)
	config.GetApplicationConfig().Name = appName

	mapping := &DynamicConfigurationServiceNameMapping{dc: dc}
	intf := constant.METADATA_SERVICE_NAME
	group := "myGroup"
	version := "myVersion"
	protocol := "myProtocol"

	err = mapping.Map(intf, group, version, protocol)
	assert.Nil(t, err)
	intf = "MyService"
	err = mapping.Map(intf, group, version, protocol)
	assert.Nil(t, err)

	var result *gxset.HashSet
	result, err = mapping.Get(intf, group, version, protocol)
	assert.Nil(t, err)
	assert.Equal(t, 1, result.Size())
	assert.True(t, result.Contains(appName))
}

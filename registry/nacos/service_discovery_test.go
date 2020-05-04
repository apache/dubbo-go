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

package nacos

import (
	"testing"
	"time"

	"github.com/apache/dubbo-go/config"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/observer"
	"github.com/apache/dubbo-go/common/observer/dispatcher"
	"github.com/apache/dubbo-go/registry"
)

var (
	testName = "test"
)

func TestNacosServiceDiscovery_Destroy(t *testing.T) {
	prepareData()
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.NACOS_KEY, testName)
	assert.Nil(t, err)
	assert.NotNil(t, serviceDiscovery)
	err = serviceDiscovery.Destroy()
	assert.Nil(t, err)
	assert.Nil(t, serviceDiscovery.(*nacosServiceDiscovery).namingClient)
}

func TestNacosServiceDiscovery_CRUD(t *testing.T) {
	prepareData()
	extension.SetEventDispatcher("mock", func() observer.EventDispatcher {
		return &dispatcher.MockEventDispatcher{}
	})

	extension.SetAndInitGlobalDispatcher("mock")

	serviceName := "service-name"
	id := "id"
	host := "host"
	port := 123
	instance := &registry.DefaultServiceInstance{
		Id:          id,
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	}

	// clean data

	serviceDiscovry, _ := extension.GetServiceDiscovery(constant.NACOS_KEY, testName)

	// clean data for local test
	serviceDiscovry.Unregister(&registry.DefaultServiceInstance{
		Id:          id,
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
	})

	err := serviceDiscovry.Register(instance)
	assert.Nil(t, err)

	page := serviceDiscovry.GetHealthyInstancesByPage(serviceName, 0, 10, true)
	assert.NotNil(t, page)

	assert.Equal(t, 0, page.GetOffset())
	assert.Equal(t, 10, page.GetPageSize())
	assert.Equal(t, 1, page.GetDataSize())

	instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	assert.NotNil(t, instance)
	assert.Equal(t, id, instance.GetId())
	assert.Equal(t, host, instance.GetHost())
	assert.Equal(t, port, instance.GetPort())
	assert.Equal(t, serviceName, instance.GetServiceName())
	assert.Equal(t, 0, len(instance.GetMetadata()))

	instance.Metadata["a"] = "b"

	err = serviceDiscovry.Update(instance)
	assert.Nil(t, err)

	pageMap := serviceDiscovry.GetRequestInstances([]string{serviceName}, 0, 1)
	assert.Equal(t, 1, len(pageMap))
	page = pageMap[serviceName]
	assert.NotNil(t, page)
	assert.Equal(t, 1, len(page.GetData()))

	instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	v, _ := instance.Metadata["a"]
	assert.Equal(t, "b", v)

	// test dispatcher event
	err = serviceDiscovry.DispatchEventByServiceName(serviceName)
	assert.Nil(t, err)

	// test AddListener
	err = serviceDiscovry.AddListener(&registry.ServiceInstancesChangedListener{})
	assert.Nil(t, err)
}

func TestNacosServiceDiscovery_GetDefaultPageSize(t *testing.T) {
	prepareData()
	serviceDiscovry, _ := extension.GetServiceDiscovery(constant.NACOS_KEY, testName)
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovry.GetDefaultPageSize())
}

func prepareData() {
	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol:  "nacos",
		RemoteRef: testName,
	}

	config.GetBaseConfig().Remotes[testName] = &config.RemoteConfig{
		Address: "console.nacos.io:80",
		Timeout: 10 * time.Second,
	}
}

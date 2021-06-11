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
	"math/rand"
	"strconv"
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
	"dubbo.apache.org/dubbo-go/v3/common/observer/dispatcher"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/event"
)

var testName = "test"

func Test_newNacosServiceDiscovery(t *testing.T) {
	name := "nacos1"
	_, err := newNacosServiceDiscovery(name)

	// the ServiceDiscoveryConfig not found
	assert.NotNil(t, err)

	sdc := &config.ServiceDiscoveryConfig{
		Protocol:  "nacos",
		RemoteRef: "mock",
	}
	config.GetBaseConfig().ServiceDiscoveries[name] = sdc

	_, err = newNacosServiceDiscovery(name)

	// RemoteConfig not found
	assert.NotNil(t, err)

	config.GetBaseConfig().Remotes["mock"] = &config.RemoteConfig{
		Address:    "console.nacos.io:80",
		TimeoutStr: "10s",
	}

	res, err := newNacosServiceDiscovery(name)
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

func TestNacosServiceDiscovery_CRUD(t *testing.T) {
	if !checkNacosServerAlive() {
		return
	}
	prepareData()
	extension.SetEventDispatcher("mock", func() observer.EventDispatcher {
		return dispatcher.NewMockEventDispatcher()
	})

	extension.SetGlobalServiceNameMapping(func() mapping.ServiceNameMapping {
		return mapping.NewMockServiceNameMapping()
	})

	extension.SetAndInitGlobalDispatcher("mock")
	rand.Seed(time.Now().Unix())
	serviceName := "service-name" + strconv.Itoa(rand.Intn(10000))
	id := "id"
	host := "host"
	port := 123
	instance := &registry.DefaultServiceInstance{
		ID:          id,
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	}

	// clean data
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.NACOS_KEY, testName)
	assert.Nil(t, err)

	// clean data for local test
	err = serviceDiscovery.Unregister(&registry.DefaultServiceInstance{
		ID:          id,
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
	})
	assert.Nil(t, err)

	err = serviceDiscovery.Register(instance)

	assert.Nil(t, err)

	// sometimes nacos may be failed to push update of instance,
	// so it need 10s to pull, we sleep 10 second to make sure instance has been update
	time.Sleep(5 * time.Second)
	page := serviceDiscovery.GetHealthyInstancesByPage(serviceName, 0, 10, true)
	assert.NotNil(t, page)
	assert.Equal(t, 0, page.GetOffset())
	assert.Equal(t, 10, page.GetPageSize())
	assert.Equal(t, 1, page.GetDataSize())

	instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	instance.ServiceName = serviceName
	assert.NotNil(t, instance)
	assert.Equal(t, id, instance.GetID())
	assert.Equal(t, host, instance.GetHost())
	assert.Equal(t, port, instance.GetPort())
	// TODO: console.nacos.io has updated to nacos 2.0 and serviceName has changed in 2.0, so ignore temporarily.
	// assert.Equal(t, serviceName, instance.GetServiceName())
	assert.Equal(t, 0, len(instance.GetMetadata()))

	instance.Metadata["a"] = "b"
	err = serviceDiscovery.Update(instance)
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)
	pageMap := serviceDiscovery.GetRequestInstances([]string{serviceName}, 0, 1)
	assert.Equal(t, 1, len(pageMap))

	page = pageMap[serviceName]
	assert.NotNil(t, page)
	assert.Equal(t, 1, len(page.GetData()))

	instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	v, ok := instance.Metadata["a"]
	assert.True(t, ok)
	assert.Equal(t, "b", v)

	// test dispatcher event
	err = serviceDiscovery.DispatchEventByServiceName(serviceName)
	assert.Nil(t, err)
	hs := gxset.NewSet()
	hs.Add(serviceName)
	// test AddListener
	err = serviceDiscovery.AddListener(event.NewServiceInstancesChangedListener(hs))
	assert.Nil(t, err)
}

func TestNacosServiceDiscovery_GetDefaultPageSize(t *testing.T) {
	prepareData()
	serviceDiscovery, _ := extension.GetServiceDiscovery(constant.NACOS_KEY, testName)
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
}

func TestNacosServiceDiscovery_Destroy(t *testing.T) {
	prepareData()
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.NACOS_KEY, testName)
	assert.Nil(t, err)
	assert.NotNil(t, serviceDiscovery)
	err = serviceDiscovery.Destroy()
	assert.Nil(t, err)
}

func prepareData() {
	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol:  "nacos",
		RemoteRef: testName,
	}

	config.GetBaseConfig().Remotes[testName] = &config.RemoteConfig{
		Address:    "console.nacos.io:80",
		TimeoutStr: "10s",
	}
}

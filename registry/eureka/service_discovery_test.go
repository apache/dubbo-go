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

package eureka

import (
	"github.com/apache/dubbo-go/registry"
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
)

func TestNewEurekaServiceDiscovery(t *testing.T) {
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.EUREKA_KEY, mockUrl())
	assert.Nil(t, err)
	assert.NotNil(t, serviceDiscovery.(*eurekaServiceDiscovery).client)
	err = serviceDiscovery.Destroy()
	assert.Nil(t, err)
	assert.Nil(t, serviceDiscovery.(*eurekaServiceDiscovery).client)
}

func TestEurekaServiceDiscovery_CRUD(t *testing.T) {
	serviceName := "DEFAULT_GROUP"
	id := "id"
	host := "localhost"
	port := 6001
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

	serviceDiscovery, _ := extension.GetServiceDiscovery(constant.EUREKA_KEY, mockUrl())
	defer func() {
		// clean data for local test
		serviceDiscovery.Unregister(&registry.DefaultServiceInstance{
			Id:          instance.Id,
			ServiceName: instance.ServiceName,
			Host:        instance.Host,
			Port:        instance.Port,
		})
	}()

	err := serviceDiscovery.Register(instance)
	assert.Nil(t, err)

	page := serviceDiscovery.GetHealthyInstancesByPage(serviceName, 0, 10, true)
	assert.NotNil(t, page)

	assert.Equal(t, 0, page.GetOffset())
	assert.Equal(t, 10, page.GetPageSize())
	assert.Equal(t, 1, page.GetDataSize())
	assert.Equal(t, 1, len(page.GetData()))

	instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	assert.NotNil(t, instance)
	assert.Equal(t, id, instance.GetId())
	assert.Equal(t, host, instance.GetHost())
	assert.Equal(t, port, instance.GetPort())
	assert.Equal(t, serviceName, instance.GetServiceName())
	assert.Equal(t, 0, len(instance.GetMetadata()))

	pageMap := serviceDiscovery.GetRequestInstances([]string{serviceName}, 0, 1)
	assert.Equal(t, 1, len(pageMap))
	page = pageMap[serviceName]
	assert.NotNil(t, page)
	assert.Equal(t, 1, len(page.GetData()))
	//
	//instance.Metadata["a"] = "b"
	//
	//err = serviceDiscovery.Update(instance)
	//assert.Nil(t, err)
	//
	//pageMap := serviceDiscovery.GetRequestInstances([]string{serviceName}, 0, 1)
	//assert.Equal(t, 1, len(pageMap))
	//page = pageMap[serviceName]
	//assert.NotNil(t, page)
	//assert.Equal(t, 1, len(page.GetData()))
	//
	//instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	//v, _ := instance.Metadata["a"]
	//assert.Equal(t, "b", v)

	// test dispatcher event
	//err = serviceDiscovery.DispatchEventByServiceName(serviceName)
	//assert.Nil(t, err)

	// test AddListener
	//err = serviceDiscovery.AddListener(&registry.ServiceInstancesChangedListener{})
	//assert.Nil(t, err)
}

func mockUrl() *common.URL {
	regurl, _ := common.NewURL("registry://106.54.227.205:8080", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	return &regurl
}

func TestEurekaServiceDiscovery_GetDefaultPageSize(t *testing.T) {
	serviceDiscovery, _ := extension.GetServiceDiscovery(constant.EUREKA_KEY, mockUrl())
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
}

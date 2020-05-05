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

package filesystem

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/registry"
)

func TestFilleSystemServiceDiscovery(t *testing.T) {
	discovery, _ := extension.GetServiceDiscovery(name, nil)
	serviceName := "my-service"
	err := discovery.Register(&registry.DefaultServiceInstance{
		ServiceName: serviceName,
		Id:          "1",
		Healthy:     true,
	})
	assert.Nil(t, err)

	err = discovery.Register(&registry.DefaultServiceInstance{
		Id:          "2",
		ServiceName: "mock-service",
		Healthy:     false,
	})

	assert.Nil(t, err)

	services := discovery.GetServices()
	assert.Equal(t, 2, services.Size())
	assert.Equal(t, registry.DefaultPageSize, discovery.GetDefaultPageSize())

	reqInstances := discovery.GetRequestInstances([]string{serviceName, "mock-service"}, 0, 10)
	assert.Equal(t, 2, len(reqInstances))

	page := discovery.GetInstancesByPage(serviceName, 0, 10)
	assert.Equal(t, 1, page.GetDataSize())

	discovery.GetHealthyInstancesByPage(serviceName, 0, 10, true)
	page = discovery.GetInstancesByPage(serviceName, 0, 10)
	assert.Equal(t, 1, page.GetDataSize())

	err = discovery.AddListener(&registry.ServiceInstancesChangedListener{})
	assert.Nil(t, err)

	err = discovery.DispatchEvent(&registry.ServiceInstancesChangedEvent{})
	assert.Nil(t, err)

	err = discovery.DispatchEventForInstances(serviceName, nil)
	assert.Nil(t, err)

	err = discovery.DispatchEventByServiceName(serviceName)
	assert.Nil(t, err)

	err = discovery.Unregister(&registry.DefaultServiceInstance{
		Id: "2",
	})
	assert.Nil(t, err)

	services = discovery.GetServices()
	assert.Equal(t, 1, services.Size())

	err = discovery.Update(&registry.DefaultServiceInstance{
		Id: "3",
	})
	assert.Nil(t, err)

	services = discovery.GetServices()
	assert.Equal(t, 2, services.Size())

	err = discovery.Destroy()
	assert.Nil(t, err)

	services = discovery.GetServices()
	assert.Equal(t, 0, services.Size())
}

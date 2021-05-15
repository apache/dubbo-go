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
	"fmt"
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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
	"dubbo.apache.org/dubbo-go/v3/common/observer/dispatcher"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/event"
	"dubbo.apache.org/dubbo-go/v3/remoting/consul"
)

var (
	testName                             = "test"
	consulCheckPassInterval              = 17000
	consulDeregisterCriticalServiceAfter = "20s"
	consulWatchTimeout                   = 60000
)

func TestConsulServiceDiscovery_newConsulServiceDiscovery(t *testing.T) {
	name := "consul1"
	_, err := newConsulServiceDiscovery(name)
	assert.NotNil(t, err)

	sdc := &config.ServiceDiscoveryConfig{
		Protocol:  "consul",
		RemoteRef: "mock",
	}

	config.GetBaseConfig().ServiceDiscoveries[name] = sdc

	_, err = newConsulServiceDiscovery(name)
	assert.NotNil(t, err)

	config.GetBaseConfig().Remotes["mock"] = &config.RemoteConfig{
		Address: "localhost:8081",
	}

	res, err := newConsulServiceDiscovery(name)
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

func TestConsulServiceDiscovery_Destroy(t *testing.T) {
	prepareData()
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.CONSUL_KEY, testName)
	prepareService()
	assert.Nil(t, err)
	assert.NotNil(t, serviceDiscovery)
	err = serviceDiscovery.Destroy()
	assert.Nil(t, err)
	assert.Nil(t, serviceDiscovery.(*consulServiceDiscovery).consulClient)
}

func TestConsulServiceDiscovery_CRUD(t *testing.T) {
	// start consul agent
	consulAgent := consul.NewConsulAgent(t, registryPort)
	defer func() {
		err := consulAgent.Shutdown()
		assert.NoError(t, err)
	}()

	prepareData()

	eventDispatcher := dispatcher.NewMockEventDispatcher()
	extension.SetEventDispatcher("mock", func() observer.EventDispatcher {
		return eventDispatcher
	})

	extension.SetGlobalServiceNameMapping(func() mapping.ServiceNameMapping {
		return mapping.NewMockServiceNameMapping()
	})

	extension.SetAndInitGlobalDispatcher("mock")
	rand.Seed(time.Now().Unix())

	instance, _ := prepareService()

	// clean data
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.CONSUL_KEY, testName)
	assert.Nil(t, err)

	err = serviceDiscovery.Unregister(instance)
	assert.Nil(t, err)

	err = serviceDiscovery.Register(instance)
	assert.Nil(t, err)

	// sometimes nacos may be failed to push update of instance,
	// so it need 10s to pull, we sleep 10 second to make sure instance has been update
	time.Sleep(3 * time.Second)
	page := serviceDiscovery.GetHealthyInstancesByPage(instance.GetServiceName(), 0, 10, true)
	assert.NotNil(t, page)
	assert.Equal(t, 0, page.GetOffset())
	assert.Equal(t, 10, page.GetPageSize())
	assert.Equal(t, 1, page.GetDataSize())

	instanceResult := page.GetData()[0].(*registry.DefaultServiceInstance)
	assert.NotNil(t, instanceResult)
	assert.Equal(t, buildID(instance), instanceResult.GetID())
	assert.Equal(t, instance.GetHost(), instanceResult.GetHost())
	assert.Equal(t, instance.GetPort(), instanceResult.GetPort())
	assert.Equal(t, instance.GetServiceName(), instanceResult.GetServiceName())
	metadata := instanceResult.GetMetadata()
	assert.Equal(t, 0, len(metadata))

	instance.GetMetadata()["aaa"] = "bbb"
	err = serviceDiscovery.Update(instance)
	assert.Nil(t, err)

	time.Sleep(3 * time.Second)
	pageMap := serviceDiscovery.GetRequestInstances([]string{instance.GetServiceName()}, 0, 1)
	assert.Equal(t, 1, len(pageMap))

	page = pageMap[instance.GetServiceName()]
	assert.NotNil(t, page)
	assert.Equal(t, 1, len(page.GetData()))

	instanceResult = page.GetData()[0].(*registry.DefaultServiceInstance)
	v, ok := instanceResult.Metadata["aaa"]
	assert.True(t, ok)
	assert.Equal(t, "bbb", v)

	// test dispatcher event
	// err = serviceDiscovery.DispatchEventByServiceName(instanceResult.GetServiceName())
	// assert.Nil(t, err)

	// test AddListener
	hs := gxset.NewSet()
	hs.Add(instance.GetServiceName())
	err = serviceDiscovery.AddListener(event.NewServiceInstancesChangedListener(hs))
	assert.Nil(t, err)
	err = serviceDiscovery.Unregister(instance)
	assert.Nil(t, err)
	timer := time.NewTimer(time.Second * 10)
	select {
	case <-eventDispatcher.Notify:
		assert.NotNil(t, eventDispatcher.Event)
		break
	case <-timer.C:
		assert.Fail(t, "")
		break
	}
}

func prepareData() {
	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol:  "consul",
		RemoteRef: testName,
	}

	config.GetBaseConfig().Remotes[testName] = &config.RemoteConfig{
		Address: fmt.Sprintf("%s:%d", registryHost, registryPort),
	}
}

func prepareService() (registry.ServiceInstance, *common.URL) {
	id := "id"

	registryUrl, _ := common.NewURL(protocol + "://" + providerHost + ":" + strconv.Itoa(providerPort) + "/" + service + "?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&consul-check-pass-interval=" + strconv.Itoa(consulCheckPassInterval) + "&consul-deregister-critical-service-after=" + consulDeregisterCriticalServiceAfter + "&" +
		"consul-watch-timeout=" + strconv.Itoa(consulWatchTimeout))

	return &registry.DefaultServiceInstance{
		ID:          id,
		ServiceName: service,
		Host:        registryHost,
		Port:        registryPort,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	}, registryUrl
}

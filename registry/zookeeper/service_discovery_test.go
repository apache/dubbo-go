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

package zookeeper

import (
	"strconv"
	"sync"
	"testing"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/observer"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
)

var testName = "test"

var tc *zk.TestCluster

func prepareData(t *testing.T) *zk.TestCluster {
	var err error
	tc, err = zk.StartTestCluster(1, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, tc.Servers[0])
	address := "127.0.0.1:" + strconv.Itoa(tc.Servers[0].Port)

	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol:  "zookeeper",
		RemoteRef: "test",
	}

	config.GetBaseConfig().Remotes[testName] = &config.RemoteConfig{
		Address:    address,
		TimeoutStr: "10s",
	}
	return tc
}

func TestNewZookeeperServiceDiscovery(t *testing.T) {
	name := "zookeeper1"
	_, err := newZookeeperServiceDiscovery(name)

	// the ServiceDiscoveryConfig not found
	assert.NotNil(t, err)

	sdc := &config.ServiceDiscoveryConfig{
		Protocol:  "zookeeper",
		RemoteRef: "mock",
	}
	config.GetBaseConfig().ServiceDiscoveries[name] = sdc
	_, err = newZookeeperServiceDiscovery(name)

	// RemoteConfig not found
	assert.NotNil(t, err)
}

func TestCURDZookeeperServiceDiscovery(t *testing.T) {
	prepareData(t)
	sd, err := newZookeeperServiceDiscovery(testName)
	assert.Nil(t, err)
	defer func() {
		_ = sd.Destroy()
	}()
	md := make(map[string]string)
	md["t1"] = "test1"
	err = sd.Register(&registry.DefaultServiceInstance{
		Id:          "testId",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    md,
	})
	assert.Nil(t, err)

	testsPager := sd.GetHealthyInstancesByPage(testName, 0, 1, true)
	assert.Equal(t, 1, testsPager.GetDataSize())
	assert.Equal(t, 1, testsPager.GetTotalPages())
	test := testsPager.GetData()[0].(registry.ServiceInstance)
	assert.Equal(t, "127.0.0.1:2233", test.GetId())
	assert.Equal(t, "test1", test.GetMetadata()["t1"])

	md["t1"] = "test12"
	err = sd.Update(&registry.DefaultServiceInstance{
		Id:          "testId",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    md,
	})
	assert.Nil(t, err)

	testsPager = sd.GetInstancesByPage(testName, 0, 1)
	assert.Equal(t, 1, testsPager.GetDataSize())
	test = testsPager.GetData()[0].(registry.ServiceInstance)
	assert.Equal(t, "test12", test.GetMetadata()["t1"])

	testsMap := sd.GetRequestInstances([]string{testName}, 0, 1)
	assert.Equal(t, 1, len(testsMap))
	assert.Equal(t, 1, testsMap[testName].GetDataSize())
	test = testsMap[testName].GetData()[0].(registry.ServiceInstance)
	assert.Equal(t, "test12", test.GetMetadata()["t1"])

	names := sd.GetServices()
	assert.Equal(t, 1, names.Size())
	assert.Equal(t, testName, names.Values()[0])

	err = sd.Unregister(&registry.DefaultServiceInstance{
		Id:          "testId",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	})
	assert.Nil(t, err)
}

func TestAddListenerZookeeperServiceDiscovery(t *testing.T) {
	defer func() {
		_ = tc.Stop()
	}()
	sd, err := newZookeeperServiceDiscovery(testName)
	assert.Nil(t, err)
	defer func() {
		_ = sd.Destroy()
	}()

	err = sd.Register(&registry.DefaultServiceInstance{
		Id:          "testId",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	})
	assert.Nil(t, err)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	tn := &testNotify{
		wg: wg,
		t:  t,
	}
	sicl := &registry.ServiceInstancesChangedListener{
		ServiceName:   testName,
		ChangedNotify: tn,
	}
	extension.SetAndInitGlobalDispatcher("direct")
	extension.GetGlobalDispatcher().AddEventListener(sicl)
	err = sd.AddListener(sicl)
	assert.NoError(t, err)

	err = sd.Update(&registry.DefaultServiceInstance{
		Id:          "testId",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	})
	assert.NoError(t, err)
	tn.wg.Wait()
}

type testNotify struct {
	wg *sync.WaitGroup
	t  *testing.T
}

func (tn *testNotify) Notify(e observer.Event) {
	ice := e.(*registry.ServiceInstancesChangedEvent)
	assert.Equal(tn.t, 1, len(ice.Instances))
	assert.Equal(tn.t, "127.0.0.1:2233", ice.Instances[0].GetId())
	tn.wg.Done()
}

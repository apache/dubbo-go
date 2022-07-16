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

//
//import (
//	"context"
//	"strconv"
//	"sync"
//	"testing"
//)
//
//import (
//	"github.com/dubbogo/go-zookeeper/zk"
//
//	gxset "github.com/dubbogo/gost/container/set"
//
//	"github.com/stretchr/testify/assert"
//)
//
//import (
//	"dubbo.apache.org/dubbo-go/v3/common"
//	"dubbo.apache.org/dubbo-go/v3/common/constant"
//	"dubbo.apache.org/dubbo-go/v3/common/extension"
//	"github.com/dubbogo/gost/gof/observer"
//	"dubbo.apache.org/dubbo-go/v3/common/observer/dispatcher"
//	"dubbo.apache.org/dubbo-go/v3/config"
//	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
//	"dubbo.apache.org/dubbo-go/v3/protocol"
//	"dubbo.apache.org/dubbo-go/v3/registry"
//	"dubbo.apache.org/dubbo-go/v3/registry/event"
//)
//
//const testName = "test"
//
//func prepareData(t *testing.T) *zk.TestCluster {
//	var err error
//	tc, err := zk.StartTestCluster(1, nil, nil)
//	assert.NoError(t, err)
//	assert.NotNil(t, tc.Servers[0])
//	address := "127.0.0.1:" + strconv.Itoa(tc.Servers[0].Port)
//	//address := "127.0.0.1:2181"
//
//	config.GetRootConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
//		Protocol:  "zookeeper",
//		RemoteRef: "test",
//	}
//
//	config.GetRootConfig().Remotes[testName] = &config.RemoteConfig{
//		Address:    address,
//		TimeoutStr: "10s",
//	}
//	return tc
//}
//
//func TestNewZookeeperServiceDiscovery(t *testing.T) {
//	_, err := newZookeeperServiceDiscovery()
//
//	// the ShutdownConfig not found
//	// err: could not init the instance because the config is invalid
//	assert.NotNil(t, err)
//
//	//sdc := &config.ServiceDiscoveryConfig{
//	//	Protocol:  "zookeeper",
//	//	RemoteRef: "mock",
//	//}
//	//config.GetRootConfig().ServiceDiscoveries[name] = sdc
//	_, err = newZookeeperServiceDiscovery()
//
//	// RemoteConfig not found
//	// err: could not find the remote config for name: mock
//	assert.NotNil(t, err)
//}
//
//func TestZookeeperServiceDiscovery_CURDAndListener(t *testing.T) {
//	tc := prepareData(t)
//	defer func() {
//		_ = tc.Stop()
//	}()
//	t.Run("testCURDZookeeperServiceDiscovery", testCURDZookeeperServiceDiscovery)
//	t.Run("testAddListenerZookeeperServiceDiscovery", testAddListenerZookeeperServiceDiscovery)
//}
//
//func testCURDZookeeperServiceDiscovery(t *testing.T) {
//	prepareData(t)
//	extension.SetEventDispatcher("mock", func() observer.EventDispatcher {
//		return dispatcher.NewMockEventDispatcher()
//	})
//	extension.SetGlobalServiceNameMapping(func() mapping.ServiceNameMapping {
//		return mapping.NewMockServiceNameMapping()
//	})
//
//	extension.SetProtocol("mock", func() protocol.Protocol {
//		return &mockProtocol{}
//	})
//
//	sd, err := newZookeeperServiceDiscovery()
//	assert.Nil(t, err)
//	defer func() {
//		_ = sd.Destroy()
//	}()
//	ins := &registry.DefaultServiceInstance{
//		ID:          "testID",
//		ServiceName: testName,
//		Host:        "127.0.0.1",
//		Port:        2233,
//		Enable:      true,
//		Healthy:     true,
//		Metadata:    nil,
//	}
//	ins.Metadata = map[string]string{"t1": "test1", constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
//	err = sd.Register(ins)
//
//	assert.Nil(t, err)
//
//	testsPager := sd.GetHealthyInstancesByPage(testName, 0, 1, true)
//	assert.Equal(t, 1, testsPager.GetDataSize())
//	assert.Equal(t, 1, testsPager.GetTotalPages())
//	test := testsPager.GetData()[0].(registry.ServiceInstance)
//	assert.Equal(t, "127.0.0.1:2233", test.GetID())
//	assert.Equal(t, "test1", test.GetMetadata()["t1"])
//
//	ins = &registry.DefaultServiceInstance{
//		ID:          "testID",
//		ServiceName: testName,
//		Host:        "127.0.0.1",
//		Port:        2233,
//		Enable:      true,
//		Healthy:     true,
//	}
//	ins.Metadata = map[string]string{"t1": "test12", constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
//
//	err = sd.Update(ins)
//
//	assert.Nil(t, err)
//
//	testsPager = sd.GetInstancesByPage(testName, 0, 1)
//	assert.Equal(t, 1, testsPager.GetDataSize())
//	test = testsPager.GetData()[0].(registry.ServiceInstance)
//	assert.Equal(t, "test12", test.GetMetadata()["t1"])
//
//	testsMap := sd.GetRequestInstances([]string{testName}, 0, 1)
//	assert.Equal(t, 1, len(testsMap))
//	assert.Equal(t, 1, testsMap[testName].GetDataSize())
//	test = testsMap[testName].GetData()[0].(registry.ServiceInstance)
//	assert.Equal(t, "test12", test.GetMetadata()["t1"])
//
//	names := sd.GetServices()
//	assert.Equal(t, 1, names.Size())
//	assert.Equal(t, testName, names.Values()[0])
//
//	err = sd.Unregister(&registry.DefaultServiceInstance{
//		ID:          "testID",
//		ServiceName: testName,
//		Host:        "127.0.0.1",
//		Port:        2233,
//		Enable:      true,
//		Healthy:     true,
//		Metadata:    nil,
//	})
//	assert.Nil(t, err)
//}
//
//func testAddListenerZookeeperServiceDiscovery(t *testing.T) {
//	sd, err := newZookeeperServiceDiscovery()
//	assert.Nil(t, err)
//	defer func() {
//		_ = sd.Destroy()
//	}()
//
//	ins := &registry.DefaultServiceInstance{
//		ID:          "testID",
//		ServiceName: testName,
//		Host:        "127.0.0.1",
//		Port:        2233,
//		Enable:      true,
//		Healthy:     true,
//		Metadata:    nil,
//	}
//	ins.Metadata = map[string]string{"t1": "test12", constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
//	err = sd.Register(ins)
//
//	assert.Nil(t, err)
//	wg := &sync.WaitGroup{}
//	wg.Add(1)
//	tn := &testNotify{
//		wg: wg,
//		t:  t,
//	}
//	hs := gxset.NewSet()
//	hs.Add(testName)
//
//	sicl := event.NewServiceInstancesChangedListener(hs)
//	sicl.AddListenerAndNotify(testName, tn)
//	extension.SetAndInitGlobalDispatcher("direct")
//	extension.GetGlobalDispatcher().AddEventListener(sicl)
//	err = sd.AddListener(sicl)
//	assert.NoError(t, err)
//
//	ins = &registry.DefaultServiceInstance{
//		ID:          "testID",
//		ServiceName: testName,
//		Host:        "127.0.0.1",
//		Port:        2233,
//		Enable:      true,
//		Healthy:     true,
//		Metadata:    nil,
//	}
//	ins.Metadata = map[string]string{"t1": "test12", constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
//	err = sd.Update(ins)
//	assert.NoError(t, err)
//	tn.wg.Wait()
//}
//
//type testNotify struct {
//	wg *sync.WaitGroup
//	t  *testing.T
//}
//
//func (tn *testNotify) Notify(e *registry.ServiceEvent) {
//	assert.Equal(tn.t, "2233", e.Service.Port)
//	tn.wg.Done()
//}
//func (tn *testNotify) NotifyAll([]*registry.ServiceEvent, func()) {
//
//}
//
//type mockProtocol struct{}
//
//func (m mockProtocol) Export(protocol.Invoker) protocol.Exporter {
//	panic("implement me")
//}
//
//func (m mockProtocol) Refer(*common.URL) protocol.Invoker {
//	return &mockInvoker{}
//}
//
//func (m mockProtocol) Destroy() {
//	panic("implement me")
//}
//
//type mockInvoker struct{}
//
//func (m *mockInvoker) GetURL() *common.URL {
//	panic("implement me")
//}
//
//func (m *mockInvoker) IsAvailable() bool {
//	panic("implement me")
//}
//
//func (m *mockInvoker) Destroy() {
//	panic("implement me")
//}
//
//func (m *mockInvoker) Invoke(context.Context, protocol.Invocation) protocol.Result {
//	// for getMetadataInfo and ServiceInstancesChangedListenerImpl onEvent
//	serviceInfo := &common.ServiceInfo{ServiceKey: "test", MatchKey: "test"}
//	services := make(map[string]*common.ServiceInfo)
//	services["test"] = serviceInfo
//	return &protocol.RPCResult{
//		Rest: &common.MetadataInfo{
//			Services: services,
//		},
//	}
//}

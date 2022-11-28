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
	"context"
	"sync"
	"testing"
)

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func Test_newZookeeperServiceDiscovery(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:2181",
		common.WithParamsValue(constant.ClientNameKey, "zk-client"))
	sd, err := newZookeeperServiceDiscovery(url)
	assert.Nil(t, err)
	err = sd.Destroy()
	assert.Nil(t, err)

}
func Test_zookeeperServiceDiscovery_DataChange(t *testing.T) {
	serviceDiscovery := &zookeeperServiceDiscovery{}
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
}

type testNotify struct {
	wg *sync.WaitGroup
	t  *testing.T
}

func (tn *testNotify) Notify(e *registry.ServiceEvent) {
	assert.Equal(tn.t, "2181", e.Service.Port)
	tn.wg.Done()
}

func (tn *testNotify) NotifyAll([]*registry.ServiceEvent, func()) {}

type mockClient struct {
	instance []interface{}
}

func (c mockClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	return true, nil
}

func (c mockClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	return true, nil
}

func (c mockClient) UpdateInstance(param vo.UpdateInstanceParam) (bool, error) {
	return true, nil
}

func (c mockClient) GetService(param vo.GetServiceParam) (model.Service, error) {
	panic("implement me")
}

func (c mockClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	panic("implement me")
}

func (c mockClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	panic("implement me")
}

func (c mockClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	panic("implement me")
}

func (c mockClient) Subscribe(param *vo.SubscribeParam) error {
	return nil
}

func (c mockClient) Unsubscribe(param *vo.SubscribeParam) error {
	panic("implement me")
}

func (c mockClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) (model.ServiceList, error) {
	panic("implement me")
}

type mockProtocol struct{}

func (m mockProtocol) Export(protocol.Invoker) protocol.Exporter {
	panic("implement me")
}

func (m mockProtocol) Refer(*common.URL) protocol.Invoker {
	return &mockInvoker{}
}

func (m mockProtocol) Destroy() {
	panic("implement me")
}

type mockInvoker struct{}

func (m *mockInvoker) GetURL() *common.URL {
	panic("implement me")
}

func (m *mockInvoker) IsAvailable() bool {
	panic("implement me")
}

func (m *mockInvoker) Destroy() {
	panic("implement me")
}

func (m *mockInvoker) Invoke(context.Context, protocol.Invocation) protocol.Result {
	// for getMetadataInfo and ServiceInstancesChangedListenerImpl onEvent
	serviceInfo := &common.ServiceInfo{ServiceKey: "test", MatchKey: "test"}
	services := make(map[string]*common.ServiceInfo)
	services["test"] = serviceInfo
	return &protocol.RPCResult{
		Rest: &common.MetadataInfo{
			Services: services,
		},
	}
}

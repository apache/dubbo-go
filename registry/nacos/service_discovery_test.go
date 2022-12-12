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
	"context"
	"fmt"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"

	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/event"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

const testName = "test"

func TestNewNacosServiceDiscovery(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:8848",
		common.WithParamsValue(constant.ClientNameKey, "nacos-client"))
	sd, err := newNacosServiceDiscovery(url)
	assert.Nil(t, err)
	err = sd.Destroy()
	assert.Nil(t, err)
}

func TestNacosServiceDiscoveryGetDefaultPageSize(t *testing.T) {
	serviceDiscovery := &nacosServiceDiscovery{}
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
}

func TestFunction(t *testing.T) {

	extension.SetProtocol("mock", func() protocol.Protocol {
		return &mockProtocol{}
	})

	url, _ := common.NewURL("dubbo://127.0.0.1:8848")
	sd, _ := newMockNacosServiceDiscovery(url)
	defer func() {
		_ = sd.Destroy()
	}()

	ins := &registry.DefaultServiceInstance{
		ID:          "testID",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	}
	ins.Metadata = map[string]string{"t1": "test12", constant.MetadataServiceURLParamsPropertyName: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
	err := sd.Register(ins)
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	tn := &testNotify{
		wg: wg,
		t:  t,
	}
	hs := gxset.NewSet()
	hs.Add(testName)

	sicl := event.NewServiceInstancesChangedListener(hs)
	sicl.AddListenerAndNotify(testName, tn)
	err = sd.AddListener(sicl)
	assert.NoError(t, err)

	ins = &registry.DefaultServiceInstance{
		ID:          "testID",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	}
	ins.Metadata = map[string]string{"t1": "test12", constant.MetadataServiceURLParamsPropertyName: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
	err = sd.Update(ins)
	assert.NoError(t, err)
	err = sd.Unregister(ins)
	assert.Nil(t, err)
}

func newMockNacosServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	discoveryURL := common.NewURLWithOptions(
		common.WithParams(url.GetParams()),
		common.WithParamsValue(constant.TimeoutKey, url.GetParam(constant.RegistryTimeoutKey, constant.DefaultRegTimeout)),
		common.WithParamsValue(constant.NacosGroupKey, url.GetParam(constant.RegistryGroupKey, defaultGroup)),
		common.WithParamsValue(constant.NacosUsername, url.Username),
		common.WithParamsValue(constant.NacosPassword, url.Password),
		common.WithParamsValue(constant.ClientNameKey, "nacos-client"),
		common.WithParamsValue(constant.NacosNamespaceID, url.GetParam(constant.RegistryNamespaceKey, "")))
	discoveryURL.Location = url.Location
	discoveryURL.Username = url.Username
	discoveryURL.Password = url.Password
	client, err := nacos.NewNacosClientByURL(discoveryURL)
	mc := mockClient{}
	client.SetClient(mc)
	if err != nil {
		return nil, perrors.WithMessage(err, "create nacos namingClient failed.")
	}

	descriptor := fmt.Sprintf("nacos-service-discovery[%s]", discoveryURL.Location)

	group := url.GetParam(constant.RegistryGroupKey, defaultGroup)
	newInstance := &nacosServiceDiscovery{
		group:               group,
		namingClient:        client,
		descriptor:          descriptor,
		registryInstances:   []registry.ServiceInstance{},
		instanceListenerMap: make(map[string]*gxset.HashSet),
	}
	return newInstance, nil
}

type testNotify struct {
	wg *sync.WaitGroup
	t  *testing.T
}

func (tn *testNotify) Notify(e *registry.ServiceEvent) {
	assert.Equal(tn.t, "2233", e.Service.Port)
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

func (c mockClient) CloseClient() {
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

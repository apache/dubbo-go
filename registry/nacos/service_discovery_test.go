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
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/servicediscovery"
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

// Registration side: toRegisterInstance writes Weight into param
func TestWeight_RegisterParam(t *testing.T) {
	// Prepare a fake registryURL to avoid a nil pointer
	u, _ := common.NewURL("nacos://127.0.0.1:8848")
	sd := &nacosServiceDiscovery{registryURL: u}

	si := &registry.DefaultServiceInstance{
		ServiceName: "Demo",
		Host:        "127.0.0.1",
		Port:        20000,
		Weight:      77,
	}
	param := sd.toRegisterInstance(si)
	assert.Equal(t, 77.0, param.Weight)
}

//Subscription side: convertInstances writes Weight back
//to the instance and injects it into the URL

func fakeInstance(w float64) model.Instance {
	return model.Instance{
		Ip:     "192.181.10.1", //NOSONAR
		Port:   20000,
		Weight: w,
		Metadata: map[string]string{
			idKey: "inst-1",
		},
	}
}

func TestWeight_ConvertInstances(t *testing.T) {
	sd := &nacosServiceDiscovery{}
	out := sd.convertInstances([]model.Instance{fakeInstance(55)})

	assert.Len(t, out, 1)
	inst := out[0].(*registry.DefaultServiceInstance)
	assert.Equal(t, int64(55), inst.Weight)

	// Add valid endpoints data to avoid ERROR logs
	inst.Metadata[constant.ServiceInstanceEndpoints] = `[{"port":20000,"protocol":"tri"}]`

	// Verify that the weight is correctly injected into the URL
	urls := inst.ToURLs(&info.ServiceInfo{Protocol: "tri"})
	if len(urls) > 0 {
		got := urls[0].GetParam(constant.WeightKey, "")
		assert.Equal(t, "55", got, "Weight not injected into URL; load balancer cannot read it")
	}
}

func TestNacosServiceDiscoveryGetDefaultPageSize(t *testing.T) {
	serviceDiscovery := &nacosServiceDiscovery{}
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
}

func TestFunction(t *testing.T) {

	extension.SetProtocol("mock", func() base.Protocol {
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

	sicl := servicediscovery.NewServiceInstancesChangedListener("test_app", hs)
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

func TestBatchRegisterInstances(t *testing.T) {
	extension.SetProtocol("mock", func() base.Protocol {
		return &mockProtocol{}
	})
	var urls []*common.URL
	url1, _ := common.NewURL("dubbo://127.0.0.1:8848")
	url2, _ := common.NewURL("tri://127.0.0.1:8848")
	port := 20000
	urls = append(urls, url1)
	urls = append(urls, url2)
	for _, url := range urls {
		pcl := url.Protocol
		port = port + 1
		sd, _ := newMockNacosServiceDiscovery(url)
		defer func() {
			_ = sd.Destroy()
		}()
		ins := &registry.DefaultServiceInstance{
			ID:          "testID",
			ServiceName: "nacos_batchRegister_test1",
			Host:        url.Ip,
			Port:        port,
			Enable:      true,
			Healthy:     true,
			Metadata:    nil,
		}
		params := map[string]string{
			"protocol": "mock",
			"timeout":  "",
			"version":  "",
			pcl:        "",
			"release":  "",
			"port":     strconv.Itoa(port),
		}
		parmjosn, _ := json.Marshal(params)
		ins.Metadata = map[string]string{"t1": "test", constant.MetadataServiceURLParamsPropertyName: string(parmjosn)}
		err := sd.Register(ins)
		assert.Nil(t, err)
	}

	url3, _ := common.NewURL("tri://127.0.0.1:8848")
	sd, _ := newMockNacosServiceDiscovery(url3)
	defer func() {
		_ = sd.Destroy()
	}()
	ins := &registry.DefaultServiceInstance{
		ID:          "testID",
		ServiceName: "nacos_batchRegister_test2",
		Host:        "127.0.0.1",
		Port:        20004,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	}
	params := map[string]string{
		"protocol":    "mock",
		"timeout":     "",
		"version":     "",
		url3.Protocol: "",
		"release":     "",
		"port":        "20004",
	}
	parmjosn, _ := json.Marshal(params)
	ins.Metadata = map[string]string{"t1": "test", constant.MetadataServiceURLParamsPropertyName: string(parmjosn)}
	err := sd.Register(ins)
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
		registryURL:         url,
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
	instance []any
}

func (c mockClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	return true, nil
}

func (c mockClient) BatchRegisterInstance(param vo.BatchRegisterInstanceParam) (bool, error) {
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

func (m mockProtocol) Export(base.Invoker) base.Exporter {
	panic("implement me")
}

func (m mockProtocol) Refer(*common.URL) base.Invoker {
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

func (m *mockInvoker) Invoke(context.Context, base.Invocation) result.Result {
	return &result.RPCResult{
		Rest: &mockResult{},
	}
}

type mockResult struct {
}

// convertInstances converts nacos model.Instance to registry.ServiceInstance
// This method is used for testing weight conversion
func (n *nacosServiceDiscovery) convertInstances(instances []model.Instance) []registry.ServiceInstance {
	res := make([]registry.ServiceInstance, 0, len(instances))
	for _, ins := range instances {
		metadata := ins.Metadata
		id := metadata[idKey]
		delete(metadata, idKey)

		res = append(res, &registry.DefaultServiceInstance{
			ID:          id,
			ServiceName: "test-service", // service name for test
			Host:        ins.Ip,
			Port:        int(ins.Port),
			Weight:      int64(math.Round(ins.Weight)), // Ensure weight is correctly passed (for test)
			Enable:      ins.Enable,
			Healthy:     ins.Healthy,
			Metadata:    metadata,
			GroupName:   n.group,
		})
	}
	return res
}

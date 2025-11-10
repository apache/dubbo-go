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

package etcdv3

// import (
// 	"context"
// 	"testing"

// 	"dubbo.apache.org/dubbo-go/v3/common"
// 	"dubbo.apache.org/dubbo-go/v3/common/constant"
// 	"dubbo.apache.org/dubbo-go/v3/common/extension"
// 	"dubbo.apache.org/dubbo-go/v3/protocol/base"
// 	"dubbo.apache.org/dubbo-go/v3/protocol/result"
// 	"dubbo.apache.org/dubbo-go/v3/registry"
// 	"github.com/stretchr/testify/assert"
// )

// const testName = "test"

// func TestNewEtcdV3ServiceDiscovery(t *testing.T) {
// 	url, _ := common.NewURL("dubbo://127.0.0.1:2379", common.WithParamsValue(constant.ClientNameKey, "etcd-client"))
// 	sd, err := newEtcdV3ServiceDiscovery(url)
// 	assert.Nil(t, err)
// 	err = sd.Destroy()
// 	assert.Nil(t, err)
// }

// func TestEtcdV3ServiceDiscoveryGetDefaultPageSize(t *testing.T) {
// 	serviceDiscovery := &etcdV3ServiceDiscovery{}
// 	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
// }

// func TestFunction(t *testing.T) {

// 	extension.SetProtocol("mock", func() base.Protocol {
// 		return &mockProtocol{}
// 	})

// 	url, _ := common.NewURL("dubbo://127.0.0.1:2379")
// 	sd, _ := newEtcdV3ServiceDiscovery(url)
// 	defer func() {
// 		_ = sd.Destroy()
// 	}()

// 	ins := &registry.DefaultServiceInstance{
// 		ID:          "testID",
// 		ServiceName: testName,
// 		Host:        "127.0.0.1",
// 		Port:        2233,
// 		Enable:      true,
// 		Healthy:     true,
// 		Metadata:    nil,
// 	}
// 	ins.Metadata = map[string]string{"t1": "test12", constant.MetadataServiceURLParamsPropertyName: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
// 	err := sd.Register(ins)
// 	assert.Nil(t, err)

// 	ins = &registry.DefaultServiceInstance{
// 		ID:          "testID",
// 		ServiceName: testName,
// 		Host:        "127.0.0.1",
// 		Port:        2233,
// 		Enable:      true,
// 		Healthy:     true,
// 		Metadata:    nil,
// 	}
// 	ins.Metadata = map[string]string{"t1": "test12", constant.MetadataServiceURLParamsPropertyName: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2233"}`}
// 	err = sd.Update(ins)
// 	assert.NoError(t, err)
// 	err = sd.Unregister(ins)
// 	assert.NoError(t, err)
// }

// type mockProtocol struct{}

// func (m mockProtocol) Export(base.Invoker) base.Exporter {
// 	panic("implement me")
// }

// func (m mockProtocol) Refer(*common.URL) base.Invoker {
// 	return &mockInvoker{}
// }

// func (m mockProtocol) Destroy() {
// 	panic("implement me")
// }

// type mockInvoker struct{}

// func (m *mockInvoker) GetURL() *common.URL {
// 	panic("implement me")
// }

// func (m *mockInvoker) IsAvailable() bool {
// 	panic("implement me")
// }

// func (m *mockInvoker) Destroy() {
// 	panic("implement me")
// }

// func (m *mockInvoker) Invoke(context.Context, base.Invocation) result.Result {
// 	return &result.RPCResult{
// 		Rest: &mockResult{},
// 	}
// }

// type mockResult struct {
// }

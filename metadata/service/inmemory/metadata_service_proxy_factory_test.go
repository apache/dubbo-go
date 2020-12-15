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

package inmemory

import (
	"context"
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/registry"
)

func TestMetadataService_GetMetadataServiceUrlParams(t *testing.T) {
	str := `{"dubbo":{"timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}}`
	tmp := make(map[string]map[string]string)
	err := json.Unmarshal([]byte(str), &tmp)
	assert.Nil(t, err)
}

func TestCreateProxy(t *testing.T) {
	extension.SetProtocol("mock", func() protocol.Protocol {
		return &mockProtocol{}
	})
	ins := &registry.DefaultServiceInstance{
		Id:          "test-id",
		ServiceName: "com.dubbo",
		Host:        "localhost",
		Port:        8080,
		Enable:      true,
		Healthy:     true,
	}

	pxy := createProxy(ins)
	assert.Nil(t, pxy)

	ins.Metadata = map[string]string{constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"mock":{"timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}}`}
	pxy = createProxy(ins)
	assert.NotNil(t, pxy)
}

type mockProtocol struct {
}

func (m mockProtocol) Export(protocol.Invoker) protocol.Exporter {
	panic("implement me")
}

func (m mockProtocol) Refer(*common.URL) protocol.Invoker {
	return &mockInvoker{}
}

func (m mockProtocol) Destroy() {
	panic("implement me")
}

type mockInvoker struct {
}

func (m *mockInvoker) GetUrl() *common.URL {
	panic("implement me")
}

func (m *mockInvoker) IsAvailable() bool {
	panic("implement me")
}

func (m *mockInvoker) Destroy() {
	panic("implement me")
}

func (m *mockInvoker) Invoke(context.Context, protocol.Invocation) protocol.Result {
	return &protocol.RPCResult{
		Rest: &[]interface{}{"dubbo://localhost"},
	}
}

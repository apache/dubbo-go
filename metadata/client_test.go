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

package metadata

import (
	"context"
	"testing"
)

import (
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

var (
	ins = &registry.DefaultServiceInstance{
		ID: "1",
		Metadata: map[string]string{
			constant.MetadataServiceURLParamsPropertyName: `{
				"application": "dubbo-go",
				"group": "BDTService",
				"port": "64658",
				"protocol": "dubbo",
				"version": "1.0.0"
			}`,
		},
		Host:        "dubbo.io",
		ServiceName: "dubbo-app",
	}
	metadataInfo = &info.MetadataInfo{
		App: "dubbo-app",
	}
)

func TestGetMetadataFromMetadataReport(t *testing.T) {
	t.Run("no report instance", func(t *testing.T) {
		_, err := GetMetadataFromMetadataReport("1", ins)
		assert.NotNil(t, err)
	})
	mockReport := new(mockMetadataReport)
	defer mockReport.AssertExpectations(t)
	instances["default"] = mockReport
	t.Run("normal", func(t *testing.T) {
		mockReport.On("GetAppMetadata").Return(metadataInfo, nil).Once()
		got, err := GetMetadataFromMetadataReport("1", ins)
		assert.Nil(t, err)
		assert.Equal(t, metadataInfo, got)
	})
	t.Run("error", func(t *testing.T) {
		mockReport.On("GetAppMetadata").Return(metadataInfo, errors.New("mock error")).Once()
		_, err := GetMetadataFromMetadataReport("1", ins)
		assert.NotNil(t, err)
	})
}

func TestGetMetadataFromRpc(t *testing.T) {
	mockInvoker := new(mockInvoker)
	defer mockInvoker.AssertExpectations(t)
	mockProtocol := new(mockProtocol)
	defer mockProtocol.AssertExpectations(t)
	extension.SetProtocol("dubbo", func() protocol.Protocol {
		return mockProtocol
	})

	result := &protocol.RPCResult{
		Attrs: map[string]interface{}{},
		Err:   nil,
		Rest:  metadataInfo,
	}
	t.Run("normal", func(t *testing.T) {
		mockProtocol.On("Refer").Return(mockInvoker).Once()
		mockInvoker.On("Invoke").Return(result).Once()
		mockInvoker.On("Destroy").Once()
		metadata, err := GetMetadataFromRpc("111", ins)
		assert.Nil(t, err)
		assert.Equal(t, metadata, result.Rest)
	})
	t.Run("refer error", func(t *testing.T) {
		mockProtocol.On("Refer").Return(nil).Once()
		_, err := GetMetadataFromRpc("111", ins)
		assert.NotNil(t, err)
	})
	t.Run("invoke timeout", func(t *testing.T) {
		mockProtocol.On("Refer").Return(mockInvoker).Once()
		mockInvoker.On("Invoke").Return(&protocol.RPCResult{
			Attrs: map[string]interface{}{},
			Err:   errors.New("timeout error"),
			Rest:  metadataInfo,
		}).Once()
		mockInvoker.On("Destroy").Once()
		_, err := GetMetadataFromRpc("111", ins)
		assert.NotNil(t, err)
	})
}

func Test_buildMetadataServiceURL(t *testing.T) {
	type args struct {
		ins registry.ServiceInstance
	}
	tests := []struct {
		name string
		args args
		want *common.URL
	}{
		{
			name: "normal",
			args: args{
				&registry.DefaultServiceInstance{
					ServiceName: "dubbo-app",
					Host:        "dubbo.io",
					Metadata: map[string]string{
						constant.ProtocolKey: "dubbo",
						constant.PortKey:     "3000",
						constant.MetadataServiceURLParamsPropertyName: `{
							"group":"dubbo_registry_nacos_server",
							"port":"3000",
							"protocol":"dubbo",
							"serialization":"hessian2",
							"version":"1.0.0"
						}`,
					},
				},
			},
			want: common.NewURLWithOptions(
				common.WithIp("dubbo.io"),
				common.WithProtocol("dubbo"),
				common.WithPath(constant.MetadataServiceName),
				common.WithProtocol("dubbo"),
				common.WithPort("3000"),
				common.WithParams(map[string][]string{
					constant.ProtocolKey:      {"dubbo"},
					constant.PortKey:          {"3000"},
					constant.SerializationKey: {constant.Hessian2Serialization},
					constant.VersionKey:       {"1.0.0"},
				}),
				common.WithParamsValue(constant.GroupKey, "dubbo-app"),
				common.WithParamsValue(constant.InterfaceKey, constant.MetadataServiceName),
			),
		},
		{
			name: "no protocol",
			args: args{
				&registry.DefaultServiceInstance{
					ServiceName: "dubbo-app",
					Host:        "dubbo.io",
					Metadata:    map[string]string{},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, buildStandardMetadataServiceURL(tt.args.ins), "buildMetadataServiceURL(%v)", tt.args.ins)
		})
	}
}

func Test_getMetadataServiceUrlParams(t *testing.T) {
	type args struct {
		ins registry.ServiceInstance
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "normal",
			args: args{
				&registry.DefaultServiceInstance{
					Metadata: map[string]string{
						constant.MetadataServiceURLParamsPropertyName: `{
							"application": "BDTService",
							"group": "BDTService",
							"port": "64658",
							"protocol": "dubbo",
							"release": "dubbo-golang-3.0.0",
							"timestamp": "1713432877",
							"version": "1.0.0"
						}`,
					},
				},
			},
			want: map[string]string{
				"application": "BDTService",
				"group":       "BDTService",
				"port":        "64658",
				"protocol":    "dubbo",
				"release":     "dubbo-golang-3.0.0",
				"timestamp":   "1713432877",
				"version":     "1.0.0",
			},
		},
		{
			name: "wrong format",
			args: args{
				&registry.DefaultServiceInstance{
					Metadata: map[string]string{
						constant.MetadataServiceURLParamsPropertyName: `xxx`,
					},
				},
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getMetadataServiceUrlParams(tt.args.ins), "getMetadataServiceUrlParams(%v)", tt.args.ins)
		})
	}
}

type mockProtocol struct {
	mock.Mock
}

func (m *mockProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	args := m.Called()
	return args.Get(0).(protocol.Exporter)
}

func (m *mockProtocol) Refer(url *common.URL) protocol.Invoker {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(protocol.Invoker)
}

func (m *mockProtocol) Destroy() {
}

type mockInvoker struct {
	mock.Mock
}

func (m *mockInvoker) GetURL() *common.URL {
	return common.NewURLWithOptions(common.WithProtocol(constant.DefaultProtocol))
}

func (m *mockInvoker) IsAvailable() bool {
	return true
}

func (m *mockInvoker) Destroy() {
	m.Called()
}

func (m *mockInvoker) Invoke(ctx context.Context, inv protocol.Invocation) protocol.Result {
	args := m.Mock.Called()
	meta := args.Get(0).(protocol.Result).Result().(*info.MetadataInfo)
	reply := inv.Reply().(*info.MetadataInfo)
	reply.App = meta.App
	reply.Tag = meta.Tag
	reply.Revision = meta.Revision
	reply.Services = meta.Services
	return args.Get(0).(protocol.Result)
}

type mockExporter struct {
	mock.Mock
}

func (m *mockExporter) GetInvoker() protocol.Invoker {
	args := m.Called()
	return args.Get(0).(protocol.Invoker)
}

func (m *mockExporter) UnExport() {
	m.Called()
}

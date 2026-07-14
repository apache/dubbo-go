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
	stderrors "errors"
	"testing"
)

import (
	pkgerrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	tripleapi "dubbo.apache.org/dubbo-go/v3/metadata/triple_api/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
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

func TestConvertMetadataInfoV2PreservesTag(t *testing.T) {
	got := convertMetadataInfoV2(&tripleapi.MetadataInfoV2{
		App:     "dubbo-app",
		Version: "revision",
		Tag:     "gray",
		Services: map[string]*tripleapi.ServiceInfoV2{
			"DemoService:tri": {
				Name:     "DemoService",
				Protocol: "tri",
			},
		},
	})

	require.NotNil(t, got)
	assert.Equal(t, "dubbo-app", got.App)
	assert.Equal(t, "revision", got.Revision)
	assert.Equal(t, "gray", got.Tag)
	assert.Contains(t, got.Services, "DemoService:tri")
}

func TestGetMetadataFromMetadataReport(t *testing.T) {
	t.Cleanup(func() { instances = make(map[string]report.MetadataReport) })

	t.Run("no report instance", func(t *testing.T) {
		instances = make(map[string]report.MetadataReport)
		_, err := GetMetadataFromMetadataReport("rev-missing-report", ins, "default")
		require.Error(t, err)

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindReportLoad, metadataErr.Kind)
		assert.Equal(t, "metadata_report", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "rev-missing-report", metadataErr.Revision)
		assert.Equal(t, "default", metadataErr.RegistryID)
	})

	t.Run("default registry routes to default report", func(t *testing.T) {
		instances = make(map[string]report.MetadataReport)
		mockReport := new(mockMetadataReport)
		defer mockReport.AssertExpectations(t)
		instances["default"] = mockReport

		mockReport.On("GetAppMetadata").Return(metadataInfo, nil).Once()
		got, err := GetMetadataFromMetadataReport("1", ins, "default")
		require.NoError(t, err)
		assert.Equal(t, metadataInfo, got)
	})

	t.Run("specific registryId routes to its own report", func(t *testing.T) {
		instances = make(map[string]report.MetadataReport)
		defaultReport := new(mockMetadataReport)
		specificReport := new(mockMetadataReport)
		defer defaultReport.AssertExpectations(t)
		defer specificReport.AssertExpectations(t)
		instances["default"] = defaultReport
		instances["reg-a"] = specificReport

		// specificReport must be called; defaultReport must NOT be called
		specificReport.On("GetAppMetadata").Return(metadataInfo, nil).Once()
		got, err := GetMetadataFromMetadataReport("1", ins, "reg-a")
		require.NoError(t, err)
		assert.Equal(t, metadataInfo, got)
	})

	t.Run("unknown registryId falls back to default report", func(t *testing.T) {
		instances = make(map[string]report.MetadataReport)
		defaultReport := new(mockMetadataReport)
		defer defaultReport.AssertExpectations(t)
		instances["default"] = defaultReport

		// When the specific registryId is not found, it falls back to "default"
		// so the default report's GetAppMetadata is called
		defaultReport.On("GetAppMetadata").Return(metadataInfo, nil).Once()
		got, err := GetMetadataFromMetadataReport("1", ins, "nonexistent-registry")
		require.NoError(t, err)
		assert.Equal(t, metadataInfo, got)
	})

	t.Run("report error propagated", func(t *testing.T) {
		instances = make(map[string]report.MetadataReport)
		mockReport := new(mockMetadataReport)
		defer mockReport.AssertExpectations(t)
		instances["default"] = mockReport

		mockReport.On("GetAppMetadata").Return(metadataInfo, pkgerrors.New("mock error")).Once()
		_, err := GetMetadataFromMetadataReport("rev-report-error", ins, "default")
		require.Error(t, err)

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindReportLoad, metadataErr.Kind)
		assert.Equal(t, "metadata_report", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "rev-report-error", metadataErr.Revision)
		assert.Equal(t, "default", metadataErr.RegistryID)
	})
}

func TestGetMetadataFromRpc(t *testing.T) {
	mockInvoker := new(mockInvoker)
	defer mockInvoker.AssertExpectations(t)
	mockProtocol := new(mockProtocol)
	defer mockProtocol.AssertExpectations(t)
	extension.SetProtocol("dubbo", func() base.Protocol {
		return mockProtocol
	})

	res := &result.RPCResult{
		Attrs: map[string]any{},
		Err:   nil,
		Rest:  metadataInfo,
	}
	t.Run("normal", func(t *testing.T) {
		mockProtocol.On("Refer").Return(mockInvoker).Once()
		mockInvoker.On("Invoke").Return(res).Once()
		mockInvoker.On("Destroy").Once()
		metadata, err := GetMetadataFromRpc("111", ins)
		require.NoError(t, err)
		assert.Equal(t, metadata, res.Rest)
	})
	t.Run("refer error", func(t *testing.T) {
		mockProtocol.On("Refer").Return(nil).Once()
		_, err := GetMetadataFromRpc("rev-refer-error", ins)
		require.Error(t, err)

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindRPCLoad, metadataErr.Kind)
		assert.Equal(t, "rpc_metadata", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "rev-refer-error", metadataErr.Revision)
	})
	t.Run("invoke timeout", func(t *testing.T) {
		mockProtocol.On("Refer").Return(mockInvoker).Once()
		mockInvoker.On("Invoke").Return(&result.RPCResult{
			Attrs: map[string]any{},
			Err:   pkgerrors.New("timeout error"),
			Rest:  metadataInfo,
		}).Once()
		mockInvoker.On("Destroy").Once()
		_, err := GetMetadataFromRpc("rev-timeout", ins)
		require.Error(t, err)

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindRPCLoad, metadataErr.Kind)
		assert.Equal(t, "rpc_metadata", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "rev-timeout", metadataErr.Revision)
	})
	t.Run("nil response", func(t *testing.T) {
		mockProtocol.On("Refer").Return(mockInvoker).Once()
		mockInvoker.On("Invoke").Return(&result.RPCResult{
			Attrs: map[string]any{},
			Err:   nil,
			Rest:  nil,
		}).Once()
		mockInvoker.On("Destroy").Once()
		_, err := GetMetadataFromRpc("rev-nil", ins)
		require.Error(t, err)

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindNil, metadataErr.Kind)
		assert.Equal(t, "rpc_metadata", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "rev-nil", metadataErr.Revision)
	})
	t.Run("unexpected response type", func(t *testing.T) {
		mockProtocol.On("Refer").Return(mockInvoker).Once()
		mockInvoker.On("Invoke").Return(&result.RPCResult{
			Attrs: map[string]any{},
			Err:   nil,
			Rest:  123,
		}).Once()
		mockInvoker.On("Destroy").Once()
		_, err := GetMetadataFromRpc("rev-unexpected", ins)
		require.Error(t, err)

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindRPCLoad, metadataErr.Kind)
		assert.Equal(t, "rpc_metadata", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "rev-unexpected", metadataErr.Revision)
	})
}

func TestGetMetadataFromRpc_MissingURLParams(t *testing.T) {
	t.Run("malformed params", func(t *testing.T) {
		insMalformedParams := &registry.DefaultServiceInstance{
			ID:          "4",
			ServiceName: "dubbo-app",
			Host:        "dubbo.io",
			Metadata: map[string]string{
				constant.MetadataServiceURLParamsPropertyName: `xxx`,
			},
		}

		_, err := GetMetadataFromRpc("rev-malformed-params", insMalformedParams)
		require.Error(t, err)

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindURLBuild, metadataErr.Kind)
		assert.Equal(t, "metadata_url", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "rev-malformed-params", metadataErr.Revision)
	})

	t.Run("missing protocol", func(t *testing.T) {
		insNoProto := &registry.DefaultServiceInstance{
			ID:          "2",
			ServiceName: "dubbo-app",
			Host:        "dubbo.io",
			Metadata:    map[string]string{},
		}
		_, err := GetMetadataFromRpc("1", insNoProto)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "protocol is empty")

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindURLBuild, metadataErr.Kind)
		assert.Equal(t, "metadata_url", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "1", metadataErr.Revision)
	})

	t.Run("missing port", func(t *testing.T) {
		insNoPort := &registry.DefaultServiceInstance{
			ID:          "3",
			ServiceName: "dubbo-app",
			Host:        "dubbo.io",
			Metadata: map[string]string{
				constant.MetadataServiceURLParamsPropertyName: `{"protocol":"dubbo"}`,
			},
		}
		_, err := GetMetadataFromRpc("1", insNoPort)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "port is empty")

		var metadataErr *MetadataError
		require.True(t, stderrors.As(err, &metadataErr))
		assert.Equal(t, MetadataErrorKindURLBuild, metadataErr.Kind)
		assert.Equal(t, "metadata_url", metadataErr.Source)
		assert.Equal(t, "dubbo-app", metadataErr.App)
		assert.Equal(t, "1", metadataErr.Revision)
	})
}

func Test_buildMetadataServiceURL(t *testing.T) {
	type args struct {
		ins registry.ServiceInstance
	}
	tests := []struct {
		name    string
		args    args
		want    *common.URL
		wantErr string
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
			wantErr: "protocol is empty",
		},
		{
			name: "no port",
			args: args{
				&registry.DefaultServiceInstance{
					ServiceName: "dubbo-app",
					Host:        "dubbo.io",
					Metadata: map[string]string{
						constant.MetadataServiceURLParamsPropertyName: `{
							"protocol":"dubbo"
						}`,
					},
				},
			},
			wantErr: "port is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildStandardMetadataServiceURL(tt.args.ins)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "buildMetadataServiceURL(%v)", tt.args.ins)
		})
	}
}

func Test_getMetadataServiceUrlParams(t *testing.T) {
	type args struct {
		ins registry.ServiceInstance
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
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
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMetadataServiceUrlParams(tt.args.ins)
			if tt.wantErr {
				require.Error(t, err)
				var metadataErr *MetadataError
				require.True(t, stderrors.As(err, &metadataErr))
				assert.Equal(t, MetadataErrorKindURLBuild, metadataErr.Kind)
				return
			}
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "getMetadataServiceUrlParams(%v)", tt.args.ins)
		})
	}
}

type mockProtocol struct {
	mock.Mock
}

func (m *mockProtocol) Export(invoker base.Invoker) base.Exporter {
	args := m.Called()
	return args.Get(0).(base.Exporter)
}

func (m *mockProtocol) Refer(url *common.URL) base.Invoker {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(base.Invoker)
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

func (m *mockInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	args := m.Called()
	res := args.Get(0).(result.Result)

	// Handle both *info.MetadataInfo and *interface{} reply types
	// This supports the new implementation that uses interface{} to handle different return types
	if replyPtr, ok := inv.Reply().(*any); ok {
		*replyPtr = res.Result()
	} else if reply, ok := inv.Reply().(*info.MetadataInfo); ok {
		// Old code path: reply is *info.MetadataInfo, copy fields
		meta := res.Result().(*info.MetadataInfo)
		reply.App = meta.App
		reply.Tag = meta.Tag
		reply.Revision = meta.Revision
		reply.Services = meta.Services
	}

	return res
}

type mockExporter struct {
	mock.Mock
}

func (m *mockExporter) GetInvoker() base.Invoker {
	args := m.Called()
	return args.Get(0).(base.Invoker)
}

func (m *mockExporter) UnExport() {
	m.Called()
}

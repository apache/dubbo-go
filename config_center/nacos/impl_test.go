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

// nolint
package nacos

import (
	"reflect"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"

	"github.com/golang/mock/gomock"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

// MockIConfigClient is a mock of IConfigClient interface
type MockIConfigClient struct {
	ctrl     *gomock.Controller
	recorder *MockIConfigClientMockRecorder
}

// MockIConfigClientMockRecorder is the mock recorder for MockIConfigClient
type MockIConfigClientMockRecorder struct {
	mock *MockIConfigClient
}

// NewMockIConfigClient creates a new mock instance
func NewMockIConfigClient(ctrl *gomock.Controller) *MockIConfigClient {
	mock := &MockIConfigClient{ctrl: ctrl}
	mock.recorder = &MockIConfigClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIConfigClient) EXPECT() *MockIConfigClientMockRecorder {
	return m.recorder
}

// GetConfig mocks base method
func (m *MockIConfigClient) GetConfig(param vo.ConfigParam) (string, error) {
	ret := m.ctrl.Call(m, "GetConfig", param)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConfig indicates an expected call of GetConfig
func (mr *MockIConfigClientMockRecorder) GetConfig(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfig", reflect.TypeOf((*MockIConfigClient)(nil).GetConfig), param)
}

// PublishConfig mocks base method
func (m *MockIConfigClient) PublishConfig(param vo.ConfigParam) (bool, error) {
	ret := m.ctrl.Call(m, "PublishConfig", param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PublishConfig indicates an expected call of PublishConfig
func (mr *MockIConfigClientMockRecorder) PublishConfig(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishConfig", reflect.TypeOf((*MockIConfigClient)(nil).PublishConfig), param)
}

// DeleteConfig mocks base method
func (m *MockIConfigClient) DeleteConfig(param vo.ConfigParam) (bool, error) {
	ret := m.ctrl.Call(m, "DeleteConfig", param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CloseClient client nacos client
func (mr *MockIConfigClient) CloseClient() {

}

// DeleteConfig indicates an expected call of DeleteConfig
func (mr *MockIConfigClientMockRecorder) DeleteConfig(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteConfig", reflect.TypeOf((*MockIConfigClient)(nil).DeleteConfig), param)
}

// ListenConfig mocks base method
func (m *MockIConfigClient) ListenConfig(params vo.ConfigParam) error {
	ret := m.ctrl.Call(m, "ListenConfig", params)
	ret0, _ := ret[0].(error)
	return ret0
}

// ListenConfig indicates an expected call of ListenConfig
func (mr *MockIConfigClientMockRecorder) ListenConfig(params interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListenConfig", reflect.TypeOf((*MockIConfigClient)(nil).ListenConfig), params)
}

// CancelListenConfig mocks base method
func (m *MockIConfigClient) CancelListenConfig(params vo.ConfigParam) error {
	ret := m.ctrl.Call(m, "CancelListenConfig", params)
	ret0, _ := ret[0].(error)
	return ret0
}

// CancelListenConfig indicates an expected call of CancelListenConfig
func (mr *MockIConfigClientMockRecorder) CancelListenConfig(params interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelListenConfig", reflect.TypeOf((*MockIConfigClient)(nil).CancelListenConfig), params)
}

// SearchConfig mocks base method
func (m *MockIConfigClient) SearchConfig(param vo.SearchConfigParam) (*model.ConfigPage, error) {
	ret := m.ctrl.Call(m, "SearchConfig", param)
	ret0, _ := ret[0].(*model.ConfigPage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SearchConfig indicates an expected call of SearchConfig
func (mr *MockIConfigClientMockRecorder) SearchConfig(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SearchConfig", reflect.TypeOf((*MockIConfigClient)(nil).SearchConfig), param)
}

// PublishAggr mocks base method
func (m *MockIConfigClient) PublishAggr(param vo.ConfigParam) (bool, error) {
	ret := m.ctrl.Call(m, "PublishAggr", param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PublishAggr indicates an expected call of PublishAggr
func (mr *MockIConfigClientMockRecorder) PublishAggr(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishAggr", reflect.TypeOf((*MockIConfigClient)(nil).PublishAggr), param)
}

type fields struct {
	BaseDynamicConfiguration config_center.BaseDynamicConfiguration
	url                      *common.URL
	rootPath                 string
	done                     chan struct{}
	client                   *nacosClient.NacosConfigClient
	parser                   parser.ConfigurationParser
}
type args struct {
	key   string
	group string
	value string
}

func newnNacosDynamicConfiguration(f *fields) *nacosDynamicConfiguration {
	return &nacosDynamicConfiguration{
		BaseDynamicConfiguration: f.BaseDynamicConfiguration,
		url:                      f.url,
		rootPath:                 f.rootPath,
		done:                     f.done,
		client:                   f.client,
		parser:                   f.parser,
	}
}

func Test_nacosDynamicConfiguration_PublishConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	mnc := NewMockIConfigClient(ctrl)
	mnc.EXPECT().PublishConfig(gomock.Any()).Return(true, nil)
	nc := &nacosClient.NacosConfigClient{}
	nc.SetClient(mnc)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: nc,
			},
			args: args{
				key:   "dubbo.properties",
				group: "dubbogo",
				value: "dubbo.protocol.name=dubbo",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newnNacosDynamicConfiguration(&tt.fields)
			if err := n.PublishConfig(tt.args.key, tt.args.group, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("PublishConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_nacosDynamicConfiguration_GetConfigKeysByGroup(t *testing.T) {
	cp := &model.ConfigPage{
		PageItems: []model.ConfigItem{
			{
				DataId: "dubbogo",
			},
		},
	}
	result := gxset.NewSet()
	result.Add("dubbogo")
	ctrl := gomock.NewController(t)
	mnc := NewMockIConfigClient(ctrl)
	mnc.EXPECT().SearchConfig(gomock.Any()).Return(cp, nil)
	nc := &nacosClient.NacosConfigClient{}
	nc.SetClient(mnc)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *gxset.HashSet
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: nc,
			},
			args: args{
				group: "dubbo",
			},
			want:    result,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newnNacosDynamicConfiguration(&tt.fields)
			got, err := n.GetConfigKeysByGroup(tt.args.group)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigKeysByGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfigKeysByGroup() got = %v, want %v", got, tt.want)
			}
		})
	}
}

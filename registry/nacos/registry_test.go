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
	"net/url"
	"reflect"
	"strconv"
	"testing"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"

	"github.com/golang/mock/gomock"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// MockINamingClient is a mock of INamingClient interface
type MockINamingClient struct {
	ctrl     *gomock.Controller
	recorder *MockINamingClientMockRecorder
}

// MockINamingClientMockRecorder is the mock recorder for MockINamingClient
type MockINamingClientMockRecorder struct {
	mock *MockINamingClient
}

// NewMockINamingClient creates a new mock instance
func NewMockINamingClient(ctrl *gomock.Controller) *MockINamingClient {
	mock := &MockINamingClient{ctrl: ctrl}
	mock.recorder = &MockINamingClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockINamingClient) EXPECT() *MockINamingClientMockRecorder {
	return m.recorder
}

// RegisterInstance mocks base method
func (m *MockINamingClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	ret := m.ctrl.Call(m, "RegisterInstance", param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RegisterInstance indicates an expected call of RegisterInstance
func (mr *MockINamingClientMockRecorder) RegisterInstance(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterInstance", reflect.TypeOf((*MockINamingClient)(nil).RegisterInstance), param)
}

// DeregisterInstance mocks base method
func (m *MockINamingClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	ret := m.ctrl.Call(m, "DeregisterInstance", param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeregisterInstance indicates an expected call of DeregisterInstance
func (mr *MockINamingClientMockRecorder) DeregisterInstance(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeregisterInstance", reflect.TypeOf((*MockINamingClient)(nil).DeregisterInstance), param)
}

// UpdateInstance mocks base method
func (m *MockINamingClient) UpdateInstance(param vo.UpdateInstanceParam) (bool, error) {
	ret := m.ctrl.Call(m, "UpdateInstance", param)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateInstance indicates an expected call of UpdateInstance
func (mr *MockINamingClientMockRecorder) UpdateInstance(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateInstance", reflect.TypeOf((*MockINamingClient)(nil).UpdateInstance), param)
}

// GetService mocks base method
func (m *MockINamingClient) GetService(param vo.GetServiceParam) (model.Service, error) {
	ret := m.ctrl.Call(m, "GetService", param)
	ret0, _ := ret[0].(model.Service)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CloseClient close nacos client
func (m *MockINamingClient) CloseClient() {
}

// GetService indicates an expected call of GetService
func (mr *MockINamingClientMockRecorder) GetService(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetService", reflect.TypeOf((*MockINamingClient)(nil).GetService), param)
}

// SelectAllInstances mocks base method
func (m *MockINamingClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	ret := m.ctrl.Call(m, "SelectAllInstances", param)
	ret0, _ := ret[0].([]model.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SelectAllInstances indicates an expected call of SelectAllInstances
func (mr *MockINamingClientMockRecorder) SelectAllInstances(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelectAllInstances", reflect.TypeOf((*MockINamingClient)(nil).SelectAllInstances), param)
}

// SelectInstances mocks base method
func (m *MockINamingClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	ret := m.ctrl.Call(m, "SelectInstances", param)
	ret0, _ := ret[0].([]model.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SelectInstances indicates an expected call of SelectInstances
func (mr *MockINamingClientMockRecorder) SelectInstances(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelectInstances", reflect.TypeOf((*MockINamingClient)(nil).SelectInstances), param)
}

// SelectOneHealthyInstance mocks base method
func (m *MockINamingClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	ret := m.ctrl.Call(m, "SelectOneHealthyInstance", param)
	ret0, _ := ret[0].(*model.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SelectOneHealthyInstance indicates an expected call of SelectOneHealthyInstance
func (mr *MockINamingClientMockRecorder) SelectOneHealthyInstance(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelectOneHealthyInstance", reflect.TypeOf((*MockINamingClient)(nil).SelectOneHealthyInstance), param)
}

// Subscribe mocks base method
func (m *MockINamingClient) Subscribe(param *vo.SubscribeParam) error {
	ret := m.ctrl.Call(m, "Subscribe", param)
	ret0, _ := ret[0].(error)
	return ret0
}

// Subscribe indicates an expected call of Subscribe
func (mr *MockINamingClientMockRecorder) Subscribe(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockINamingClient)(nil).Subscribe), param)
}

// Unsubscribe mocks base method
func (m *MockINamingClient) Unsubscribe(param *vo.SubscribeParam) error {
	ret := m.ctrl.Call(m, "Unsubscribe", param)
	ret0, _ := ret[0].(error)
	return ret0
}

// Unsubscribe indicates an expected call of Unsubscribe
func (mr *MockINamingClientMockRecorder) Unsubscribe(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribe", reflect.TypeOf((*MockINamingClient)(nil).Unsubscribe), param)
}

// GetAllServicesInfo mocks base method
func (m *MockINamingClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) (model.ServiceList, error) {
	ret := m.ctrl.Call(m, "GetAllServicesInfo", param)
	ret0, _ := ret[0].(model.ServiceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllServicesInfo indicates an expected call of GetAllServicesInfo
func (mr *MockINamingClientMockRecorder) GetAllServicesInfo(param interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllServicesInfo", reflect.TypeOf((*MockINamingClient)(nil).GetAllServicesInfo), param)
}

type fields struct {
	URL          *common.URL
	namingClient *nacosClient.NacosNamingClient
	registryUrls []*common.URL
}
type args struct {
	url            *common.URL
	notifyListener registry.NotifyListener
}

func newNacosRegistryForTest(f fields) *nacosRegistry {
	return &nacosRegistry{
		URL:          f.URL,
		namingClient: f.namingClient,
		registryUrls: f.registryUrls,
	}
}

func Test_nacosRegistry_Register(t *testing.T) {
	params := url.Values{}
	params.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	params.Set(constant.NacosNotLoadLocalCache, "true")
	params.Set(constant.ClientNameKey, "nacos-client")

	regURL, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(params))

	urlMap := url.Values{}
	urlMap.Set(constant.GroupKey, "guangzhou-idc")
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.InterfaceKey, "com.ikurento.user.UserProvider")
	urlMap.Set(constant.VersionKey, "1.0.0")
	urlMap.Set(constant.ClusterKey, "mock")
	urlMap.Set(constant.ClientNameKey, "nacos-client")
	testURL, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))

	ctrl := gomock.NewController(t)
	mnc := NewMockINamingClient(ctrl)
	mnc.EXPECT().RegisterInstance(gomock.Any()).Return(true, nil)
	nc := &nacosClient.NacosNamingClient{}
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
				URL:          regURL,
				namingClient: nc,
				registryUrls: nil,
			},
			args: args{
				url: testURL,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nr := newNacosRegistryForTest(tt.fields)
			if err := nr.Register(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_nacosRegistry_UnRegister(t *testing.T) {
	params := url.Values{}
	params.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	params.Set(constant.NacosNotLoadLocalCache, "true")
	params.Set(constant.ClientNameKey, "nacos-client")

	regURL, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(params))

	urlMap := url.Values{}
	urlMap.Set(constant.GroupKey, "guangzhou-idc")
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.InterfaceKey, "com.ikurento.user.UserProvider")
	urlMap.Set(constant.VersionKey, "1.0.0")
	urlMap.Set(constant.ClusterKey, "mock")
	urlMap.Set(constant.ClientNameKey, "nacos-client")
	testURL, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))

	ctrl := gomock.NewController(t)
	mnc := NewMockINamingClient(ctrl)
	mnc.EXPECT().DeregisterInstance(gomock.Any()).Return(true, nil)
	nc := &nacosClient.NacosNamingClient{}
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
				URL:          regURL,
				namingClient: nc,
				registryUrls: nil,
			},
			args: args{
				url: testURL,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nr := newNacosRegistryForTest(tt.fields)
			if err := nr.UnRegister(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("UnRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_nacosRegistry_Subscribe(t *testing.T) {
	params := url.Values{}
	params.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	params.Set(constant.NacosNotLoadLocalCache, "true")
	params.Set(constant.ClientNameKey, "nacos-client")

	regURL, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(params))

	urlMap := url.Values{}
	urlMap.Set(constant.GroupKey, "guangzhou-idc")
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.InterfaceKey, "com.ikurento.user.UserProvider")
	urlMap.Set(constant.VersionKey, "1.0.0")
	urlMap.Set(constant.ClusterKey, "mock")
	urlMap.Set(constant.ClientNameKey, "nacos-client")
	testURL, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))

	ctrl := gomock.NewController(t)
	mnc := NewMockINamingClient(ctrl)
	nc := &nacosClient.NacosNamingClient{}
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
				URL:          regURL,
				namingClient: nc,
				registryUrls: nil,
			},
			args: args{
				url: testURL,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nr := newNacosRegistryForTest(tt.fields)
			if err := nr.Subscribe(tt.args.url, tt.args.notifyListener); (err != nil) != tt.wantErr {
				t.Errorf("Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

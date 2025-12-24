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
	"encoding/json"
	"reflect"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/info"

	nacosClient "github.com/dubbogo/gost/database/kv/nacos"

	"github.com/golang/mock/gomock"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// MockIConfigClient is a mock of IConfigClient interface
type MockIConfigClient struct {
	ctrl     *gomock.Controller
	recorder *MockIConfigClientMockRecorder
}

type MockIConfigClientMockRecorder struct {
	mock *MockIConfigClient
}

func NewMockIConfigClient(ctrl *gomock.Controller) *MockIConfigClient {
	mock := &MockIConfigClient{ctrl: ctrl}
	mock.recorder = &MockIConfigClientMockRecorder{mock}
	return mock
}

func (m *MockIConfigClient) EXPECT() *MockIConfigClientMockRecorder { return m.recorder }
func (m *MockIConfigClient) CloseClient()                           {}

func (m *MockIConfigClient) GetConfig(param vo.ConfigParam) (string, error) {
	ret := m.ctrl.Call(m, "GetConfig", param)
	return ret[0].(string), ret[1].(error)
}

func (mr *MockIConfigClientMockRecorder) GetConfig(param any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfig", reflect.TypeOf((*MockIConfigClient)(nil).GetConfig), param)
}

func (m *MockIConfigClient) PublishConfig(param vo.ConfigParam) (bool, error) {
	ret := m.ctrl.Call(m, "PublishConfig", param)
	return ret[0].(bool), ret[1].(error)
}

func (mr *MockIConfigClientMockRecorder) PublishConfig(param any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishConfig", reflect.TypeOf((*MockIConfigClient)(nil).PublishConfig), param)
}

func (m *MockIConfigClient) DeleteConfig(param vo.ConfigParam) (bool, error) {
	ret := m.ctrl.Call(m, "DeleteConfig", param)
	return ret[0].(bool), ret[1].(error)
}

func (mr *MockIConfigClientMockRecorder) DeleteConfig(param any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteConfig", reflect.TypeOf((*MockIConfigClient)(nil).DeleteConfig), param)
}

func (m *MockIConfigClient) ListenConfig(params vo.ConfigParam) error {
	return m.ctrl.Call(m, "ListenConfig", params)[0].(error)
}

func (mr *MockIConfigClientMockRecorder) ListenConfig(params any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListenConfig", reflect.TypeOf((*MockIConfigClient)(nil).ListenConfig), params)
}

func (m *MockIConfigClient) CancelListenConfig(params vo.ConfigParam) error {
	return m.ctrl.Call(m, "CancelListenConfig", params)[0].(error)
}

func (mr *MockIConfigClientMockRecorder) CancelListenConfig(params any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelListenConfig", reflect.TypeOf((*MockIConfigClient)(nil).CancelListenConfig), params)
}

func (m *MockIConfigClient) SearchConfig(param vo.SearchConfigParam) (*model.ConfigPage, error) {
	ret := m.ctrl.Call(m, "SearchConfig", param)
	return ret[0].(*model.ConfigPage), ret[1].(error)
}

func (mr *MockIConfigClientMockRecorder) SearchConfig(param any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SearchConfig", reflect.TypeOf((*MockIConfigClient)(nil).SearchConfig), param)
}

func (m *MockIConfigClient) PublishAggr(param vo.ConfigParam) (bool, error) {
	ret := m.ctrl.Call(m, "PublishAggr", param)
	return ret[0].(bool), ret[1].(error)
}

func (mr *MockIConfigClientMockRecorder) PublishAggr(param any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishAggr", reflect.TypeOf((*MockIConfigClient)(nil).PublishAggr), param)
}

func createMockClient(ctrl *gomock.Controller) *nacosClient.NacosConfigClient {
	mnc := NewMockIConfigClient(ctrl)
	nc := &nacosClient.NacosConfigClient{}
	nc.SetClient(mnc)
	return nc
}

func TestNacosMetadataReportGetAppMetadata(t *testing.T) {
	mi := info.MetadataInfo{App: "test-app"}
	data, _ := json.Marshal(mi)

	ctrl := gomock.NewController(t)
	mnc := NewMockIConfigClient(ctrl)
	mnc.EXPECT().GetConfig(gomock.Any()).Return(string(data), nil)

	nc := &nacosClient.NacosConfigClient{}
	nc.SetClient(mnc)
	n := &nacosMetadataReport{client: nc}

	got, err := n.GetAppMetadata("dubbo", "revision")
	if err != nil {
		t.Errorf("GetAppMetadata() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, &mi) {
		t.Errorf("GetAppMetadata() got = %v, want %v", got, &mi)
	}
}

func TestNacosMetadataReportPublishAppMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	mnc := NewMockIConfigClient(ctrl)
	mnc.EXPECT().PublishConfig(gomock.Any()).Return(true, nil)

	nc := &nacosClient.NacosConfigClient{}
	nc.SetClient(mnc)
	n := &nacosMetadataReport{client: nc}

	err := n.PublishAppMetadata("app", "rev", &info.MetadataInfo{App: "test"})
	if err != nil {
		t.Errorf("PublishAppMetadata() error = %v", err)
	}
}

func TestNacosMetadataReportRegisterServiceAppMapping(t *testing.T) {
	ctrl := gomock.NewController(t)
	mnc := NewMockIConfigClient(ctrl)
	mnc.EXPECT().GetConfig(gomock.Any()).Return("oldValue", nil)

	nc := &nacosClient.NacosConfigClient{}
	nc.SetClient(mnc)
	n := &nacosMetadataReport{client: nc}

	err := n.RegisterServiceAppMapping("key", "group", "oldValue")
	if err != nil {
		t.Errorf("RegisterServiceAppMapping() error = %v", err)
	}
}

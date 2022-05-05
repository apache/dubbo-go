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

package etcd

/*
import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

import (
	"github.com/agiledragon/gomonkey"

	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"

	"go.etcd.io/etcd/client/v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
)

func newSubscribeMetadataIdentifier() *identifier.SubscriberMetadataIdentifier {
	return &identifier.SubscriberMetadataIdentifier{
		Revision: "subscribe",
		BaseApplicationMetadataIdentifier: identifier.BaseApplicationMetadataIdentifier{
			Application: "provider",
		},
	}
}

func newServiceMetadataIdentifier() *identifier.ServiceMetadataIdentifier {
	return &identifier.ServiceMetadataIdentifier{
		Protocol: "nacos",
		Revision: "a",
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: "com.test.MyTest",
			Version:          "1.0.0",
			Group:            "test_group",
			Side:             "service",
		},
	}
}

func newMetadataIdentifier(side string) *identifier.MetadataIdentifier {
	return &identifier.MetadataIdentifier{
		Application: "test",
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: "com.test.MyTest",
			Version:          "1.0.0",
			Group:            "test_group",
			Side:             side,
		},
	}
}

type fields struct {
	client *gxetcd.Client
	root   string
}
type args struct {
	subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier
	info                         *common.MetadataInfo
	providerIdentifier           *identifier.MetadataIdentifier
	serviceDefinitions           string
	consumerMetadataIdentifier   *identifier.MetadataIdentifier
	serviceParameterString       string
	serviceMetadataIdentifier    *identifier.ServiceMetadataIdentifier
	url                          *common.URL
	urls                         string
}

func newEtcdMetadataReport(f fields) *etcdMetadataReport {
	return &etcdMetadataReport{
		client: f.client,
		root:   f.root,
	}
}

func Test_etcdMetadataReport_PublishAppMetadata(t *testing.T) {
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "Put", func(_ *gxetcd.Client, k, v string, opts ...clientv3.OpOption) error {
		return nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
				root:   "/dubbo",
			},
			args: args{
				subscriberMetadataIdentifier: newSubscribeMetadataIdentifier(),
				info:                         &common.MetadataInfo{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEtcdMetadataReport(tt.fields)
			if err := e.PublishAppMetadata(tt.args.subscriberMetadataIdentifier, tt.args.info); (err != nil) != tt.wantErr {
				t.Errorf("PublishAppMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_etcdMetadataReport_StoreProviderMetadata(t *testing.T) {
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "Put", func(_ *gxetcd.Client, k, v string, opts ...clientv3.OpOption) error {
		return nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
				root:   "/dubbo",
			},
			args: args{
				providerIdentifier: newMetadataIdentifier("provuder"),
				serviceDefinitions: "provider",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEtcdMetadataReport(tt.fields)
			if err := e.StoreProviderMetadata(tt.args.providerIdentifier, tt.args.serviceDefinitions); (err != nil) != tt.wantErr {
				t.Errorf("StoreProviderMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_etcdMetadataReport_StoreConsumerMetadata(t *testing.T) {
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "Put", func(_ *gxetcd.Client, k, v string, opts ...clientv3.OpOption) error {
		return nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
				root:   "/dubbo",
			},
			args: args{
				consumerMetadataIdentifier: newMetadataIdentifier("conusmer"),
				serviceParameterString:     "conusmer",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEtcdMetadataReport(tt.fields)
			if err := e.StoreConsumerMetadata(tt.args.consumerMetadataIdentifier, tt.args.serviceParameterString); (err != nil) != tt.wantErr {
				t.Errorf("StoreConsumerMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_etcdMetadataReport_SaveServiceMetadata(t *testing.T) {
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "Put", func(_ *gxetcd.Client, k, v string, opts ...clientv3.OpOption) error {
		return nil
	})
	defer patches.Reset()
	serviceURL, _ := common.NewURL("registry://localhost:8848", common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)))

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
				root:   "/dubbo",
			},
			args: args{
				serviceMetadataIdentifier: newServiceMetadataIdentifier(),
				url:                       serviceURL,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEtcdMetadataReport(tt.fields)
			if err := e.SaveServiceMetadata(tt.args.serviceMetadataIdentifier, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("SaveServiceMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_etcdMetadataReport_SaveSubscribedData(t *testing.T) {
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "Put", func(_ *gxetcd.Client, k, v string, opts ...clientv3.OpOption) error {
		return nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
				root:   "/dubbo",
			},
			args: args{
				subscriberMetadataIdentifier: newSubscribeMetadataIdentifier(),
				urls:                         "dubbogo",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEtcdMetadataReport(tt.fields)
			if err := e.SaveSubscribedData(tt.args.subscriberMetadataIdentifier, tt.args.urls); (err != nil) != tt.wantErr {
				t.Errorf("SaveSubscribedData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_etcdMetadataReport_RemoveServiceMetadata(t *testing.T) {
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "Delete", func(_ *gxetcd.Client, k string) error {
		return nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
				root:   DEFAULT_ROOT,
			},
			args: args{
				serviceMetadataIdentifier: newServiceMetadataIdentifier(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEtcdMetadataReport(tt.fields)
			if err := e.RemoveServiceMetadata(tt.args.serviceMetadataIdentifier); (err != nil) != tt.wantErr {
				t.Errorf("RemoveServiceMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_etcdMetadataReport_GetAppMetadata(t *testing.T) {
	info := &common.MetadataInfo{}
	target, _ := json.Marshal(info)
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "Get", func(_ *gxetcd.Client, k string) (string, error) {
		return string(target), nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *common.MetadataInfo
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
				root:   DEFAULT_ROOT,
			},
			args: args{
				subscriberMetadataIdentifier: newSubscribeMetadataIdentifier(),
			},
			want:    &common.MetadataInfo{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEtcdMetadataReport(tt.fields)
			got, err := e.GetAppMetadata(tt.args.subscriberMetadataIdentifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAppMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAppMetadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}
*/

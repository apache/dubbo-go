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

package client

import (
	"reflect"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestCreateRpcClientByUrl(t *testing.T) {
	//url, err := common.NewURL("dubbo://10.253.77.49:61086?group=dubbo.io&interface=org.apache.dubbo.metadata.MetadataService&side=consumer&version=1.0.0")
	//service, destory := createRpcClientByUrl(url)
	//defer destory()
	//meta, err := service.GetMetadataInfo(context.Background(), "4647984502")
	//if err != nil {
	//	logger.Error(err)
	//}
	//fmt.Println(meta)
}

func TestGetMetadataFromMetadataReport(t *testing.T) {
	type args struct {
		revision string
		instance registry.ServiceInstance
	}
	tests := []struct {
		name    string
		args    args
		want    *info.MetadataInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMetadataFromMetadataReport(tt.args.revision, tt.args.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetadataFromMetadataReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMetadataFromMetadataReport() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMetadataFromRpc(t *testing.T) {
	type args struct {
		revision string
		instance registry.ServiceInstance
	}
	tests := []struct {
		name    string
		args    args
		want    *info.MetadataInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMetadataFromRpc(tt.args.revision, tt.args.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetadataFromRpc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMetadataFromRpc() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildMetadataServiceURL(t *testing.T) {
	type args struct {
		serviceName string
		host        string
		ps          map[string]string
	}
	tests := []struct {
		name string
		args args
		want *common.URL
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildMetadataServiceURL(tt.args.serviceName, tt.args.host, tt.args.ps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildMetadataServiceURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createRpcClient(t *testing.T) {
	type args struct {
		instance registry.ServiceInstance
	}
	tests := []struct {
		name  string
		args  args
		want  *metadataService
		want1 func()
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := createRpcClient(tt.args.instance)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createRpcClient() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("createRpcClient() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_createRpcClientByUrl(t *testing.T) {
	type args struct {
		url *common.URL
	}
	tests := []struct {
		name  string
		args  args
		want  *metadataService
		want1 func()
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := createRpcClientByUrl(tt.args.url)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createRpcClientByUrl() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("createRpcClientByUrl() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getMetadataServiceUrlParams(t *testing.T) {
	type args struct {
		jsonStr string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMetadataServiceUrlParams(tt.args.jsonStr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMetadataServiceUrlParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

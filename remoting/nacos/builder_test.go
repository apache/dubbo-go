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

/*
import (
	"net/url"
	"reflect"
	"testing"
)

import (
	"github.com/agiledragon/gomonkey"

	nacosClient "github.com/dubbogo/gost/database/kv/nacos"

	nacosConstant "github.com/nacos-group/nacos-sdk-gocommon/constant"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
)

func getRegURL() *common.URL {
	regURLMap := url.Values{}
	regURLMap.Set(constant.NacosNotLoadLocalCache, "true")
	regURLMap.Set(constant.NacosNamespaceID, "nacos")
	regURLMap.Set(constant.TimeoutKey, "5s")
	regURLMap.Set(constant.ClientNameKey, "nacos-client")
	regURL, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(regURLMap))

	return regURL
}

type args struct {
	url *common.URL
	rc  *config.RemoteConfig
}

func TestNewNacosClientByURL(t *testing.T) {
	patches := gomonkey.NewPatches()
	patches = patches.ApplyFunc(nacosClient.NewNacosNamingClient, func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosNamingClient, error) {
		return &nacosClient.NacosNamingClient{}, nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		args    args
		want    *nacosClient.NacosNamingClient
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				url: getRegURL(),
			},
			want:    &nacosClient.NacosNamingClient{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNacosClientByURL(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNacosClientByURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNacosClientByURL() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewNacosClient(t *testing.T) {
	patches := gomonkey.NewPatches()
	patches = patches.ApplyFunc(nacosClient.NewNacosNamingClient, func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosNamingClient, error) {
		return &nacosClient.NacosNamingClient{}, nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		args    args
		want    *nacosClient.NacosNamingClient
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				rc: &config.RemoteConfig{
					Address:  "test.nacos.io:80/nacos",
					Protocol: "nacos",
					Timeout:  "10s",
					Username: "naocs",
					Password: "nacos",
				},
			},
			want:    &nacosClient.NacosNamingClient{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNacosClient(tt.args.rc)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNacosClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNacosClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNacosConfig(t *testing.T) {
	tests := []struct {
		name    string
		args    args
		want    []nacosConstant.ServerConfig
		want1   nacosConstant.ClientConfig
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				url: getRegURL(),
			},
			want: []nacosConstant.ServerConfig{
				{
					IpAddr: "test.nacos.io",
					Port:   80,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetNacosConfig(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNacosConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNacosConfig() got = %v, want %v", got, tt.want)
			}
			assert.NotNil(t, got1)
		})
	}
}

func TestNewNacosConfigClientByUrl(t *testing.T) {
	patches := gomonkey.NewPatches()
	patches = patches.ApplyFunc(nacosClient.NewNacosNamingClient, func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosNamingClient, error) {
		return &nacosClient.NacosNamingClient{}, nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		args    args
		want    *nacosClient.NacosConfigClient
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				url: getRegURL(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNacosConfigClientByUrl(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNacosConfigClientByUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.NotNil(t, got)
		})
	}
}

*/

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
	"testing"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"

	nacosConstant "github.com/nacos-group/nacos-sdk-go/v2/common/constant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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
}

func TestNewNacosClientByURL(t *testing.T) {
	oldNewNacosNamingClient := newNacosNamingClient
	newNacosNamingClient = func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosNamingClient, error) {
		return &nacosClient.NacosNamingClient{}, nil
	}
	t.Cleanup(func() {
		newNacosNamingClient = oldNewNacosNamingClient
	})

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

func TestNewNacosClientByURLMissingClientName(t *testing.T) {
	regURLMap := url.Values{}
	regURLMap.Set(constant.NacosNamespaceID, "nacos")
	regURL, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(regURLMap))

	client, err := NewNacosClientByURL(regURL)
	assert.Nil(t, client)
	assert.Error(t, err)
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
		{
			name: "with_context_path",
			args: args{
				url: func() *common.URL {
					regURLMap := url.Values{}
					regURLMap.Set(constant.ClientNameKey, "nacos-client")
					regURL, _ := common.NewURL("registry://test.nacos.io:80/nacos", common.WithParams(regURLMap))
					return regURL
				}(),
			},
			want: []nacosConstant.ServerConfig{
				{
					IpAddr:      "test.nacos.io",
					Port:        80,
					ContextPath: "/nacos",
				},
			},
			wantErr: false,
		},
		{
			name: "endpoint_only",
			args: args{
				url: func() *common.URL {
					regURLMap := url.Values{}
					regURLMap.Set(constant.NacosEndpoint, "acm.aliyun.com")
					regURLMap.Set(constant.ClientNameKey, "nacos-client")
					regURL, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(regURLMap))
					return regURL
				}(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "nil_url",
			args: args{
				url: nil,
			},
			want:    []nacosConstant.ServerConfig{},
			wantErr: true,
		},
		{
			name: "empty_location",
			args: args{
				url: func() *common.URL {
					regURL, _ := common.NewURL("registry://", common.WithParams(url.Values{}))
					return regURL
				}(),
			},
			want:    []nacosConstant.ServerConfig{},
			wantErr: true,
		},
		{
			name: "bad_address",
			args: args{
				url: func() *common.URL {
					regURL, _ := common.NewURL("registry://bad-address", common.WithParams(url.Values{}))
					return regURL
				}(),
			},
			want:    []nacosConstant.ServerConfig{},
			wantErr: true,
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
			if !tt.wantErr {
				assert.NotNil(t, got1)
			}
		})
	}
}

func TestNewNacosConfigClientByUrl(t *testing.T) {
	oldNewNacosConfigClient := newNacosConfigClient
	newNacosConfigClient = func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosConfigClient, error) {
		return &nacosClient.NacosConfigClient{}, nil
	}
	t.Cleanup(func() {
		newNacosConfigClient = oldNewNacosConfigClient
	})

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

func TestNewNacosConfigClientByUrlMissingClientName(t *testing.T) {
	regURLMap := url.Values{}
	regURLMap.Set(constant.NacosNamespaceID, "nacos")
	regURL, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(regURLMap))

	client, err := NewNacosConfigClientByUrl(regURL)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestNacosClientPoolKeySharedAcrossRoles(t *testing.T) {
	var captured []string
	oldNewNacosConfigClient := newNacosConfigClient
	newNacosConfigClient = func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosConfigClient, error) {
		captured = append(captured, name)
		assert.True(t, share)
		return &nacosClient.NacosConfigClient{}, nil
	}
	t.Cleanup(func() {
		newNacosConfigClient = oldNewNacosConfigClient
	})

	mkURL := func(clientName, namespace string) *common.URL {
		m := url.Values{}
		m.Set(constant.NacosNamespaceID, namespace)
		m.Set(constant.ClientNameKey, clientName)
		u, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(m))
		return u
	}

	// config-center and metadata-report carry role-scoped client names, but
	// point at the same cluster: they must resolve to one pool key (#3573).
	_, err := NewNacosConfigClientByUrl(mkURL("dubbo.config-center-nacos-test.nacos.io:80", "ns1"))
	require.NoError(t, err)
	_, err = NewNacosConfigClientByUrl(mkURL("dubbo.metadata-report-nacos-test.nacos.io:80", "ns1"))
	require.NoError(t, err)
	assert.Equal(t, captured[0], captured[1])

	// A different namespace is a different connection identity and must not
	// share the client.
	_, err = NewNacosConfigClientByUrl(mkURL("dubbo.config-center-nacos-test.nacos.io:80", "ns2"))
	require.NoError(t, err)
	assert.NotEqual(t, captured[0], captured[2])
}

func TestNacosClientPoolKeyCredentialAndEndpointIdentity(t *testing.T) {
	key := func(params map[string]string) string {
		m := url.Values{}
		for k, v := range params {
			m.Set(k, v)
		}
		u, _ := common.NewURL("registry://test.nacos.io:80", common.WithParams(m))
		return nacosClientPoolKey("config", u)
	}

	userPass := map[string]string{constant.NacosUsername: "alice", constant.NacosPassword: "s3cret-A"}

	// Same server and same full credential set share one key.
	assert.Equal(t, key(userPass), key(map[string]string{constant.NacosUsername: "alice", constant.NacosPassword: "s3cret-A"}))

	// A different password (or secretKey) is a differently authenticated
	// client and must never collapse into the same pool entry.
	assert.NotEqual(t, key(userPass), key(map[string]string{constant.NacosUsername: "alice", constant.NacosPassword: "s3cret-B"}))
	assert.NotEqual(t,
		key(map[string]string{constant.NacosAccessKey: "AKID", constant.NacosSecretKey: "SK-1"}),
		key(map[string]string{constant.NacosAccessKey: "AKID", constant.NacosSecretKey: "SK-2"}))

	// Credentials must not appear in the key verbatim (keys can be logged).
	assert.NotContains(t, key(userPass), "alice")
	assert.NotContains(t, key(userPass), "s3cret-A")

	// When an endpoint is set, GetNacosConfig ignores url.Location; the pool
	// key mirrors that, so differing (ignored) locations still share.
	withEndpoint := func(location string) string {
		m := url.Values{}
		m.Set(constant.NacosEndpoint, "acm.aliyun.com")
		u, _ := common.NewURL("registry://"+location, common.WithParams(m))
		return nacosClientPoolKey("config", u)
	}
	assert.Equal(t, withEndpoint("a.nacos.io:80"), withEndpoint("b.nacos.io:80"))
}

func TestNacosNamingAndConfigPoolKeysDistinct(t *testing.T) {
	var namingKey, configKey string
	oldNaming := newNacosNamingClient
	oldConfig := newNacosConfigClient
	newNacosNamingClient = func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosNamingClient, error) {
		namingKey = name
		return &nacosClient.NacosNamingClient{}, nil
	}
	newNacosConfigClient = func(name string, share bool, sc []nacosConstant.ServerConfig,
		cc nacosConstant.ClientConfig) (*nacosClient.NacosConfigClient, error) {
		configKey = name
		return &nacosClient.NacosConfigClient{}, nil
	}
	t.Cleanup(func() {
		newNacosNamingClient = oldNaming
		newNacosConfigClient = oldConfig
	})

	regURL := getRegURL()
	_, err := NewNacosClientByURL(regURL)
	require.NoError(t, err)
	_, err = NewNacosConfigClientByUrl(regURL)
	require.NoError(t, err)
	assert.NotEqual(t, namingKey, configKey)
}

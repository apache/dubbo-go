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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestNewOptions(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		wantApp   string
		wantType  string
		wantPort  int
		wantProto string
	}{
		{
			name:      "default",
			opts:      nil,
			wantApp:   "",
			wantType:  constant.DefaultMetadataStorageType,
			wantPort:  0,
			wantProto: constant.DefaultProtocol,
		},
		{
			name:      "with-app",
			opts:      []Option{WithAppName("my-app")},
			wantApp:   "my-app",
			wantType:  constant.DefaultMetadataStorageType,
			wantPort:  0,
			wantProto: constant.DefaultProtocol,
		},
		{
			name:      "with-type",
			opts:      []Option{WithMetadataType("remote")},
			wantApp:   "",
			wantType:  "remote",
			wantPort:  0,
			wantProto: constant.DefaultProtocol,
		},
		{
			name:      "with-port",
			opts:      []Option{WithPort(20880)},
			wantApp:   "",
			wantType:  constant.DefaultMetadataStorageType,
			wantPort:  20880,
			wantProto: constant.DefaultProtocol,
		},
		{
			name:      "with-protocol",
			opts:      []Option{WithMetadataProtocol("tri")},
			wantApp:   "",
			wantType:  constant.DefaultMetadataStorageType,
			wantPort:  0,
			wantProto: "tri",
		},
		{
			name: "all-set",
			opts: []Option{
				WithAppName("my-app"),
				WithMetadataType("remote"),
				WithPort(20880),
				WithMetadataProtocol("tri"),
			},
			wantApp:   "my-app",
			wantType:  "remote",
			wantPort:  20880,
			wantProto: "tri",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(tt.opts...)
			assert.Equal(t, tt.wantApp, opts.appName)
			assert.Equal(t, tt.wantType, opts.metadataType)
			assert.Equal(t, tt.wantPort, opts.port)
			assert.Equal(t, tt.wantProto, opts.protocol)
		})
	}
}

func TestNewReportOptions(t *testing.T) {
	tests := []struct {
		name string
		opts []ReportOption
		get  func(*ReportOptions) string
		want string
	}{
		{
			name: "default",
			opts: nil,
			get:  func(o *ReportOptions) string { return o.Protocol },
			want: "",
		},
		{
			name: "with-registryId",
			opts: []ReportOption{WithRegistryId("registry-1")},
			get:  func(o *ReportOptions) string { return o.registryId },
			want: "registry-1",
		},
		{
			name: "with-zookeeper",
			opts: []ReportOption{WithZookeeper()},
			get:  func(o *ReportOptions) string { return o.Protocol },
			want: constant.ZookeeperKey,
		},
		{
			name: "with-nacos",
			opts: []ReportOption{WithNacos()},
			get:  func(o *ReportOptions) string { return o.Protocol },
			want: constant.NacosKey,
		},
		{
			name: "with-etcdv3",
			opts: []ReportOption{WithEtcdV3()},
			get:  func(o *ReportOptions) string { return o.Protocol },
			want: constant.EtcdV3Key,
		},
		{
			name: "with-protocol-generic",
			opts: []ReportOption{WithProtocol("consul")},
			get:  func(o *ReportOptions) string { return o.Protocol },
			want: "consul",
		},
		{
			name: "with-address",
			opts: []ReportOption{WithAddress("127.0.0.1:2181")},
			get:  func(o *ReportOptions) string { return o.Address },
			want: "127.0.0.1:2181",
		},
		{
			name: "with-username",
			opts: []ReportOption{WithUsername("admin")},
			get:  func(o *ReportOptions) string { return o.Username },
			want: "admin",
		},
		{
			name: "with-password",
			opts: []ReportOption{WithPassword("secret")},
			get:  func(o *ReportOptions) string { return o.Password },
			want: "secret",
		},
		{
			name: "with-timeout",
			opts: []ReportOption{WithTimeout(5 * time.Second)},
			get:  func(o *ReportOptions) string { return o.Timeout },
			want: "5000",
		},
		{
			name: "with-group",
			opts: []ReportOption{WithGroup("test-group")},
			get:  func(o *ReportOptions) string { return o.Group },
			want: "test-group",
		},
		{
			name: "with-namespace",
			opts: []ReportOption{WithNamespace("test-ns")},
			get:  func(o *ReportOptions) string { return o.Namespace },
			want: "test-ns",
		},
		{
			name: "with-params",
			opts: []ReportOption{WithParams(map[string]string{"key": "value"})},
			get:  func(o *ReportOptions) string { return o.Params["key"] },
			want: "value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewReportOptions(tt.opts...)
			assert.Equal(t, tt.want, tt.get(o))
		})
	}
}

func TestWithAddressProtocolParsing(t *testing.T) {
	tests := []struct {
		address, expectedProtocol string
	}{
		{"127.0.0.1:2181", ""},
		{"zookeeper://127.0.0.1:2181", "zookeeper"},
		{"nacos://localhost:8848", "nacos"},
	}
	for _, tt := range tests {
		opts := defaultReportOptions()
		WithAddress(tt.address)(opts)
		assert.Equal(t, tt.expectedProtocol, opts.Protocol)
	}
}

func TestReportOptionsToUrl(t *testing.T) {
	// Valid options
	opts := NewReportOptions(
		WithZookeeper(),
		WithAddress("127.0.0.1:2181"),
		WithParams(map[string]string{"key": "value"}),
	)
	url, err := opts.toUrl()
	require.NoError(t, err)
	assert.Equal(t, "zookeeper", url.Protocol)
	assert.Equal(t, "zookeeper", url.GetParam("metadata", ""))
	assert.Equal(t, "value", url.GetParam("key", ""))

	// The typed switch must survive URL construction so the publisher can
	// distinguish an explicit false from the default-on absent value.
	disabled := NewReportOptions(
		WithNacos(),
		WithAddress("127.0.0.1:8848"),
		WithReportDefinition(false),
	)
	disabledURL, err := disabled.toUrl()
	require.NoError(t, err)
	assert.False(t, disabledURL.GetParamBool(constant.MetadataReportReportDefinitionKey, true))

	// Invalid options - empty protocol
	opts = NewReportOptions(WithAddress("127.0.0.1:2181"))
	url, err = opts.toUrl()
	require.Error(t, err)
	assert.Nil(t, url)
}

func TestFromRegistry(t *testing.T) {
	rc := &global.RegistryConfig{
		Protocol:  "zookeeper",
		Address:   "127.0.0.1:2181",
		Username:  "admin",
		Password:  "secret",
		Group:     "dubbo",
		Namespace: "public",
		Timeout:   "3s",
	}
	opts := fromRegistry("zk-registry", rc)
	assert.Equal(t, "zk-registry", opts.registryId)
	assert.Equal(t, "zookeeper", opts.Protocol)
	assert.Equal(t, "3000", opts.Timeout)

	// Invalid timeout
	rc.Timeout = "invalid"
	opts = fromRegistry("test", rc)
	assert.Empty(t, opts.Timeout)
}

func TestInitRegistryMetadataReport(t *testing.T) {
	// Empty/nil registries
	require.NoError(t, InitRegistryMetadataReport(nil))
	require.NoError(t, InitRegistryMetadataReport(map[string]*global.RegistryConfig{}))

	// Invalid UseAsMetaReport
	err := InitRegistryMetadataReport(map[string]*global.RegistryConfig{
		"zk": {Protocol: "zookeeper", Address: "127.0.0.1:2181", UseAsMetaReport: "invalid"},
	})
	require.Error(t, err)
}

func TestOptionsOverride(t *testing.T) {
	opts := NewOptions(WithAppName("app1"), WithAppName("app2"))
	assert.Equal(t, "app2", opts.appName)

	reportOpts := NewReportOptions(WithZookeeper(), WithNacos(), WithEtcdV3())
	assert.Equal(t, constant.EtcdV3Key, reportOpts.Protocol)
}

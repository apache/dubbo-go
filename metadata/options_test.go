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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// ============================================
// Options Tests
// ============================================
func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()
	assert.NotNil(t, opts)
	assert.Equal(t, constant.DefaultMetadataStorageType, opts.metadataType)
	assert.Equal(t, constant.DefaultProtocol, opts.protocol)
	assert.Empty(t, opts.appName)
	assert.Zero(t, opts.port)
}

func TestNewOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected *Options
	}{
		{
			name: "default options",
			opts: nil,
			expected: &Options{
				metadataType: constant.DefaultMetadataStorageType,
				protocol:     constant.DefaultProtocol,
			},
		},
		{
			name: "with app name only",
			opts: []Option{WithAppName("test-app")},
			expected: &Options{
				appName:      "test-app",
				metadataType: constant.DefaultMetadataStorageType,
				protocol:     constant.DefaultProtocol,
			},
		},
		{
			name: "with all options",
			opts: []Option{
				WithAppName("my-app"),
				WithMetadataType("remote"),
				WithPort(20880),
				WithMetadataProtocol("tri"),
			},
			expected: &Options{
				appName:      "my-app",
				metadataType: "remote",
				port:         20880,
				protocol:     "tri",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(tt.opts...)
			assert.Equal(t, tt.expected.appName, opts.appName)
			assert.Equal(t, tt.expected.metadataType, opts.metadataType)
			assert.Equal(t, tt.expected.port, opts.port)
			assert.Equal(t, tt.expected.protocol, opts.protocol)
		})
	}
}

func TestWithAppName(t *testing.T) {
	tests := []struct {
		name     string
		appName  string
		expected string
	}{
		{"normal app name", "my-app", "my-app"},
		{"empty app name", "", ""},
		{"app name with special chars", "my-app_v1.0", "my-app_v1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()
			WithAppName(tt.appName)(opts)
			assert.Equal(t, tt.expected, opts.appName)
		})
	}
}

func TestWithMetadataType(t *testing.T) {
	tests := []struct {
		name         string
		metadataType string
		expected     string
	}{
		{"local type", "local", "local"},
		{"remote type", "remote", "remote"},
		{"empty type", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()
			WithMetadataType(tt.metadataType)(opts)
			assert.Equal(t, tt.expected, opts.metadataType)
		})
	}
}

func TestWithPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected int
	}{
		{"standard port", 20880, 20880},
		{"zero port", 0, 0},
		{"high port", 65535, 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()
			WithPort(tt.port)(opts)
			assert.Equal(t, tt.expected, opts.port)
		})
	}
}

func TestWithMetadataProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		expected string
	}{
		{"dubbo protocol", "dubbo", "dubbo"},
		{"tri protocol", "tri", "tri"},
		{"empty protocol", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()
			WithMetadataProtocol(tt.protocol)(opts)
			assert.Equal(t, tt.expected, opts.protocol)
		})
	}
}

// ============================================
// ReportOptions Tests
// ============================================

func TestDefaultReportOptions(t *testing.T) {
	opts := defaultReportOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.MetadataReportConfig)
	assert.Empty(t, opts.registryId)
	assert.NotNil(t, opts.Params)
}

func TestNewReportOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []ReportOption
		validate func(*testing.T, *ReportOptions)
	}{
		{
			name: "default report options",
			opts: nil,
			validate: func(t *testing.T, opts *ReportOptions) {
				assert.NotNil(t, opts.MetadataReportConfig)
				assert.Empty(t, opts.registryId)
			},
		},
		{
			name: "with all options",
			opts: []ReportOption{
				WithRegistryId("registry-1"),
				WithProtocol("zookeeper"),
				WithAddress("127.0.0.1:2181"),
				WithUsername("admin"),
				WithPassword("secret"),
				WithTimeout(5 * time.Second),
				WithGroup("test-group"),
				WithNamespace("test-ns"),
				WithParams(map[string]string{"key": "value"}),
			},
			validate: func(t *testing.T, opts *ReportOptions) {
				assert.Equal(t, "registry-1", opts.registryId)
				assert.Equal(t, "zookeeper", opts.Protocol)
				assert.Equal(t, "127.0.0.1:2181", opts.Address)
				assert.Equal(t, "admin", opts.Username)
				assert.Equal(t, "secret", opts.Password)
				assert.Equal(t, "5000", opts.Timeout)
				assert.Equal(t, "test-group", opts.Group)
				assert.Equal(t, "test-ns", opts.Namespace)
				assert.Equal(t, "value", opts.Params["key"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewReportOptions(tt.opts...)
			tt.validate(t, opts)
		})
	}
}

func TestProtocolOptions(t *testing.T) {
	tests := []struct {
		name     string
		option   ReportOption
		expected string
	}{
		{"zookeeper", WithZookeeper(), constant.ZookeeperKey},
		{"nacos", WithNacos(), constant.NacosKey},
		{"etcdv3", WithEtcdV3(), constant.EtcdV3Key},
		{"custom protocol", WithProtocol("consul"), "consul"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			tt.option(opts)
			assert.Equal(t, tt.expected, opts.Protocol)
		})
	}
}

func TestWithAddress(t *testing.T) {
	tests := []struct {
		name             string
		address          string
		expectedAddress  string
		expectedProtocol string
	}{
		{
			name:             "address without protocol",
			address:          "127.0.0.1:2181",
			expectedAddress:  "127.0.0.1:2181",
			expectedProtocol: "",
		},
		{
			name:             "zookeeper address with protocol",
			address:          "zookeeper://127.0.0.1:2181",
			expectedAddress:  "zookeeper://127.0.0.1:2181",
			expectedProtocol: "zookeeper",
		},
		{
			name:             "nacos address with protocol",
			address:          "nacos://localhost:8848",
			expectedAddress:  "nacos://localhost:8848",
			expectedProtocol: "nacos",
		},
		{
			name:             "etcd address with protocol",
			address:          "etcd://localhost:2379",
			expectedAddress:  "etcd://localhost:2379",
			expectedProtocol: "etcd",
		},
		{
			name:             "empty address",
			address:          "",
			expectedAddress:  "",
			expectedProtocol: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			WithAddress(tt.address)(opts)
			assert.Equal(t, tt.expectedAddress, opts.Address)
			assert.Equal(t, tt.expectedProtocol, opts.Protocol)
		})
	}
}

func TestWithCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
	}{
		{"normal credentials", "admin", "secret"},
		{"empty credentials", "", ""},
		{"special chars", "user@domain", "p@ss!word#123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			WithUsername(tt.username)(opts)
			WithPassword(tt.password)(opts)
			assert.Equal(t, tt.username, opts.Username)
			assert.Equal(t, tt.password, opts.Password)
		})
	}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected string
	}{
		{"1 second", 1 * time.Second, "1000"},
		{"5 seconds", 5 * time.Second, "5000"},
		{"500 milliseconds", 500 * time.Millisecond, "500"},
		{"zero timeout", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			WithTimeout(tt.timeout)(opts)
			assert.Equal(t, tt.expected, opts.Timeout)
		})
	}
}

func TestWithGroup(t *testing.T) {
	tests := []struct {
		name     string
		group    string
		expected string
	}{
		{"normal group", "my-group", "my-group"},
		{"empty group", "", ""},
		{"group with slash", "dubbo/group", "dubbo/group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			WithGroup(tt.group)(opts)
			assert.Equal(t, tt.expected, opts.Group)
		})
	}
}

func TestWithNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{"normal namespace", "my-namespace", "my-namespace"},
		{"empty namespace", "", ""},
		{"public namespace", "public", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			WithNamespace(tt.namespace)(opts)
			assert.Equal(t, tt.expected, opts.Namespace)
		})
	}
}

func TestWithParams(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
	}{
		{"single param", map[string]string{"key": "value"}},
		{"multiple params", map[string]string{"k1": "v1", "k2": "v2"}},
		{"empty params", map[string]string{}},
		{"nil params", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			WithParams(tt.params)(opts)
			assert.Equal(t, tt.params, opts.Params)
		})
	}
}

func TestWithRegistryId(t *testing.T) {
	tests := []struct {
		name       string
		registryId string
		expected   string
	}{
		{"normal id", "registry-1", "registry-1"},
		{"empty id", "", ""},
		{"id with numbers", "registry123", "registry123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultReportOptions()
			WithRegistryId(tt.registryId)(opts)
			assert.Equal(t, tt.expected, opts.registryId)
		})
	}
}

// ============================================
// ReportOptions.toUrl Tests
// ============================================

func TestReportOptionsToUrl(t *testing.T) {
	tests := []struct {
		name     string
		opts     *ReportOptions
		wantErr  bool
		errMsg   string
		validate func(*testing.T, *ReportOptions)
	}{
		{
			name: "valid zookeeper options",
			opts: NewReportOptions(
				WithProtocol("zookeeper"),
				WithAddress("127.0.0.1:2181"),
				WithUsername("admin"),
				WithPassword("secret"),
				WithTimeout(5*time.Second),
				WithGroup("test-group"),
				WithNamespace("test-ns"),
				WithParams(map[string]string{"custom": "param"}),
			),
			wantErr: false,
			validate: func(t *testing.T, opts *ReportOptions) {
				url, err := opts.toUrl()
				assert.Nil(t, err)
				assert.NotNil(t, url)
				assert.Equal(t, "zookeeper", url.Protocol)
				assert.Equal(t, "admin", url.Username)
				assert.Equal(t, "secret", url.Password)
				assert.Equal(t, "5000", url.GetParam(constant.TimeoutKey, ""))
				assert.Equal(t, "test-group", url.GetParam(constant.MetadataReportGroupKey, ""))
				assert.Equal(t, "test-ns", url.GetParam(constant.MetadataReportNamespaceKey, ""))
				assert.Equal(t, "zookeeper", url.GetParam("metadata", ""))
				assert.Equal(t, "param", url.GetParam("custom", ""))
			},
		},
		{
			name: "valid nacos options",
			opts: NewReportOptions(
				WithNacos(),
				WithAddress("localhost:8848"),
			),
			wantErr: false,
			validate: func(t *testing.T, opts *ReportOptions) {
				url, err := opts.toUrl()
				assert.Nil(t, err)
				assert.Equal(t, "nacos", url.Protocol)
				assert.Equal(t, "nacos", url.GetParam("metadata", ""))
			},
		},
		{
			name: "invalid options - empty protocol",
			opts: NewReportOptions(
				WithAddress("127.0.0.1:2181"),
			),
			wantErr: true,
			errMsg:  "Invalid MetadataReport Config.",
		},
		{
			name:    "invalid options - empty address and protocol",
			opts:    NewReportOptions(),
			wantErr: true,
			errMsg:  "Invalid MetadataReport Config.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := tt.opts.toUrl()
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, url)
				if tt.errMsg != "" {
					assert.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, url)
				if tt.validate != nil {
					tt.validate(t, tt.opts)
				}
			}
		})
	}
}

// ============================================
// fromRegistry Tests
// ============================================

func TestFromRegistry(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		rc       *global.RegistryConfig
		validate func(*testing.T, *ReportOptions)
	}{
		{
			name: "basic registry config",
			id:   "zk-registry",
			rc: &global.RegistryConfig{
				Protocol:  "zookeeper",
				Address:   "127.0.0.1:2181",
				Username:  "admin",
				Password:  "secret",
				Group:     "dubbo",
				Namespace: "public",
				Params:    map[string]string{"key": "value"},
			},
			validate: func(t *testing.T, opts *ReportOptions) {
				assert.Equal(t, "zk-registry", opts.registryId)
				assert.Equal(t, "zookeeper", opts.Protocol)
				assert.Equal(t, "127.0.0.1:2181", opts.Address)
				assert.Equal(t, "admin", opts.Username)
				assert.Equal(t, "secret", opts.Password)
				assert.Equal(t, "dubbo", opts.Group)
				assert.Equal(t, "public", opts.Namespace)
				assert.Equal(t, "value", opts.Params["key"])
			},
		},
		{
			name: "registry config with timeout",
			id:   "nacos-registry",
			rc: &global.RegistryConfig{
				Protocol: "nacos",
				Address:  "localhost:8848",
				Timeout:  "3s",
			},
			validate: func(t *testing.T, opts *ReportOptions) {
				assert.Equal(t, "nacos-registry", opts.registryId)
				assert.Equal(t, "nacos", opts.Protocol)
				assert.Equal(t, "localhost:8848", opts.Address)
				assert.Equal(t, "3000", opts.Timeout)
			},
		},
		{
			name: "registry config with invalid timeout",
			id:   "test-registry",
			rc: &global.RegistryConfig{
				Protocol: "zookeeper",
				Address:  "127.0.0.1:2181",
				Timeout:  "invalid",
			},
			validate: func(t *testing.T, opts *ReportOptions) {
				assert.Equal(t, "test-registry", opts.registryId)
				assert.Equal(t, "zookeeper", opts.Protocol)
				// timeout should remain empty due to parse error
				assert.Empty(t, opts.Timeout)
			},
		},
		{
			name: "registry config with nil params",
			id:   "test-registry",
			rc: &global.RegistryConfig{
				Protocol: "etcd",
				Address:  "localhost:2379",
				Params:   nil,
			},
			validate: func(t *testing.T, opts *ReportOptions) {
				assert.Equal(t, "etcd", opts.Protocol)
				assert.Nil(t, opts.Params)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := fromRegistry(tt.id, tt.rc)
			assert.NotNil(t, opts)
			tt.validate(t, opts)
		})
	}
}

// ============================================
// InitRegistryMetadataReport Tests
// ============================================

func TestInitRegistryMetadataReport(t *testing.T) {
	tests := []struct {
		name       string
		registries map[string]*global.RegistryConfig
		wantErr    bool
	}{
		{
			name:       "empty registries",
			registries: map[string]*global.RegistryConfig{},
			wantErr:    false,
		},
		{
			name:       "nil registries",
			registries: nil,
			wantErr:    false,
		},
		{
			name: "registry with UseAsMetaReport false",
			registries: map[string]*global.RegistryConfig{
				"zk": {
					Protocol:        "zookeeper",
					Address:         "127.0.0.1:2181",
					UseAsMetaReport: "false",
				},
			},
			wantErr: false,
		},
		{
			name: "registry with invalid UseAsMetaReport",
			registries: map[string]*global.RegistryConfig{
				"zk": {
					Protocol:        "zookeeper",
					Address:         "127.0.0.1:2181",
					UseAsMetaReport: "invalid_bool",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitRegistryMetadataReport(tt.registries)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// ============================================
// Options Chaining Tests
// ============================================

func TestOptionsChaining(t *testing.T) {
	t.Run("options can be chained", func(t *testing.T) {
		opts := NewOptions(
			WithAppName("app1"),
			WithMetadataType("local"),
		)
		// Apply more options
		WithPort(8080)(opts)
		WithMetadataProtocol("dubbo")(opts)

		assert.Equal(t, "app1", opts.appName)
		assert.Equal(t, "local", opts.metadataType)
		assert.Equal(t, 8080, opts.port)
		assert.Equal(t, "dubbo", opts.protocol)
	})

	t.Run("report options can be chained", func(t *testing.T) {
		opts := NewReportOptions(
			WithZookeeper(),
			WithAddress("127.0.0.1:2181"),
		)
		// Apply more options
		WithUsername("admin")(opts)
		WithPassword("secret")(opts)
		WithTimeout(10 * time.Second)(opts)

		assert.Equal(t, constant.ZookeeperKey, opts.Protocol)
		assert.Equal(t, "127.0.0.1:2181", opts.Address)
		assert.Equal(t, "admin", opts.Username)
		assert.Equal(t, "secret", opts.Password)
		assert.Equal(t, "10000", opts.Timeout)
	})
}

// ============================================
// Options Override Tests
// ============================================

func TestOptionsOverride(t *testing.T) {
	t.Run("later options override earlier ones", func(t *testing.T) {
		opts := NewOptions(
			WithAppName("app1"),
			WithAppName("app2"),
		)
		assert.Equal(t, "app2", opts.appName)
	})

	t.Run("report options override", func(t *testing.T) {
		opts := NewReportOptions(
			WithZookeeper(),
			WithNacos(),
			WithEtcdV3(),
		)
		assert.Equal(t, constant.EtcdV3Key, opts.Protocol)
	})
}

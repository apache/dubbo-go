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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"

	"github.com/stretchr/testify/assert"
)

func TestNewOptions(t *testing.T) {
	// Test default options
	opts := NewOptions()
	assert.Equal(t, constant.DefaultMetadataStorageType, opts.metadataType)
	assert.Equal(t, constant.DefaultProtocol, opts.protocol)

	// Test with all options
	opts = NewOptions(
		WithAppName("my-app"),
		WithMetadataType("remote"),
		WithPort(20880),
		WithMetadataProtocol("tri"),
	)
	assert.Equal(t, "my-app", opts.appName)
	assert.Equal(t, "remote", opts.metadataType)
	assert.Equal(t, 20880, opts.port)
	assert.Equal(t, "tri", opts.protocol)
}

func TestNewReportOptions(t *testing.T) {
	// Test default
	opts := NewReportOptions()
	assert.NotNil(t, opts.MetadataReportConfig)

	// Test with all options
	opts = NewReportOptions(
		WithRegistryId("registry-1"),
		WithZookeeper(),
		WithAddress("127.0.0.1:2181"),
		WithUsername("admin"),
		WithPassword("secret"),
		WithTimeout(5*time.Second),
		WithGroup("test-group"),
		WithNamespace("test-ns"),
		WithParams(map[string]string{"key": "value"}),
	)
	assert.Equal(t, "registry-1", opts.registryId)
	assert.Equal(t, constant.ZookeeperKey, opts.Protocol)
	assert.Equal(t, "127.0.0.1:2181", opts.Address)
	assert.Equal(t, "admin", opts.Username)
	assert.Equal(t, "secret", opts.Password)
	assert.Equal(t, "5000", opts.Timeout)
	assert.Equal(t, "test-group", opts.Group)
	assert.Equal(t, "test-ns", opts.Namespace)
	assert.Equal(t, "value", opts.Params["key"])
}

func TestProtocolOptions(t *testing.T) {
	tests := []struct {
		option   ReportOption
		expected string
	}{
		{WithZookeeper(), constant.ZookeeperKey},
		{WithNacos(), constant.NacosKey},
		{WithEtcdV3(), constant.EtcdV3Key},
	}
	for _, tt := range tests {
		opts := defaultReportOptions()
		tt.option(opts)
		assert.Equal(t, tt.expected, opts.Protocol)
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
	opts := NewReportOptions(WithZookeeper(), WithAddress("127.0.0.1:2181"))
	url, err := opts.toUrl()
	assert.Nil(t, err)
	assert.Equal(t, "zookeeper", url.Protocol)

	// Invalid options - empty protocol
	opts = NewReportOptions(WithAddress("127.0.0.1:2181"))
	url, err = opts.toUrl()
	assert.NotNil(t, err)
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
	assert.Nil(t, InitRegistryMetadataReport(nil))
	assert.Nil(t, InitRegistryMetadataReport(map[string]*global.RegistryConfig{}))

	// Invalid UseAsMetaReport
	err := InitRegistryMetadataReport(map[string]*global.RegistryConfig{
		"zk": {Protocol: "zookeeper", Address: "127.0.0.1:2181", UseAsMetaReport: "invalid"},
	})
	assert.NotNil(t, err)
}

func TestOptionsOverride(t *testing.T) {
	opts := NewOptions(WithAppName("app1"), WithAppName("app2"))
	assert.Equal(t, "app2", opts.appName)

	reportOpts := NewReportOptions(WithZookeeper(), WithNacos(), WithEtcdV3())
	assert.Equal(t, constant.EtcdV3Key, reportOpts.Protocol)
}

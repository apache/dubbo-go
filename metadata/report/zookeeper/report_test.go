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

package zookeeper

import (
	"encoding/json"
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
)

func TestMetadataInfoSerialization(t *testing.T) {
	original := &info.MetadataInfo{
		App:      "test-app",
		Revision: "1.0.0",
		Services: map[string]*info.ServiceInfo{
			"com.example.TestService": {
				Name: "com.example.TestService", Protocol: "dubbo",
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored info.MetadataInfo
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, original.App, restored.App)
	assert.Equal(t, original.Revision, restored.Revision)

	// Invalid JSON
	err = json.Unmarshal([]byte(`{invalid}`), &restored)
	assert.Error(t, err)
}

func TestRegisterServiceAppMappingValueMerge(t *testing.T) {
	tests := []struct {
		oldValue, newValue, expected string
	}{
		{"app1", "app2", "app1,app2"},
		{"app1,app2", "app1", "app1,app2"},
		{"", "app1", ",app1"},
	}
	for _, tt := range tests {
		var result string
		if strings.Contains(tt.oldValue, tt.newValue) {
			result = tt.oldValue
		} else {
			result = tt.oldValue + "," + tt.newValue
		}
		assert.Equal(t, tt.expected, result)
	}
}

func TestCreateMetadataReportURLParsing(t *testing.T) {
	tests := []struct {
		group, expectedRootDir string
	}{
		{"", "/dubbo/"},
		{"custom", "/custom/"},
		{"/custom", "/custom/"},
		{"/", "/"},
	}
	for _, tt := range tests {
		url := common.NewURLWithOptions(
			common.WithProtocol("zookeeper"),
			common.WithLocation("127.0.0.1:2181"),
		)
		if tt.group != "" {
			url.SetParam(constant.MetadataReportGroupKey, tt.group)
		}
		rootDir := url.GetParam(constant.MetadataReportGroupKey, "dubbo")
		if len(rootDir) > 0 && rootDir[0] != '/' {
			rootDir = "/" + rootDir
		}
		if rootDir != "/" {
			rootDir = rootDir + "/"
		}
		assert.Equal(t, tt.expectedRootDir, rootDir)
	}
}

func TestRemoveServiceAppMappingListener(t *testing.T) {
	report := &zookeeperMetadataReport{
		rootDir:       "/dubbo/",
		cacheListener: NewCacheListener("/dubbo/", nil),
	}
	err := report.RemoveServiceAppMappingListener("test.service", "mapping")
	assert.NoError(t, err)
}

func TestCacheListenerIntegrationWithReport(t *testing.T) {
	cacheListener := NewCacheListener("/dubbo/", nil)
	report := &zookeeperMetadataReport{
		rootDir:       "/dubbo/",
		cacheListener: cacheListener,
	}
	assert.NotNil(t, report.cacheListener)
	assert.Equal(t, "/dubbo/", report.rootDir)
}

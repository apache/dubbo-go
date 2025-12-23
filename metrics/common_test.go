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

package metrics

import (
	"os"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestMain(m *testing.M) {
	InitAppInfo("test-app", "1.0.0")
	os.Exit(m.Run())
}

func TestGetApplicationLevel(t *testing.T) {
	level := GetApplicationLevel()

	assert.NotNil(t, level)
	assert.Equal(t, "test-app", level.ApplicationName)
	assert.Equal(t, "1.0.0", level.Version)
	assert.NotEmpty(t, level.Ip)
	assert.NotEmpty(t, level.HostName)
}

func TestApplicationMetricLevelTags(t *testing.T) {
	level := GetApplicationLevel()
	tags := level.Tags()

	assert.NotNil(t, tags)
	assert.Equal(t, "test-app", tags[constant.TagApplicationName])
	assert.Equal(t, "1.0.0", tags[constant.TagApplicationVersion])
	assert.NotEmpty(t, tags[constant.TagIp])
	assert.NotEmpty(t, tags[constant.TagHostname])
	assert.Equal(t, "", tags[constant.TagGitCommitId])
}

func TestServiceMetricLevelTags(t *testing.T) {
	serviceMetric := NewServiceMetric("com.example.Service")
	tags := serviceMetric.Tags()

	assert.NotNil(t, tags)
	assert.Equal(t, "test-app", tags[constant.TagApplicationName])
	assert.Equal(t, "1.0.0", tags[constant.TagApplicationVersion])
	assert.Equal(t, "com.example.Service", tags[constant.TagInterface])
	assert.NotEmpty(t, tags[constant.TagIp])
	assert.NotEmpty(t, tags[constant.TagHostname])
}

func TestMethodMetricLevelTags(t *testing.T) {
	serviceMetric := NewServiceMetric("com.example.Service")
	methodLevel := MethodMetricLevel{
		ServiceMetricLevel: serviceMetric,
		Method:             "TestMethod",
		Group:              "test-group",
		Version:            "1.0.0",
	}

	tags := methodLevel.Tags()

	assert.NotNil(t, tags)
	assert.Equal(t, "test-app", tags[constant.TagApplicationName])
	assert.Equal(t, "1.0.0", tags[constant.TagApplicationVersion])
	assert.Equal(t, "com.example.Service", tags[constant.TagInterface])
	assert.Equal(t, "TestMethod", tags[constant.TagMethod])
	assert.Equal(t, "test-group", tags[constant.TagGroup])
	assert.Equal(t, "1.0.0", tags[constant.TagVersion])
}

func TestConfigCenterLevelTags(t *testing.T) {
	level := NewConfigCenterLevel("test-key", "test-group", "nacos", "added")
	tags := level.Tags()

	assert.NotNil(t, tags)
	assert.Equal(t, "test-app", tags[constant.TagApplicationName])
	assert.Equal(t, "test-key", tags[constant.TagKey])
	assert.Equal(t, "test-group", tags[constant.TagGroup])
	assert.Equal(t, "nacos", tags[constant.TagConfigCenter])
	assert.Equal(t, "added", tags[constant.TagChangeType])
	assert.NotEmpty(t, tags[constant.TagIp])
	assert.NotEmpty(t, tags[constant.TagHostname])
}

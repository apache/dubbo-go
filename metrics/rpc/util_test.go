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

package rpc

import (
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestBuildLabels(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
		common.WithParamsValue(constant.AppVersionKey, "1.0.0"),
		common.WithParamsValue(constant.VersionKey, "2.0.0"),
		common.WithParamsValue(constant.GroupKey, "test-group"),
		common.WithParamsValue(constant.InterfaceKey, "com.example.Service"),
	)

	invoc := invocation.NewRPCInvocation("TestMethod", []any{}, nil)

	labels := buildLabels(url, invoc)

	assert.NotNil(t, labels)
	assert.Equal(t, "test-app", labels[constant.TagApplicationName])
	assert.Equal(t, "1.0.0", labels[constant.TagApplicationVersion])
	assert.Equal(t, "com.example.Service", labels[constant.TagInterface])
	assert.Equal(t, "TestMethod", labels[constant.TagMethod])
	assert.Equal(t, "test-group", labels[constant.TagGroup])
	assert.Equal(t, "2.0.0", labels[constant.TagVersion])
	assert.NotEmpty(t, labels[constant.TagHostname])
	assert.NotEmpty(t, labels[constant.TagIp])
}

func TestGetRole(t *testing.T) {
	tests := []struct {
		name     string
		url      *common.URL
		expected string
	}{
		{
			name: "provider role",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)),
			),
			expected: constant.SideProvider,
		},
		{
			name: "consumer role",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
			),
			expected: constant.SideConsumer,
		},
		{
			name: "no role",
			url: common.NewURLWithOptions(
				common.WithProtocol("dubbo"),
			),
			expected: "",
		},
		{
			name: "invalid role",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, "invalid"),
			),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := getRole(tt.url)
			assert.Equal(t, tt.expected, role)
		})
	}
}

func TestIsProvider(t *testing.T) {
	tests := []struct {
		name     string
		url      *common.URL
		expected bool
	}{
		{
			name: "provider",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)),
			),
			expected: true,
		},
		{
			name: "provider string",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, "3"),
			),
			expected: true,
		},
		{
			name: "not provider",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
			),
			expected: false,
		},
		{
			name: "no role",
			url: common.NewURLWithOptions(
				common.WithProtocol("dubbo"),
			),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isProvider(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConsumer(t *testing.T) {
	tests := []struct {
		name     string
		url      *common.URL
		expected bool
	}{
		{
			name: "consumer",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
			),
			expected: true,
		},
		{
			name: "consumer string",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, "0"),
			),
			expected: true,
		},
		{
			name: "not consumer",
			url: common.NewURLWithOptions(
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)),
			),
			expected: false,
		},
		{
			name: "no role",
			url: common.NewURLWithOptions(
				common.WithProtocol("dubbo"),
			),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConsumer(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

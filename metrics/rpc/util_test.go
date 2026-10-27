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
	"dubbo.apache.org/dubbo-go/v3/global"
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
	assert.Equal(t, url.Service(), labels[constant.TagInterface])
	assert.NotEqual(t, url.ServiceKey(), labels[constant.TagInterface])
	assert.Equal(t, "test-group/com.example.Service:2.0.0", url.ServiceKey())
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

// A discovered service retains the provider application param. The local
// reference carries this consumer instance's application as an attribute.
func TestBuildLabelsConsumerDiscoveredURL(t *testing.T) {
	provider := common.NewURLWithOptions(
		common.WithProtocol("tri"),
		common.WithPath("com.example.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "provider-app"),
		common.WithParamsValue(constant.AppVersionKey, "provider-v1"),
		common.WithParamsValue(constant.GroupKey, "sample-group"),
		common.WithParamsValue(constant.VersionKey, "service-v1"),
	)
	for _, application := range []*global.ApplicationConfig{
		{Name: "consumer-one", Version: "consumer-v1"},
		{Name: "consumer-two", Version: "consumer-v2"},
	} {
		t.Run(application.Name, func(t *testing.T) {
			reference := common.NewURLWithOptions(
				common.WithProtocol("tri"),
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
				common.WithParamsValue(constant.ApplicationKey, application.Name),
				common.WithParamsValue(constant.AppVersionKey, application.Version),
				common.WithAttribute(constant.ApplicationKey, application),
			)
			discovered := provider.MergeURL(reference)
			labels := buildLabels(discovered, invocation.NewRPCInvocation("Greet", nil, nil))
			assert.Equal(t, application.Name, labels[constant.TagApplicationName])
			assert.Equal(t, application.Version, labels[constant.TagApplicationVersion])
			assert.Equal(t, "provider-app", discovered.GetParam(constant.ApplicationKey, ""))
			assert.Equal(t, "provider-v1", discovered.GetParam(constant.AppVersionKey, ""))
			assert.Equal(t, "com.example.GreetService", labels[constant.TagInterface])
			assert.Equal(t, "sample-group", labels[constant.TagGroup])
			assert.Equal(t, "service-v1", labels[constant.TagVersion])
		})
	}
	assert.Equal(t, "provider-app", provider.GetParam(constant.ApplicationKey, ""))
}

func TestBuildLabelsApplicationAttributeFallback(t *testing.T) {
	var nilApplication *global.ApplicationConfig
	tests := []struct {
		name        string
		role        common.RoleType
		attribute   any
		wantName    string
		wantVersion string
	}{
		{"missing", common.CONSUMER, nil, "param-app", "param-version"},
		{"typed nil", common.CONSUMER, nilApplication, "param-app", "param-version"},
		{"unexpected type", common.CONSUMER, 42, "param-app", "param-version"},
		{"empty name", common.CONSUMER, &global.ApplicationConfig{Version: "local-version"}, "param-app", "param-version"},
		{"pointer", common.CONSUMER, &global.ApplicationConfig{Name: "local-app", Version: "local-version"}, "local-app", "local-version"},
		{"value", common.CONSUMER, global.ApplicationConfig{Name: "local-app", Version: "local-version"}, "local-app", "local-version"},
		{"empty local version", common.CONSUMER, &global.ApplicationConfig{Name: "local-app"}, "local-app", ""},
		{"provider", common.PROVIDER, &global.ApplicationConfig{Name: "attribute-app", Version: "attribute-version"}, "param-app", "param-version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithParamsValue(constant.ApplicationKey, "param-app"),
				common.WithParamsValue(constant.AppVersionKey, "param-version"),
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(int(tt.role))),
			)
			if tt.attribute != nil {
				url.SetAttribute(constant.ApplicationKey, tt.attribute)
			}
			labels := buildLabels(url, invocation.NewRPCInvocation("Greet", nil, nil))
			assert.Equal(t, tt.wantName, labels[constant.TagApplicationName])
			assert.Equal(t, tt.wantVersion, labels[constant.TagApplicationVersion])
		})
	}
}

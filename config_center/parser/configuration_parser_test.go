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

package parser

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestDefaultConfigurationParserParser(t *testing.T) {
	parser := &DefaultConfigurationParser{}
	m, err := parser.Parse("dubbo.registry.address=172.0.0.1\ndubbo.registry.name=test")
	require.NoError(t, err)
	assert.Len(t, m, 2)
	assert.Equal(t, "172.0.0.1", m["dubbo.registry.address"])
}

func TestDefaultConfigurationParserAppItemToUrls_ParserToUrls(t *testing.T) {
	parser := &DefaultConfigurationParser{}
	content := `configVersion: 2.7.1
scope: application
key: org.apache.dubbo-go.mockService
enabled: true
configs:
- type: application
  enabled: true
  addresses:
  - 0.0.0.0
  providerAddresses: []
  services:
  - org.apache.dubbo-go.mockService
  applications: []
  parameters:
    cluster: mock1
  side: provider`
	urls, err := parser.ParseToUrls(content)
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "org.apache.dubbo-go.mockService", urls[0].GetParam("application", ""))
	assert.Equal(t, "mock1", urls[0].GetParam("cluster", ""))
	assert.Equal(t, "override", urls[0].Protocol)
	assert.Equal(t, "0.0.0.0", urls[0].Location)
}

func TestDefaultConfigurationParserAppScopeDefaults(t *testing.T) {
	parser := &DefaultConfigurationParser{}
	content := `configVersion: 3.0.0
scope: application
key: app-key
enabled: true
configs:
- type: custom
  enabled: false
  addresses: []
  providerAddresses: []
  services: []
  applications: []
  parameters:
    mock: v
  side: consumer`
	urls, err := parser.ParseToUrls(content)
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "override", urls[0].Protocol)
	assert.Equal(t, "0.0.0.0", urls[0].Location)
	assert.Equal(t, "*", urls[0].Service())
	assert.Equal(t, "app-key", urls[0].GetParam("application", ""))
	assert.Equal(t, "dynamicconfigurators", urls[0].GetParam("category", ""))
	assert.Equal(t, "3.0.0", urls[0].GetParam(constant.RuleConfigVersionKey, ""))
	assert.Equal(t, "false", urls[0].GetParam("enabled", ""))
}

func TestConditionMatchIsMatch(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20880/org.apache.dubbo.quickstart.GreeterDynamic?application=demo-provider&env=prod&group=demo&version=1.0.0")
	require.NoError(t, err)

	const host = "10.0.0.1"
	listMatch := func(value string) *common.ListStringMatch {
		return &common.ListStringMatch{Oneof: []common.StringMatch{{Exact: value}}}
	}
	paramMatch := func(value string) []*common.ParamMatch {
		return []*common.ParamMatch{{Key: "env", Value: common.StringMatch{Exact: value}}}
	}

	tests := []struct {
		name  string
		match *ConditionMatch
		want  bool
	}{
		{name: "nil matches all", match: nil, want: true},
		{name: "empty matches all", match: &ConditionMatch{}, want: true},
		{name: "address matches", match: &ConditionMatch{Address: &common.AddressMatch{Exact: host}}, want: true},
		{name: "address rejects", match: &ConditionMatch{Address: &common.AddressMatch{Exact: "10.0.0.2"}}, want: false},
		{name: "provider address matches", match: &ConditionMatch{ProviderAddress: &common.AddressMatch{Exact: url.Location}}, want: true},
		{name: "provider address rejects", match: &ConditionMatch{ProviderAddress: &common.AddressMatch{Exact: "127.0.0.2:20880"}}, want: false},
		{name: "service matches", match: &ConditionMatch{Service: listMatch(url.ServiceKey())}, want: true},
		{name: "service rejects", match: &ConditionMatch{Service: listMatch("demo/another.Service:1.0.0")}, want: false},
		{name: "application matches", match: &ConditionMatch{App: listMatch("demo-provider")}, want: true},
		{name: "application rejects", match: &ConditionMatch{App: listMatch("another-provider")}, want: false},
		{name: "parameter matches", match: &ConditionMatch{Param: paramMatch("prod")}, want: true},
		{name: "parameter rejects", match: &ConditionMatch{Param: paramMatch("staging")}, want: false},
		{
			name: "all configured dimensions match",
			match: &ConditionMatch{
				Address:         &common.AddressMatch{Exact: host},
				ProviderAddress: &common.AddressMatch{Exact: url.Location},
				Service:         listMatch(url.ServiceKey()),
				App:             listMatch("demo-provider"),
				Param:           paramMatch("prod"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.match.IsMatch(host, url))
		})
	}
}

func TestDefaultConfigurationParserServiceItemToUrls_ParserToUrls(t *testing.T) {
	parser := &DefaultConfigurationParser{}
	content := `configVersion: 2.7.1
scope: notApplication
key: groupA/test:1
enabled: true
configs:
- type: application
  enabled: true
  addresses:
  - 0.0.0.0
  providerAddresses: []
  services:
  - org.apache.dubbo-go.mockService
  applications: []
  parameters:
    cluster: mock1
  side: provider`
	urls, err := parser.ParseToUrls(content)
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "groupA", urls[0].GetParam("group", ""))
	assert.Equal(t, "/test", urls[0].Path)
	assert.Equal(t, "mock1", urls[0].GetParam("cluster", ""))
	assert.Equal(t, "override", urls[0].Protocol)
	assert.Equal(t, "0.0.0.0", urls[0].Location)
}

func TestGetEnabledString(t *testing.T) {
	item := ConfigItem{Enabled: false}
	cfg := ConfiguratorConfig{Enabled: true}
	// when type empty/general use config.enabled
	assert.Equal(t, "&enabled=true", getEnabledString(item, cfg))

	item.Type = "custom"
	item.Enabled = false
	assert.Equal(t, "&enabled=false", getEnabledString(item, cfg))
}

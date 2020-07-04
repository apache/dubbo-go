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
)

func TestDefaultConfigurationParserParser(t *testing.T) {
	parser := &DefaultConfigurationParser{}
	m, err := parser.Parse("dubbo.registry.address=172.0.0.1\ndubbo.registry.name=test")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(m))
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
	assert.NoError(t, err)
	assert.Equal(t, 1, len(urls))
	assert.Equal(t, "org.apache.dubbo-go.mockService", urls[0].GetParam("application", ""))
	assert.Equal(t, "mock1", urls[0].GetParam("cluster", ""))
	assert.Equal(t, "override", urls[0].Protocol)
	assert.Equal(t, "0.0.0.0", urls[0].Location)
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
	assert.NoError(t, err)
	assert.Equal(t, 1, len(urls))
	assert.Equal(t, "groupA", urls[0].GetParam("group", ""))
	assert.Equal(t, "/test", urls[0].Path)
	assert.Equal(t, "mock1", urls[0].GetParam("cluster", ""))
	assert.Equal(t, "override", urls[0].Protocol)
	assert.Equal(t, "0.0.0.0", urls[0].Location)
}

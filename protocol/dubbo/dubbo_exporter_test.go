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

package dubbo

import (
	"context"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// MockInvoker is a simple mock implementation of base.Invoker for testing
type MockInvoker struct {
	url       *common.URL
	available bool
	destroyed bool
}

func (m *MockInvoker) GetURL() *common.URL {
	return m.url
}

func (m *MockInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	return &result.RPCResult{}
}

func (m *MockInvoker) IsAvailable() bool {
	return m.available && !m.destroyed
}

func (m *MockInvoker) Destroy() {
	m.destroyed = true
}

// TestNewDubboExporter tests the NewDubboExporter function
func TestNewDubboExporter(t *testing.T) {
	tests := []struct {
		desc              string
		key               string
		invoker           base.Invoker
		expectedNotNil    bool
		expectedAvailable bool
	}{
		{
			desc: "with valid invoker",
			key:  "test_key",
			invoker: &MockInvoker{
				url: func() *common.URL {
					url, _ := common.NewURL("dubbo://localhost:20000/test.TestService?interface=test.TestService&group=test&version=1.0")
					return url
				}(),
				available: true,
				destroyed: false,
			},
			expectedNotNil:    true,
			expectedAvailable: true,
		},
		{
			desc:              "with nil invoker",
			key:               "test_key",
			invoker:           nil,
			expectedNotNil:    true,
			expectedAvailable: false,
		},
		{
			desc: "with different key",
			key:  "another_key",
			invoker: &MockInvoker{
				url: func() *common.URL {
					url, _ := common.NewURL("dubbo://localhost:20000/service?interface=test.Service")
					return url
				}(),
				available: true,
				destroyed: false,
			},
			expectedNotNil:    true,
			expectedAvailable: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			exporterMap := &sync.Map{}
			exporter := NewDubboExporter(test.key, test.invoker, exporterMap)

			assert.NotNil(t, exporter)
			if test.invoker != nil {
				assert.Equal(t, test.invoker, exporter.GetInvoker())
				assert.Equal(t, test.expectedAvailable, exporter.GetInvoker().IsAvailable())
			} else {
				assert.Nil(t, exporter.GetInvoker())
			}
		})
	}
}

// TestDubboExporterGetInvoker tests the GetInvoker method
func TestDubboExporterGetInvoker(t *testing.T) {
	tests := []struct {
		desc      string
		key       string
		urlString string
		checkURL  bool
	}{
		{
			desc:      "get invoker from exporter",
			key:       "test_key",
			urlString: "dubbo://localhost:20000/test.TestService?interface=test.TestService",
			checkURL:  true,
		},
		{
			desc:      "get invoker with group and version",
			key:       "service_key",
			urlString: "dubbo://localhost:20000/service?interface=com.example.Service&group=test&version=1.0",
			checkURL:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			exporterMap := &sync.Map{}
			url, _ := common.NewURL(test.urlString)
			mockInvoker := &MockInvoker{
				url:       url,
				available: true,
				destroyed: false,
			}

			exporter := NewDubboExporter(test.key, mockInvoker, exporterMap)
			retrievedInvoker := exporter.GetInvoker()

			assert.Equal(t, mockInvoker, retrievedInvoker)
			if test.checkURL {
				assert.Equal(t, url, retrievedInvoker.GetURL())
			}
		})
	}
}

// TestDubboExporterMultipleExporters tests creating multiple exporters with different keys coexisting in the same map
func TestDubboExporterMultipleExporters(t *testing.T) {
	tests := []struct {
		desc          string
		key           string
		interfaceName string
	}{
		{
			desc:          "service1",
			key:           "com.example.Service1_key",
			interfaceName: "com.example.Service1",
		},
		{
			desc:          "service2",
			key:           "com.example.Service2_key",
			interfaceName: "com.example.Service2",
		},
		{
			desc:          "test service",
			key:           "test.Service_key",
			interfaceName: "test.Service",
		},
	}

	// Create a shared exporterMap for all test cases
	exporterMap := &sync.Map{}
	exporters := make(map[string]*DubboExporter)
	mockInvokers := make(map[string]*MockInvoker)

	// Create multiple exporters with different keys in the shared map
	for _, test := range tests {
		url, _ := common.NewURL("dubbo://localhost:20000/" + test.interfaceName + "?interface=" + test.interfaceName)
		mockInvoker := &MockInvoker{
			url:       url,
			available: true,
			destroyed: false,
		}
		mockInvokers[test.key] = mockInvoker

		exporter := NewDubboExporter(test.key, mockInvoker, exporterMap)
		exporters[test.key] = exporter
	}

	// Verify all exporters coexist in memory - validate the count
	assert.Len(t, exporters, len(tests), "all exporters should coexist in the exporters map")

	// Verify each exporter has the correct invoker and can be accessed independently
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			exporter, ok := exporters[test.key]
			assert.True(t, ok, "exporter with key %s should exist in exporters map", test.key)
			assert.NotNil(t, exporter, "exporter should not be nil")
			assert.Equal(t, mockInvokers[test.key], exporter.GetInvoker(), "exporter should return the correct invoker")
		})
	}
}

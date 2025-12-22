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

package impl

import (
	"bytes"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

// TestNewDubboPackage tests creating a new DubboPackage
func TestNewDubboPackage(t *testing.T) {
	tests := []struct {
		desc      string
		data      *bytes.Buffer
		expectNil bool
	}{
		{
			desc:      "with nil data",
			data:      nil,
			expectNil: false,
		},
		{
			desc:      "with empty buffer",
			data:      bytes.NewBuffer([]byte{}),
			expectNil: false,
		},
		{
			desc:      "with buffer containing data",
			data:      bytes.NewBuffer([]byte{0x01, 0x02, 0x03}),
			expectNil: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(test.data)

			if test.expectNil {
				assert.Nil(t, pkg)
			} else {
				assert.NotNil(t, pkg)
				assert.NotNil(t, pkg.Codec)
				assert.Nil(t, pkg.Body)
				assert.Nil(t, pkg.Err)
			}
		})
	}
}

// TestDubboPackageIsHeartBeat tests IsHeartBeat method
func TestDubboPackageIsHeartBeat(t *testing.T) {
	tests := []struct {
		desc       string
		headerType PackageType
		expected   bool
	}{
		{
			desc:       "heartbeat package",
			headerType: PackageHeartbeat,
			expected:   true,
		},
		{
			desc:       "request package",
			headerType: PackageRequest,
			expected:   false,
		},
		{
			desc:       "response package",
			headerType: PackageResponse,
			expected:   false,
		},
		{
			desc:       "heartbeat with other flags",
			headerType: PackageHeartbeat | PackageRequest,
			expected:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.Type = test.headerType

			result := pkg.IsHeartBeat()
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDubboPackageIsRequest tests IsRequest method
func TestDubboPackageIsRequest(t *testing.T) {
	tests := []struct {
		desc       string
		headerType PackageType
		expected   bool
	}{
		{
			desc:       "request package",
			headerType: PackageRequest,
			expected:   true,
		},
		{
			desc:       "two way request package",
			headerType: PackageRequest_TwoWay,
			expected:   true,
		},
		{
			desc:       "response package",
			headerType: PackageResponse,
			expected:   false,
		},
		{
			desc:       "heartbeat package",
			headerType: PackageHeartbeat,
			expected:   false,
		},
		{
			desc:       "request with twoway flag",
			headerType: PackageRequest | PackageRequest_TwoWay,
			expected:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.Type = test.headerType

			result := pkg.IsRequest()
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDubboPackageIsResponse tests IsResponse method
func TestDubboPackageIsResponse(t *testing.T) {
	tests := []struct {
		desc       string
		headerType PackageType
		expected   bool
	}{
		{
			desc:       "response package",
			headerType: PackageResponse,
			expected:   true,
		},
		{
			desc:       "request package",
			headerType: PackageRequest,
			expected:   false,
		},
		{
			desc:       "heartbeat package",
			headerType: PackageHeartbeat,
			expected:   false,
		},
		{
			desc:       "response with exception",
			headerType: PackageResponse | PackageResponse_Exception,
			expected:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.Type = test.headerType

			result := pkg.IsResponse()
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDubboPackageIsResponseWithException tests IsResponseWithException method
func TestDubboPackageIsResponseWithException(t *testing.T) {
	tests := []struct {
		desc       string
		headerType PackageType
		expected   bool
	}{
		{
			desc:       "response with exception",
			headerType: PackageResponse | PackageResponse_Exception,
			expected:   true,
		},
		{
			desc:       "response without exception",
			headerType: PackageResponse,
			expected:   false,
		},
		{
			desc:       "request package",
			headerType: PackageRequest,
			expected:   false,
		},
		{
			desc:       "only exception flag",
			headerType: PackageResponse_Exception,
			expected:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.Type = test.headerType

			result := pkg.IsResponseWithException()
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDubboPackageGetBodyLen tests GetBodyLen method
func TestDubboPackageGetBodyLen(t *testing.T) {
	tests := []struct {
		desc     string
		bodyLen  int
		expected int
	}{
		{
			desc:     "zero length body",
			bodyLen:  0,
			expected: 0,
		},
		{
			desc:     "100 bytes body",
			bodyLen:  100,
			expected: 100,
		},
		{
			desc:     "1000 bytes body",
			bodyLen:  1000,
			expected: 1000,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.BodyLen = test.bodyLen

			result := pkg.GetBodyLen()
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDubboPackageGetLen tests GetLen method
func TestDubboPackageGetLen(t *testing.T) {
	tests := []struct {
		desc     string
		bodyLen  int
		expected int
	}{
		{
			desc:     "zero length body",
			bodyLen:  0,
			expected: 16, // HEADER_LENGTH = 16
		},
		{
			desc:     "100 bytes body",
			bodyLen:  100,
			expected: 116, // 16 + 100
		},
		{
			desc:     "1000 bytes body",
			bodyLen:  1000,
			expected: 1016, // 16 + 1000
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.BodyLen = test.bodyLen

			result := pkg.GetLen()
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDubboPackageGetBody tests GetBody method
func TestDubboPackageGetBody(t *testing.T) {
	tests := []struct {
		desc     string
		body     any
		expected any
	}{
		{
			desc:     "with string body",
			body:     "test body",
			expected: "test body",
		},
		{
			desc:     "with integer body",
			body:     42,
			expected: 42,
		},
		{
			desc:     "with map body",
			body:     map[string]any{"key": "value"},
			expected: map[string]any{"key": "value"},
		},
		{
			desc:     "with nil body",
			body:     nil,
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Body = test.body

			result := pkg.GetBody()
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDubboPackageSetBody tests SetBody method
func TestDubboPackageSetBody(t *testing.T) {
	pkg := NewDubboPackage(nil)
	testBody := "test body content"

	pkg.SetBody(testBody)

	assert.Equal(t, testBody, pkg.Body)
	assert.Equal(t, testBody, pkg.GetBody())
}

// TestDubboPackageSetAndGetHeader tests SetHeader and GetHeader methods
func TestDubboPackageSetAndGetHeader(t *testing.T) {
	pkg := NewDubboPackage(nil)
	header := DubboHeader{
		SerialID:       1,
		Type:           PackageRequest,
		ID:             12345,
		BodyLen:        100,
		ResponseStatus: 0,
	}

	pkg.SetHeader(header)

	result := pkg.GetHeader()
	assert.Equal(t, header, result)
	assert.Equal(t, header.ID, result.ID)
	assert.Equal(t, header.Type, result.Type)
}

// TestDubboPackageSetAndGetID tests SetID method
func TestDubboPackageSetAndGetID(t *testing.T) {
	pkg := NewDubboPackage(nil)
	testID := int64(999999)

	pkg.SetID(testID)

	assert.Equal(t, testID, pkg.Header.ID)
	assert.Equal(t, testID, pkg.GetHeader().ID)
}

// TestDubboPackageSetAndGetService tests SetService and GetService methods
func TestDubboPackageSetAndGetService(t *testing.T) {
	pkg := NewDubboPackage(nil)
	service := Service{
		Path:      "com.example.service",
		Interface: "UserService",
		Group:     "production",
		Version:   "1.0.0",
		Method:    "getUser",
		Timeout:   5 * time.Second,
	}

	pkg.SetService(service)

	result := pkg.GetService()
	assert.Equal(t, service, result)
	assert.Equal(t, "com.example.service", result.Path)
	assert.Equal(t, "UserService", result.Interface)
}

// TestDubboPackageSetResponseStatus tests SetResponseStatus method
func TestDubboPackageSetResponseStatus(t *testing.T) {
	pkg := NewDubboPackage(nil)
	status := byte(20)

	pkg.SetResponseStatus(status)

	assert.Equal(t, status, pkg.Header.ResponseStatus)
}

// TestDubboPackageString tests String method
func TestDubboPackageString(t *testing.T) {
	pkg := NewDubboPackage(nil)
	pkg.Service.Path = "test.service"
	pkg.Body = "test body"

	result := pkg.String()

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "HessianPackage")
	assert.Contains(t, result, "test.service")
}

// TestDubboPackageSetSerializer tests SetSerializer method
func TestDubboPackageSetSerializer(t *testing.T) {
	pkg := NewDubboPackage(nil)
	serializer := &HessianSerializer{}

	// Should not panic
	assert.NotPanics(t, func() {
		pkg.SetSerializer(serializer)
	})

	assert.NotNil(t, pkg.Codec)
}

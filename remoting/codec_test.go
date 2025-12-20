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

package remoting

import (
	"bytes"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// ============================================
// Mock Codec Implementation
// ============================================

type mockCodec struct {
	encodeRequestCalled  bool
	encodeResponseCalled bool
	decodeCalled         bool
}

func (m *mockCodec) EncodeRequest(request *Request) (*bytes.Buffer, error) {
	m.encodeRequestCalled = true
	return bytes.NewBuffer([]byte("encoded-request")), nil
}

func (m *mockCodec) EncodeResponse(response *Response) (*bytes.Buffer, error) {
	m.encodeResponseCalled = true
	return bytes.NewBuffer([]byte("encoded-response")), nil
}

func (m *mockCodec) Decode(data []byte) (*DecodeResult, int, error) {
	m.decodeCalled = true
	return &DecodeResult{
		IsRequest: true,
		Result:    "decoded-result",
	}, len(data), nil
}

// ============================================
// DecodeResult Tests
// ============================================

func TestDecodeResult(t *testing.T) {
	tests := []struct {
		name      string
		isRequest bool
		result    any
	}{
		{
			name:      "request decode result",
			isRequest: true,
			result:    "request-data",
		},
		{
			name:      "response decode result",
			isRequest: false,
			result:    "response-data",
		},
		{
			name:      "nil result",
			isRequest: true,
			result:    nil,
		},
		{
			name:      "complex result",
			isRequest: false,
			result:    map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := &DecodeResult{
				IsRequest: tt.isRequest,
				Result:    tt.result,
			}
			assert.Equal(t, tt.isRequest, dr.IsRequest)
			assert.Equal(t, tt.result, dr.Result)
		})
	}
}

// ============================================
// Codec Registry Tests
// ============================================

func TestRegistryCodec(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{"dubbo protocol", "dubbo"},
		{"triple protocol", "triple"},
		{"grpc protocol", "grpc"},
		{"custom protocol", "custom-protocol"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockC := &mockCodec{}
			RegistryCodec(tt.protocol, mockC)

			retrieved := GetCodec(tt.protocol)
			assert.NotNil(t, retrieved)
			assert.Equal(t, mockC, retrieved)
		})
	}
}

func TestGetCodecNotFound(t *testing.T) {
	result := GetCodec("non-existent-protocol")
	assert.Nil(t, result)
}

func TestRegistryCodecOverwrite(t *testing.T) {
	protocol := "overwrite-test-unique"

	// Register first codec
	codec1 := &mockCodec{}
	RegistryCodec(protocol, codec1)
	retrieved1 := GetCodec(protocol)
	assert.NotNil(t, retrieved1)

	// Overwrite with second codec
	codec2 := &mockCodec{}
	RegistryCodec(protocol, codec2)
	retrieved2 := GetCodec(protocol)
	assert.NotNil(t, retrieved2)

	// The second codec should be retrieved (overwritten)
	assert.Equal(t, codec2, retrieved2)
}

// ============================================
// Codec Interface Tests
// ============================================

func TestMockCodecEncodeRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  *Request
		wantErr  bool
	}{
		{
			name: "basic request",
			request: &Request{
				ID:      1,
				Version: "2.0.2",
				Data:    "test-data",
			},
			wantErr: false,
		},
		{
			name: "request with nil data",
			request: &Request{
				ID:      2,
				Version: "2.0.2",
				Data:    nil,
			},
			wantErr: false,
		},
		{
			name: "request with complex data",
			request: &Request{
				ID:      3,
				Version: "2.0.2",
				Data:    map[string]any{"key": "value"},
				TwoWay:  true,
				Event:   false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockC := &mockCodec{}
			buf, err := mockC.EncodeRequest(tt.request)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, buf)
				assert.True(t, mockC.encodeRequestCalled)
			}
		})
	}
}

func TestMockCodecEncodeResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		wantErr  bool
	}{
		{
			name: "basic response",
			response: &Response{
				ID:      1,
				Version: "2.0.2",
				Result:  "test-result",
			},
			wantErr: false,
		},
		{
			name: "response with error",
			response: &Response{
				ID:      2,
				Version: "2.0.2",
				Error:   assert.AnError,
			},
			wantErr: false,
		},
		{
			name: "heartbeat response",
			response: &Response{
				ID:     3,
				Event:  true,
				Result: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockC := &mockCodec{}
			buf, err := mockC.EncodeResponse(tt.response)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, buf)
				assert.True(t, mockC.encodeResponseCalled)
			}
		})
	}
}

func TestMockCodecDecode(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectedLength int
		wantErr        bool
	}{
		{
			name:           "basic data",
			data:           []byte("test-data"),
			expectedLength: 9,
			wantErr:        false,
		},
		{
			name:           "empty data",
			data:           []byte{},
			expectedLength: 0,
			wantErr:        false,
		},
		{
			name:           "binary data",
			data:           []byte{0x00, 0x01, 0x02, 0x03},
			expectedLength: 4,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockC := &mockCodec{}
			result, length, err := mockC.Decode(tt.data)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedLength, length)
				assert.True(t, mockC.decodeCalled)
			}
		})
	}
}

// ============================================
// Concurrent Access Tests
// ============================================

// Note: The original codec map is not thread-safe, so we test sequential access
// In production, codec registration typically happens during initialization
func TestCodecRegistrySequentialAccess(t *testing.T) {
	// Sequential registration
	for i := 0; i < 10; i++ {
		protocol := "sequential-protocol"
		mockC := &mockCodec{}
		RegistryCodec(protocol, mockC)
	}

	// Sequential retrieval
	for i := 0; i < 10; i++ {
		result := GetCodec("sequential-protocol")
		assert.NotNil(t, result)
	}
}

// ============================================
// Edge Cases Tests
// ============================================

func TestCodecEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		codec    Codec
		validate func(*testing.T, string, Codec)
	}{
		{
			name:     "empty protocol name",
			protocol: "",
			codec:    &mockCodec{},
			validate: func(t *testing.T, protocol string, codec Codec) {
				RegistryCodec(protocol, codec)
				assert.Equal(t, codec, GetCodec(protocol))
			},
		},
		{
			name:     "special characters in protocol",
			protocol: "protocol-v1.0_test",
			codec:    &mockCodec{},
			validate: func(t *testing.T, protocol string, codec Codec) {
				RegistryCodec(protocol, codec)
				assert.Equal(t, codec, GetCodec(protocol))
			},
		},
		{
			name:     "nil codec registration",
			protocol: "nil-codec-unique",
			codec:    nil,
			validate: func(t *testing.T, protocol string, codec Codec) {
				RegistryCodec(protocol, codec)
				assert.Nil(t, GetCodec(protocol))
			},
		},
		{
			name:     "unicode protocol name",
			protocol: "协议名称",
			codec:    &mockCodec{},
			validate: func(t *testing.T, protocol string, codec Codec) {
				RegistryCodec(protocol, codec)
				assert.Equal(t, codec, GetCodec(protocol))
			},
		},
		{
			name:     "very long protocol name",
			protocol: "this-is-a-very-long-protocol-name-for-testing-purposes-that-exceeds-normal-length",
			codec:    &mockCodec{},
			validate: func(t *testing.T, protocol string, codec Codec) {
				RegistryCodec(protocol, codec)
				assert.Equal(t, codec, GetCodec(protocol))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.protocol, tt.codec)
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkRegistryCodec(b *testing.B) {
	mockC := &mockCodec{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RegistryCodec("bench-protocol", mockC)
	}
}

func BenchmarkGetCodec(b *testing.B) {
	mockC := &mockCodec{}
	RegistryCodec("bench-get-protocol", mockC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetCodec("bench-get-protocol")
	}
}

func BenchmarkCodecEncode(b *testing.B) {
	mockC := &mockCodec{}
	request := &Request{ID: 1, Version: "2.0.2"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mockC.EncodeRequest(request)
	}
}

func BenchmarkCodecDecode(b *testing.B) {
	mockC := &mockCodec{}
	data := []byte("test-data-for-benchmark")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = mockC.Decode(data)
	}
}

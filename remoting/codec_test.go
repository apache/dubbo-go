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

type mockCodec struct{}

func (m *mockCodec) EncodeRequest(request *Request) (*bytes.Buffer, error) {
	return bytes.NewBuffer([]byte("encoded-request")), nil
}

func (m *mockCodec) EncodeResponse(response *Response) (*bytes.Buffer, error) {
	return bytes.NewBuffer([]byte("encoded-response")), nil
}

func (m *mockCodec) Decode(data []byte) (*DecodeResult, int, error) {
	return &DecodeResult{IsRequest: true, Result: "decoded"}, len(data), nil
}

// Compile-time check
var _ Codec = (*mockCodec)(nil)

func TestDecodeResult(t *testing.T) {
	dr := &DecodeResult{IsRequest: true, Result: "test-data"}
	assert.True(t, dr.IsRequest)
	assert.Equal(t, "test-data", dr.Result)

	dr2 := &DecodeResult{IsRequest: false, Result: nil}
	assert.False(t, dr2.IsRequest)
	assert.Nil(t, dr2.Result)
}

func TestRegistryCodec(t *testing.T) {
	protocol := "test-protocol"
	mockC := &mockCodec{}

	RegistryCodec(protocol, mockC)
	retrieved := GetCodec(protocol)

	assert.NotNil(t, retrieved)
	assert.Equal(t, mockC, retrieved)
}

func TestGetCodecNotFound(t *testing.T) {
	result := GetCodec("non-existent-protocol")
	assert.Nil(t, result)
}

func TestRegistryCodecOverwrite(t *testing.T) {
	protocol := "overwrite-test"

	codec1 := &mockCodec{}
	RegistryCodec(protocol, codec1)

	codec2 := &mockCodec{}
	RegistryCodec(protocol, codec2)

	assert.Equal(t, codec2, GetCodec(protocol))
}

func TestCodecInterface(t *testing.T) {
	mockC := &mockCodec{}

	// EncodeRequest
	buf, err := mockC.EncodeRequest(&Request{ID: 1, Version: "2.0.2"})
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	// EncodeResponse
	buf, err = mockC.EncodeResponse(&Response{ID: 1, Version: "2.0.2"})
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	// Decode
	result, length, err := mockC.Decode([]byte("test-data"))
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 9, length)
}

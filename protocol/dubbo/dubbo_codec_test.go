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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// TestIsRequest tests isRequest with various bit patterns
func TestIsRequest(t *testing.T) {
	codec := &DubboCodec{}

	tests := []struct {
		desc     string
		data     []byte
		expected bool
	}{
		{
			desc:     "request with 0x80 at position 2",
			data:     []byte{0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: true,
		},
		{
			desc:     "request with 0xFF at position 2",
			data:     []byte{0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: true,
		},
		{
			desc:     "response with 0x00 at position 2",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			desc:     "response with 0x7F at position 2",
			data:     []byte{0x00, 0x00, 0x7F, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := codec.isRequest(test.data)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestDecodeWithInsufficientData tests decode with insufficient header data
func TestDecodeWithInsufficientData(t *testing.T) {
	codec := &DubboCodec{}

	// Data shorter than header length
	data := []byte{0x01, 0x02}

	decodeResult, length, err := codec.Decode(data)
	assert.Nil(t, decodeResult)
	assert.Equal(t, 0, length)
	assert.NoError(t, err)
}

// TestDecodeWithEmptyData tests decode with empty data
func TestDecodeWithEmptyData(t *testing.T) {
	codec := &DubboCodec{}

	data := []byte{}

	decodeResult, length, err := codec.Decode(data)
	assert.Nil(t, decodeResult)
	assert.Equal(t, 0, length)
	assert.NoError(t, err)
}

// TestCodecType tests that DubboCodec is properly instantiated
func TestCodecType(t *testing.T) {
	codec := &DubboCodec{}

	// Verify codec implements the expected interface by checking it has the methods
	assert.NotNil(t, codec.EncodeRequest)
	assert.NotNil(t, codec.EncodeResponse)
	assert.NotNil(t, codec.Decode)
	assert.NotNil(t, codec.encodeHeartbeatRequest)
}

// TestIsRequestEdgeCases tests isRequest with edge case bit patterns
func TestIsRequestEdgeCases(t *testing.T) {
	codec := &DubboCodec{}

	tests := []struct {
		desc     string
		data     []byte
		expected bool
			desc:     "bit 0 only",
			data:     []byte{0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			desc:     "bit 1 only",
			data:     []byte{0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: false,
			desc:     "bit 2 only (not bit 5)",
		{
			desc:     "bit 2 only",
			data:     []byte{0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			desc:     "multiple bits set including bit 7",
			data:     []byte{0xFF, 0xFF, 0x80, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: true,
		},
		{
			desc:     "all bits set except bit 7",
		{
			desc:     "all bits set except bit 7 (no request bit)",
			data:     []byte{0xFF, 0xFF, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := codec.isRequest(test.data)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestCodecMultipleInstances tests that multiple codec instances work independently
func TestCodecMultipleInstances(t *testing.T) {
	codec1 := &DubboCodec{}
	codec2 := &DubboCodec{}

	// Both should produce the same results independently
	testData := []byte{0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00}

	result1 := codec1.isRequest(testData)
	result2 := codec2.isRequest(testData)

	assert.Equal(t, result1, result2)
	assert.True(t, result1)
}

// TestDecodeDataLength tests various data lengths
func TestDecodeDataLength(t *testing.T) {
	codec := &DubboCodec{}

	tests := []struct {
		desc            string
		dataLen         int
		shouldReturnNil bool
	}{
		{
			desc:            "1 byte",
			dataLen:         1,
			shouldReturnNil: true,
		},
		{
			desc:            "5 bytes",
			dataLen:         5,
			shouldReturnNil: true,
		},
		{
			desc:            "15 bytes",
			dataLen:         15,
			shouldReturnNil: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			data := make([]byte, test.dataLen)
			decodeResult, _, err := codec.Decode(data)

			if test.shouldReturnNil {
				assert.Nil(t, decodeResult)
			}
			assert.NoError(t, err)
		})
	}
}

// TestIsRequestConsistency tests that isRequest gives consistent results
func TestIsRequestConsistency(t *testing.T) {
	codec := &DubboCodec{}
	data := []byte{0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00}

	// Call multiple times, should get same result
	result1 := codec.isRequest(data)
	result2 := codec.isRequest(data)
	result3 := codec.isRequest(data)

	assert.Equal(t, result1, result2)
	assert.Equal(t, result2, result3)
	assert.True(t, result1)
}

// TestRequestBitMask tests the request bit mask logic
func TestRequestBitMask(t *testing.T) {
	codec := &DubboCodec{}

	// Test boundary values for byte at position 2 with bit 7 (0x80) set
	tests := []struct {
		desc      string
		byteValue byte
		expected  bool
	}{
		{
			desc:      "0x00 - no bits set",
			byteValue: 0x00,
			expected:  false,
		},
		{
			desc:      "0x01 - only bit 0 set",
			byteValue: 0x01,
			expected:  false,
		},
		{
			desc:      "0x7F - bits 0-6 set",
			byteValue: 0x7F,
			expected:  false,
		},
		{
			desc:      "0x80 - only bit 7 set",
			byteValue: 0x80,
			expected:  true,
		},
		{
			desc:      "0x81 - bits 0 and 7 set",
			byteValue: 0x81,
			expected:  true,
		},
		{
			desc:      "0xFF - all bits set",
			byteValue: 0xFF,
			expected:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			data := []byte{0x00, 0x00, test.byteValue, 0x00, 0x00, 0x00, 0x00, 0x00}
			result := codec.isRequest(data)
			assert.Equal(t, test.expected, result)
		})
	}
}

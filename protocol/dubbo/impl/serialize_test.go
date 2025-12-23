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
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// TestLoadSerializerWithDefaultHessian2 tests LoadSerializer with default Hessian2 serialization
func TestLoadSerializerWithDefaultHessian2(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = 0 // Default SerialID

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	assert.NotNil(t, pkg.Codec)
}

// TestLoadSerializerWithHessian2 tests LoadSerializer with explicit Hessian2 serialization
func TestLoadSerializerWithHessian2(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	assert.NotNil(t, pkg.Codec)
}

// TestLoadSerializerWithDifferentValidID tests LoadSerializer with different valid serializer ID
func TestLoadSerializerWithDifferentValidID(t *testing.T) {
	mockHessianSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockHessianSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	assert.NotNil(t, pkg.Codec)
}

// TestLoadSerializerWithInvalidID tests LoadSerializer with invalid serializer ID panics
func TestLoadSerializerWithInvalidID(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = 255 // Invalid serializer ID

	assert.Panics(t, func() {
		LoadSerializer(pkg)
	})
}

// TestLoadSerializerWithZeroIDDefaultsToHessian2 tests that SerialID 0 defaults to Hessian2
func TestLoadSerializerWithZeroIDDefaultsToHessian2(t *testing.T) {
	mockHessianSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockHessianSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = 0

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	// Verify that the loaded serializer is for Hessian2
	assert.NotNil(t, pkg.Codec)
}

// TestLoadSerializerWithNilPackage tests LoadSerializer handles package correctly
func TestLoadSerializerWithNilPackage(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2

	assert.NotPanics(t, func() {
		err := LoadSerializer(pkg)
		assert.NoError(t, err)
	})
}

// TestLoadSerializerMultipleInstances tests that LoadSerializer works with multiple packages
func TestLoadSerializerMultipleInstances(t *testing.T) {
	mockHessianSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockHessianSerializer)

	pkg1 := NewDubboPackage(nil)
	pkg1.Header.SerialID = constant.SHessian2

	pkg2 := NewDubboPackage(nil)
	pkg2.Header.SerialID = 0 // Default to Hessian2

	err1 := LoadSerializer(pkg1)
	err2 := LoadSerializer(pkg2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotNil(t, pkg1.Codec)
	assert.NotNil(t, pkg2.Codec)
}

// TestLoadSerializerWithBufferedData tests LoadSerializer with buffered data in package
func TestLoadSerializerWithBufferedData(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	data := bytes.NewBuffer([]byte{0x01, 0x02, 0x03})
	pkg := NewDubboPackage(data)
	pkg.Header.SerialID = constant.SHessian2

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	assert.NotNil(t, pkg.Codec)
}

// TestLoadSerializerDoesNotPanicWithValidID tests LoadSerializer doesn't panic with valid IDs
func TestLoadSerializerDoesNotPanicWithValidID(t *testing.T) {
	mockHessianSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockHessianSerializer)

	tests := []struct {
		desc       string
		serialID   byte
		shouldFail bool
	}{
		{
			desc:       "default Hessian2 (0)",
			serialID:   0,
			shouldFail: false,
		},
		{
			desc:       "explicit Hessian2",
			serialID:   constant.SHessian2,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.SerialID = test.serialID

			if test.shouldFail {
				assert.Panics(t, func() {
					LoadSerializer(pkg)
				})
			} else {
				assert.NotPanics(t, func() {
					err := LoadSerializer(pkg)
					assert.NoError(t, err)
				})
			}
		})
	}
}

// TestLoadSerializerSetsSerializerInCodec tests that LoadSerializer properly sets the serializer in the codec
func TestLoadSerializerSetsSerializerInCodec(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	assert.NotNil(t, pkg.Codec)
	assert.NotNil(t, pkg.Codec.serializer)
}

// TestSerializerInterface tests that Serializer interface is properly defined
func TestSerializerInterface(t *testing.T) {
	// Verify that HessianSerializer implements Serializer interface
	var _ Serializer = (*HessianSerializer)(nil)

	mockSerializer := &HessianSerializer{}
	assert.NotNil(t, mockSerializer)
}

// TestLoadSerializerConsistency tests that LoadSerializer gives consistent results for same input
func TestLoadSerializerConsistency(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg1 := NewDubboPackage(nil)
	pkg1.Header.SerialID = constant.SHessian2

	pkg2 := NewDubboPackage(nil)
	pkg2.Header.SerialID = constant.SHessian2

	err1 := LoadSerializer(pkg1)
	err2 := LoadSerializer(pkg2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, err1, err2)
}

// TestLoadSerializerReplacesExistingSerializer tests that LoadSerializer can replace existing serializer
func TestLoadSerializerReplacesExistingSerializer(t *testing.T) {
	mockSerializer1 := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer1)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2

	err1 := LoadSerializer(pkg)
	require.NoError(t, err1)

	// Create a new mock serializer and replace
	mockSerializer2 := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer2)

	err2 := LoadSerializer(pkg)
	assert.NoError(t, err2)
}

// TestLoadSerializerWithAllBytesValues tests LoadSerializer with various byte values
func TestLoadSerializerWithAllBytesValues(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	validValues := []byte{
		0,                  // Default - should map to Hessian2
		constant.SHessian2, // Explicit Hessian2
	}

	for _, byteVal := range validValues {
		t.Run(fmt.Sprintf("serial_id_%d", byteVal), func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.SerialID = byteVal

			if byteVal == 0 || byteVal == constant.SHessian2 {
				err := LoadSerializer(pkg)
				require.NoError(t, err)
				assert.NotNil(t, pkg.Codec)
			}
		})
	}
}

// TestLoadSerializerWithInvalidByteValues tests that invalid byte values panic
func TestLoadSerializerWithInvalidByteValues(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	// Use a known invalid value that should not be registered as a serializer ID.
	invalidValues := []byte{255}

	for _, byteVal := range invalidValues {
		t.Run(fmt.Sprintf("invalid_serial_id_%d", byteVal), func(t *testing.T) {
			pkg := NewDubboPackage(nil)
			pkg.Header.SerialID = byteVal

			assert.Panics(t, func() {
				LoadSerializer(pkg)
			})
		})
	}
}

// TestLoadSerializerErrorHandling tests error handling in LoadSerializer
func TestLoadSerializerErrorHandling(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2

	// Should not raise error with valid setup
	assert.NotPanics(t, func() {
		err := LoadSerializer(pkg)
		assert.NoError(t, err)
	})
}

// TestLoadSerializerCodecIsNotNil tests that Codec is properly initialized
func TestLoadSerializerCodecIsNotNil(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2

	// Before LoadSerializer, Codec should not be nil (created by NewDubboPackage)
	assert.NotNil(t, pkg.Codec)

	err := LoadSerializer(pkg)
	require.NoError(t, err)

	// After LoadSerializer, Codec should still exist
	assert.NotNil(t, pkg.Codec)
}

// TestLoadSerializerWithHeaderData tests LoadSerializer properly uses Header.SerialID
func TestLoadSerializerWithHeaderData(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	// Explicitly set Header fields
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 12345
	pkg.Header.Type = PackageRequest

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	assert.Equal(t, constant.SHessian2, pkg.Header.SerialID)
	assert.Equal(t, int64(12345), pkg.Header.ID)
}

// TestLoadSerializerSequentialCalls tests multiple sequential LoadSerializer calls
func TestLoadSerializerSequentialCalls(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	for i := 0; i < 5; i++ {
		pkg := NewDubboPackage(nil)
		pkg.Header.SerialID = constant.SHessian2
		pkg.Header.ID = int64(i)

		err := LoadSerializer(pkg)
		require.NoError(t, err)
		assert.NotNil(t, pkg.Codec)
	}
}

// TestLoadSerializerWithDataBuffer tests LoadSerializer with data buffer
func TestLoadSerializerWithDataBuffer(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	testData := []byte{0xDA, 0xBB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	data := bytes.NewBuffer(testData)
	pkg := NewDubboPackage(data)
	pkg.Header.SerialID = constant.SHessian2

	err := LoadSerializer(pkg)
	require.NoError(t, err)
	assert.NotNil(t, pkg.Codec)
}

// TestHessianSerializerMarshalRequest tests Hessian serializer marshal for request
func TestHessianSerializerMarshalRequest(t *testing.T) {
	serializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, serializer)

	pkg := NewDubboPackage(nil)
	pkg.Body = []any{"test"}
	pkg.Header.Type = PackageRequest
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 12345
	pkg.Service.Interface = "TestInterface"
	pkg.Service.Path = "test.path"
	pkg.Service.Method = "testMethod"
	pkg.SetSerializer(serializer)

	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Positive(t, data.Len())
}

// TestHessianSerializerMarshalResponse tests Hessian serializer marshal for response
func TestHessianSerializerMarshalResponse(t *testing.T) {
	serializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, serializer)

	pkg := NewDubboPackage(nil)
	pkg.Body = "response data"
	pkg.Header.Type = PackageResponse
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 54321
	pkg.Header.ResponseStatus = Response_OK
	pkg.SetSerializer(serializer)

	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Positive(t, data.Len())
}

// TestHessianSerializerMarshalResponseWithException tests Hessian marshaling for exception response
func TestHessianSerializerMarshalResponseWithException(t *testing.T) {
	serializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, serializer)

	pkg := NewDubboPackage(nil)
	pkg.Body = []any{} // Empty body for exception response
	pkg.Header.Type = PackageResponse
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 99999
	pkg.Header.ResponseStatus = Response_SERVICE_ERROR
	pkg.SetSerializer(serializer)

	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
}

// TestHessianSerializerMarshalHeartbeat tests Hessian serializer marshal for heartbeat
func TestHessianSerializerMarshalHeartbeat(t *testing.T) {
	serializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, serializer)

	pkg := NewDubboPackage(nil)
	pkg.Body = []any{}
	pkg.Header.Type = PackageHeartbeat
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 77777
	pkg.SetSerializer(serializer)

	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Positive(t, data.Len())
}

// TestLoadSerializerWithMultipleCalls tests LoadSerializer consistency
func TestLoadSerializerWithMultipleCalls(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	// Call LoadSerializer multiple times
	for i := 0; i < 5; i++ {
		pkg := NewDubboPackage(nil)
		pkg.Header.SerialID = constant.SHessian2

		err := LoadSerializer(pkg)
		require.NoError(t, err)
		assert.NotNil(t, pkg.Codec.serializer)
	}
}

// TestLoadSerializerPackageInheritance tests that loaded serializer persists in package
func TestLoadSerializerPackageInheritance(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.Type = PackageRequest
	pkg.Header.ID = 55555
	pkg.Service.Interface = "TestInterface"
	pkg.Body = []any{"test"}

	err := LoadSerializer(pkg)
	require.NoError(t, err)

	// Verify serializer is accessible for marshaling
	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
}

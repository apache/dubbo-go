// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"google.golang.org/protobuf/proto"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/interoperability"
)

// TestUser is a test POJO for hessian2 serialization
type TestUser struct {
	ID   string
	Name string
	Age  int32
}

func (u *TestUser) JavaClassName() string {
	return "org.apache.dubbo.samples.User"
}

func init() {
	hessian.RegisterPOJO(&TestUser{})
}

// =============================================================================
// protoWrapperCodec Tests
// =============================================================================

func TestProtoWrapperCodec_Name(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})
	assert.Equal(t, codec.Name(), codecNameHessian2)
}

func TestProtoWrapperCodec_WireCodecName(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})
	assert.Equal(t, codec.WireCodecName(), codecNameProto)
}

func TestProtoWrapperCodec_ImplementsWrapperCodec(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})
	var _ WrapperCodec = codec // Compile-time check
}

func TestProtoWrapperCodec_MarshalRequest_SingleArg(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Marshal a single string argument
	data, err := codec.Marshal([]any{"hello"})
	assert.Nil(t, err)
	assert.True(t, len(data) > 0)

	// Verify it's a valid TripleRequestWrapper
	var wrapper interoperability.TripleRequestWrapper
	err = proto.Unmarshal(data, &wrapper)
	assert.Nil(t, err)
	assert.Equal(t, wrapper.SerializeType, codecNameHessian2)
	assert.Equal(t, len(wrapper.Args), 1)
	assert.Equal(t, len(wrapper.ArgTypes), 1)
	assert.Equal(t, wrapper.ArgTypes[0], "java.lang.String")
}

func TestProtoWrapperCodec_MarshalRequest_MultipleArgs(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Marshal multiple arguments
	data, err := codec.Marshal([]any{"hello", int32(42), true})
	assert.Nil(t, err)

	var wrapper interoperability.TripleRequestWrapper
	err = proto.Unmarshal(data, &wrapper)
	assert.Nil(t, err)
	assert.Equal(t, len(wrapper.Args), 3)
	assert.Equal(t, wrapper.ArgTypes[0], "java.lang.String")
	assert.Equal(t, wrapper.ArgTypes[1], "int")
	assert.Equal(t, wrapper.ArgTypes[2], "boolean")
}

func TestProtoWrapperCodec_MarshalRequest_POJO(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	user := &TestUser{ID: "001", Name: "test", Age: 25}
	data, err := codec.Marshal([]any{user})
	assert.Nil(t, err)

	var wrapper interoperability.TripleRequestWrapper
	err = proto.Unmarshal(data, &wrapper)
	assert.Nil(t, err)
	assert.Equal(t, len(wrapper.Args), 1)
	assert.Equal(t, wrapper.ArgTypes[0], "org.apache.dubbo.samples.User")
}

func TestProtoWrapperCodec_UnmarshalRequest(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Create a TripleRequestWrapper
	hessianCodec := &hessian2Codec{}
	arg1, _ := hessianCodec.Marshal("hello")
	arg2, _ := hessianCodec.Marshal(int32(42))

	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: codecNameHessian2,
		Args:          [][]byte{arg1, arg2},
		ArgTypes:      []string{"java.lang.String", "int"},
	}
	data, _ := proto.Marshal(wrapper)

	// Unmarshal - use interface pointers that hessian2 can fill
	results := make([]any, 2)
	for i := range results {
		var v any
		results[i] = &v
	}
	err := codec.Unmarshal(data, results)
	assert.Nil(t, err)

	// Verify the unmarshaled values
	val0 := *(results[0].(*any))
	val1 := *(results[1].(*any))
	assert.Equal(t, val0, "hello")
	assert.Equal(t, val1, int32(42))
}

func TestProtoWrapperCodec_UnmarshalResponse(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Create a TripleResponseWrapper
	hessianCodec := &hessian2Codec{}
	respData, _ := hessianCodec.Marshal(map[string]any{
		"id":   "001",
		"name": "test",
		"age":  25,
	})

	wrapper := &interoperability.TripleResponseWrapper{
		SerializeType: codecNameHessian2,
		Data:          respData,
		Type:          "java.util.Map",
	}
	data, _ := proto.Marshal(wrapper)

	// Unmarshal
	var result any
	err := codec.Unmarshal(data, &result)
	assert.Nil(t, err)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[any]any)
	assert.True(t, ok)
	assert.Equal(t, resultMap["id"], "001")
	assert.Equal(t, resultMap["name"], "test")
}

func TestProtoWrapperCodec_RoundTrip_Request(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Marshal
	original := []any{"hello", int32(42)}
	data, err := codec.Marshal(original)
	assert.Nil(t, err)

	// Unmarshal into request format (simulating server receiving)
	var str string
	var num int32
	params := []any{&str, &num}

	// First parse as TripleRequestWrapper to verify format
	var wrapper interoperability.TripleRequestWrapper
	err = proto.Unmarshal(data, &wrapper)
	assert.Nil(t, err)

	// Now unmarshal the actual data
	hessianCodec := &hessian2Codec{}
	err = hessianCodec.Unmarshal(wrapper.Args[0], &str)
	assert.Nil(t, err)
	err = hessianCodec.Unmarshal(wrapper.Args[1], &num)
	assert.Nil(t, err)

	assert.Equal(t, str, "hello")
	assert.Equal(t, num, int32(42))

	_ = params // suppress unused warning
}

// =============================================================================
// protoBinaryCodec Wrapper Tests
// =============================================================================

func TestProtoBinaryCodec_MarshalNonProtoReturnsError(t *testing.T) {
	t.Parallel()

	codec := &protoBinaryCodec{}

	// Marshal a non-proto message should return error
	result := map[string]any{"id": "001", "name": "test"}
	_, err := codec.Marshal(result)
	assert.NotNil(t, err)
}

func TestProtoBinaryCodec_UnmarshalWrappedResponse(t *testing.T) {
	t.Parallel()

	codec := &protoBinaryCodec{}

	// Create a TripleResponseWrapper
	hessianCodec := &hessian2Codec{}
	respData, _ := hessianCodec.Marshal("hello world")

	wrapper := &interoperability.TripleResponseWrapper{
		SerializeType: codecNameHessian2,
		Data:          respData,
		Type:          "java.lang.String",
	}
	data, _ := proto.Marshal(wrapper)

	// Unmarshal
	var result any
	err := codec.Unmarshal(data, &result)
	assert.Nil(t, err)
	assert.Equal(t, result, "hello world")
}

func TestProtoBinaryCodec_UnmarshalWrappedRequest(t *testing.T) {
	t.Parallel()

	codec := &protoBinaryCodec{}

	// Create a TripleRequestWrapper
	hessianCodec := &hessian2Codec{}
	arg1, _ := hessianCodec.Marshal("arg1")
	arg2, _ := hessianCodec.Marshal(int64(123))

	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: codecNameHessian2,
		Args:          [][]byte{arg1, arg2},
		ArgTypes:      []string{"java.lang.String", "long"},
	}
	data, _ := proto.Marshal(wrapper)

	// Unmarshal - use interface pointers that hessian2 can fill
	results := make([]any, 2)
	for i := range results {
		var v any
		results[i] = &v
	}
	err := codec.Unmarshal(data, results)
	assert.Nil(t, err)

	// Verify the unmarshaled values
	val0 := *(results[0].(*any))
	val1 := *(results[1].(*any))
	assert.Equal(t, val0, "arg1")
	assert.Equal(t, val1, int64(123))
}

func TestProtoBinaryCodec_ResponseThenRequestFallback(t *testing.T) {
	t.Parallel()

	codec := &protoBinaryCodec{}

	// Test that it tries TripleResponseWrapper first, then falls back to TripleRequestWrapper
	// Create a valid TripleRequestWrapper
	hessianCodec := &hessian2Codec{}
	arg1, _ := hessianCodec.Marshal("test")

	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: codecNameHessian2,
		Args:          [][]byte{arg1},
		ArgTypes:      []string{"java.lang.String"},
	}
	data, _ := proto.Marshal(wrapper)

	// Should successfully unmarshal as request (after response fallback)
	// Use interface pointer that hessian2 can fill
	results := make([]any, 1)
	var v any
	results[0] = &v
	err := codec.Unmarshal(data, results)
	assert.Nil(t, err)
	assert.Equal(t, *(results[0].(*any)), "test")
}

// =============================================================================
// WrapperCodec Interface Tests
// =============================================================================

func TestGetWireCodecName_WrapperCodec(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})
	wireCodecName := getWireCodecName(codec)
	assert.Equal(t, wireCodecName, codecNameProto)
}

func TestGetWireCodecName_RegularCodec(t *testing.T) {
	t.Parallel()

	codec := &protoBinaryCodec{}
	wireCodecName := getWireCodecName(codec)
	assert.Equal(t, wireCodecName, codecNameProto)
}

func TestGetWireCodecName_Hessian2Codec(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}
	wireCodecName := getWireCodecName(codec)
	assert.Equal(t, wireCodecName, codecNameHessian2)
}

// =============================================================================
// hessian2Codec Tests
// =============================================================================

func TestHessian2Codec_Name(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}
	assert.Equal(t, codec.Name(), codecNameHessian2)
}

func TestHessian2Codec_RoundTrip_String(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}

	original := "hello world"
	data, err := codec.Marshal(original)
	assert.Nil(t, err)

	var result string
	err = codec.Unmarshal(data, &result)
	assert.Nil(t, err)
	assert.Equal(t, result, original)
}

func TestHessian2Codec_RoundTrip_Int(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}

	original := int32(12345)
	data, err := codec.Marshal(original)
	assert.Nil(t, err)

	var result int32
	err = codec.Unmarshal(data, &result)
	assert.Nil(t, err)
	assert.Equal(t, result, original)
}

func TestHessian2Codec_RoundTrip_Map(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}

	original := map[string]any{"key1": "value1", "key2": int64(42)}
	data, err := codec.Marshal(original)
	assert.Nil(t, err)

	var result any
	err = codec.Unmarshal(data, &result)
	assert.Nil(t, err)

	resultMap, ok := result.(map[any]any)
	assert.True(t, ok)
	assert.Equal(t, resultMap["key1"], "value1")
	assert.Equal(t, resultMap["key2"], int64(42))
}

func TestHessian2Codec_RoundTrip_Slice(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}

	original := []string{"a", "b", "c"}
	data, err := codec.Marshal(original)
	assert.Nil(t, err)

	var result any
	err = codec.Unmarshal(data, &result)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

func TestHessian2Codec_RoundTrip_POJO(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}

	original := &TestUser{ID: "001", Name: "test", Age: 25}
	data, err := codec.Marshal(original)
	assert.Nil(t, err)

	var result any
	err = codec.Unmarshal(data, &result)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

// =============================================================================
// getArgType Tests
// =============================================================================

func TestGetArgType_Nil(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(nil), "V")
}

func TestGetArgType_Bool(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(true), "boolean")
	assert.Equal(t, getArgType(false), "boolean")
}

func TestGetArgType_BoolSlice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType([]bool{true, false}), "[Z")
}

func TestGetArgType_Byte(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(byte(1)), "byte")
}

func TestGetArgType_ByteSlice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType([]byte{1, 2, 3}), "[B")
}

func TestGetArgType_Int8(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(int8(1)), "byte")
}

func TestGetArgType_Int16(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(int16(1)), "short")
}

func TestGetArgType_Int32(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(int32(1)), "int")
}

func TestGetArgType_Int64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(int64(1)), "long")
}

func TestGetArgType_Int(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(int(1)), "long")
}

func TestGetArgType_Float32(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(float32(1.0)), "float")
}

func TestGetArgType_Float64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(float64(1.0)), "double")
}

func TestGetArgType_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType("hello"), "java.lang.String")
}

func TestGetArgType_StringSlice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType([]string{"a", "b"}), "[Ljava.lang.String;")
}

func TestGetArgType_Time(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(time.Now()), "java.util.Date")
}

func TestGetArgType_Map(t *testing.T) {
	t.Parallel()
	assert.Equal(t, getArgType(map[any]any{}), "java.util.Map")
	assert.Equal(t, getArgType(map[string]int{}), "java.util.Map")
}

func TestGetArgType_Slice(t *testing.T) {
	t.Parallel()
	// []int maps to [J (Java long array) because Go's int is 64-bit
	assert.Equal(t, getArgType([]int{1, 2, 3}), "[J")
	assert.Equal(t, getArgType([]int32{1, 2, 3}), "[I")
	assert.Equal(t, getArgType([]int64{1, 2, 3}), "[J")
	assert.Equal(t, getArgType([]float64{1.0, 2.0}), "[D")
}

func TestGetArgType_POJO(t *testing.T) {
	t.Parallel()
	user := &TestUser{ID: "001", Name: "test", Age: 25}
	assert.Equal(t, getArgType(user), "org.apache.dubbo.samples.User")
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestProtoWrapperCodec_UnmarshalRequest_ArgCountMismatch(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Create a wrapper with 2 args
	hessianCodec := &hessian2Codec{}
	arg1, _ := hessianCodec.Marshal("hello")
	arg2, _ := hessianCodec.Marshal(int32(42))

	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: codecNameHessian2,
		Args:          [][]byte{arg1, arg2},
		ArgTypes:      []string{"java.lang.String", "int"},
	}
	data, _ := proto.Marshal(wrapper)

	// Try to unmarshal into 1 param (mismatch)
	var str string
	err := codec.Unmarshal(data, []any{&str})
	assert.NotNil(t, err)
}

func TestProtoBinaryCodec_Unmarshal_InvalidData(t *testing.T) {
	t.Parallel()

	codec := &protoBinaryCodec{}

	// Try to unmarshal invalid data into a non-proto type
	invalidData := []byte{0x01, 0x02, 0x03}
	var result any
	err := codec.Unmarshal(invalidData, &result)
	// Should fail because it can't parse as either wrapper
	assert.NotNil(t, err)
}

func TestProtoWrapperCodec_Marshal_EmptyArgs(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Marshal empty args (for no-arg methods)
	data, err := codec.Marshal([]any{})
	assert.Nil(t, err)

	var wrapper interoperability.TripleRequestWrapper
	err = proto.Unmarshal(data, &wrapper)
	assert.Nil(t, err)
	assert.Equal(t, len(wrapper.Args), 0)
}

func TestProtoWrapperCodec_Unmarshal_EmptyRequest(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})

	// Create an empty request wrapper
	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: codecNameHessian2,
		Args:          [][]byte{},
		ArgTypes:      []string{},
	}
	data, _ := proto.Marshal(wrapper)

	// Unmarshal into empty params
	err := codec.Unmarshal(data, []any{})
	assert.Nil(t, err)
}

// =============================================================================
// Msgpack Wrapper Tests
// =============================================================================

func TestProtoWrapperCodec_Msgpack(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&msgpackCodec{})
	assert.Equal(t, codec.Name(), codecNameMsgPack)
	assert.Equal(t, codec.WireCodecName(), codecNameProto)
}

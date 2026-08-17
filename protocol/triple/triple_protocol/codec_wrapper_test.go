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

package triple_protocol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestProtoWrapperCodec_UnmarshalWrappedResponse(t *testing.T) {
	t.Parallel()

	// Client-side: protoWrapperCodec decodes a TripleResponseWrapper.
	codec := newProtoWrapperCodec(&hessian2Codec{})

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

func TestServerCodecSession_UnmarshalWrappedRequest(t *testing.T) {
	t.Parallel()

	// Server-side: tripleServerCodecSession decodes a TripleRequestWrapper and
	// captures SerializeType for the subsequent response Marshal.
	codecSession := &tripleServerCodecSession{delegate: &protoBinaryCodec{}}

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
	err := codecSession.Unmarshal(data, results)
	assert.Nil(t, err)

	// Verify the unmarshaled values
	val0 := *(results[0].(*any))
	val1 := *(results[1].(*any))
	assert.Equal(t, val0, "arg1")
	assert.Equal(t, val1, int64(123))
	assert.Equal(t, codecSession.serializeType, codecNameHessian2)
}

func TestServerCodecSession_UnmarshalSingleArgRequest(t *testing.T) {
	t.Parallel()

	session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}}

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
	err := session.Unmarshal(data, results)
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

func TestServerCodecSession_UnmarshalWrappedRequest_MsgPack(t *testing.T) {
	t.Parallel()

	// Provider configured serialization=msgpack, so msgpack is on the allowlist.
	session := &tripleServerCodecSession{
		delegate:             &protoBinaryCodec{},
		allowedSerializeType: codecNameMsgPack,
	}

	msgpCodec := &msgpackCodec{}
	arg1, _ := msgpCodec.Marshal("msgarg")
	arg2, _ := msgpCodec.Marshal(int64(7))

	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: codecNameMsgPack,
		Args:          [][]byte{arg1, arg2},
		ArgTypes:      []string{"java.lang.String", "long"},
	}
	data, _ := proto.Marshal(wrapper)

	results := make([]any, 2)
	var str string
	var num int64
	results[0] = &str
	results[1] = &num
	// This MUST decode via msgpack directly (SerializeType dispatch).
	err := session.Unmarshal(data, results)
	assert.Nil(t, err)

	assert.Equal(t, str, "msgarg")
	assert.Equal(t, num, int64(7))
	assert.Equal(t, session.serializeType, codecNameMsgPack)
}

func TestServerCodecSession_UnmarshalWrappedRequest_UnknownType(t *testing.T) {
	t.Parallel()

	session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}}
	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: "unknown",
		Args:          [][]byte{{0x01}},
		ArgTypes:      []string{"java.lang.String"},
	}
	data, _ := proto.Marshal(wrapper)

	var v any
	err := session.Unmarshal(data, []any{&v})
	assert.NotNil(t, err) // explicit error, no silent fallback
}

func TestServerCodecSession_UnmarshalWrappedRequest_CorruptPayload(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		serializeType string
		allowed       string // provider allowlist; hessian2 is always implicitly allowed
		payload       []byte
	}{
		// 0xff is not a valid hessian2 type code: decode fails.
		{"hessian2-corrupt", codecNameHessian2, "", []byte{0xff}},
		// 0xc1 is reserved (never used) in the msgpack spec: decode fails.
		{"msgpack-corrupt", codecNameMsgPack, codecNameMsgPack, []byte{0xc1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := &tripleServerCodecSession{
				delegate:             &protoBinaryCodec{},
				allowedSerializeType: tc.allowed,
			}
			wrapper := &interoperability.TripleRequestWrapper{
				SerializeType: tc.serializeType,
				Args:          [][]byte{tc.payload},
				ArgTypes:      []string{"java.lang.String"},
			}
			data, _ := proto.Marshal(wrapper)

			var v any
			// Payload does not match the declared SerializeType: the selected
			// inner codec must surface an explicit decode error.
			err := session.Unmarshal(data, []any{&v})
			assert.NotNil(t, err)
			assert.True(t, strings.Contains(err.Error(), "triple wrapper request"))
		})
	}
}

func TestProtoWrapperCodec_UnmarshalWrappedResponse_UnknownType(t *testing.T) {
	t.Parallel()

	codec := newProtoWrapperCodec(&hessian2Codec{})
	wrapper := &interoperability.TripleResponseWrapper{
		SerializeType: "unknown",
		Data:          []byte{0x01},
	}
	data, _ := proto.Marshal(wrapper)

	var result any
	err := codec.Unmarshal(data, &result)
	assert.NotNil(t, err)
}

func TestServerCodecSession_MarshalWrappedResponse_Hessian2(t *testing.T) {
	t.Parallel()

	session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}, serializeType: codecNameHessian2}
	data, err := session.Marshal("hello")
	assert.Nil(t, err)

	// Decode the produced wrapper and verify round-trip.
	var wrapper interoperability.TripleResponseWrapper
	assert.Nil(t, proto.Unmarshal(data, &wrapper))
	assert.Equal(t, wrapper.SerializeType, codecNameHessian2)

	hessianCodec := &hessian2Codec{}
	var out any
	assert.Nil(t, hessianCodec.Unmarshal(wrapper.Data, &out))
	assert.Equal(t, out, "hello")
}

func TestServerCodecSession_MarshalWrappedResponse_FollowsSerializeType(t *testing.T) {
	t.Parallel()

	session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}, serializeType: codecNameMsgPack}
	data, err := session.Marshal(int64(42))
	assert.Nil(t, err)

	var wrapper interoperability.TripleResponseWrapper
	assert.Nil(t, proto.Unmarshal(data, &wrapper))
	assert.Equal(t, wrapper.SerializeType, codecNameMsgPack)

	msgp := &msgpackCodec{}
	var out int64
	assert.Nil(t, msgp.Unmarshal(wrapper.Data, &out))
	assert.Equal(t, out, int64(42))
}

func TestNonIDLResponse_RoundTrip_Hessian2(t *testing.T) {
	t.Parallel()

	// Server encode.
	session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}, serializeType: codecNameHessian2}
	data, err := session.Marshal("roundtrip-hessian")
	assert.Nil(t, err)

	// Client decode (via protoWrapperCodec, simulating a Non-IDL client).
	codec := newProtoWrapperCodec(&hessian2Codec{})
	var out any
	assert.Nil(t, codec.Unmarshal(data, &out))
	assert.Equal(t, out, "roundtrip-hessian")
}

func TestNonIDLResponse_RoundTrip_MsgPack(t *testing.T) {
	t.Parallel()

	session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}, serializeType: codecNameMsgPack}
	data, err := session.Marshal("roundtrip-msgpack")
	assert.Nil(t, err)

	codec := newProtoWrapperCodec(&msgpackCodec{})
	var out string
	assert.Nil(t, codec.Unmarshal(data, &out))
	assert.Equal(t, out, "roundtrip-msgpack")
}

func TestServerCodecSession_Allowlist_RejectsUnauthorizedSerializeType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		allowed string // provider's configured serialization; "" = hessian2-only
	}{
		{"not-configured-default", ""}, // serialization defaults to protobuf (IDL), not msgpack
		{"configured-hessian2", codecNameHessian2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := &tripleServerCodecSession{
				delegate:             &protoBinaryCodec{},
				allowedSerializeType: tc.allowed,
			}
			msgpCodec := &msgpackCodec{}
			arg, _ := msgpCodec.Marshal("sneaky")
			wrapper := &interoperability.TripleRequestWrapper{
				SerializeType: codecNameMsgPack,
				Args:          [][]byte{arg},
				ArgTypes:      []string{"java.lang.String"},
			}
			data, _ := proto.Marshal(wrapper)

			var v any
			err := session.Unmarshal(data, []any{&v})
			assert.NotNil(t, err)
			assert.True(t, strings.Contains(err.Error(), "not allowed by provider"))
		})
	}
}

func TestServerCodecSession_Allowlist_Hessian2AlwaysAllowed(t *testing.T) {
	t.Parallel()

	// Provider configured serialization=protobuf (the IDL default); hessian2
	// Non-IDL request must still be accepted.
	session := &tripleServerCodecSession{
		delegate:             &protoBinaryCodec{},
		allowedSerializeType: "protobuf",
	}
	hessianCodec := &hessian2Codec{}
	arg, _ := hessianCodec.Marshal("ok")
	wrapper := &interoperability.TripleRequestWrapper{
		SerializeType: codecNameHessian2,
		Args:          [][]byte{arg},
		ArgTypes:      []string{"java.lang.String"},
	}
	data, _ := proto.Marshal(wrapper)

	var v any
	err := session.Unmarshal(data, []any{&v})
	assert.Nil(t, err)
}

// TestServerCodecSession_Marshal_ProductionShape covers the wire shape produced
// by the real handler path (server.go wrapTripleResponse constructs
// []any{result}). The session MUST unwrap the one-element container and serialize
// only the scalar return value into TripleResponseWrapper.Data, matching Java's
// ReflectionPackableMethod.WrapResponsePack. Serializing the whole []any would
// make Hessian2 panic (copySlice on string) and MsgPack return a type error.
func TestServerCodecSession_Marshal_ProductionShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		serializeType string
		allowed       string
		payload       any      // wrapped as []any{payload} to mirror production shape
		newDest       func() any // returns a typed pointer to decode into
		assertDecoded func(t *testing.T, dest any)
	}{
		{
			name:          "hessian2-string",
			serializeType: codecNameHessian2,
			payload:       "hello-prod",
			newDest:       func() any { var s string; return &s },
			assertDecoded: func(t *testing.T, dest any) { assert.Equal(t, *(dest.(*string)), "hello-prod") },
		},
		{
			name:          "hessian2-int64",
			serializeType: codecNameHessian2,
			payload:       int64(99),
			newDest:       func() any { var n int64; return &n },
			assertDecoded: func(t *testing.T, dest any) { assert.Equal(t, *(dest.(*int64)), int64(99)) },
		},
		{
			name:          "msgpack-string",
			serializeType: codecNameMsgPack,
			allowed:       codecNameMsgPack,
			payload:       "msgpack-prod",
			newDest:       func() any { var s string; return &s },
			assertDecoded: func(t *testing.T, dest any) { assert.Equal(t, *(dest.(*string)), "msgpack-prod") },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Server-side encode with the PRODUCTION shape: []any{result}.
			session := &tripleServerCodecSession{
				delegate:             &protoBinaryCodec{},
				serializeType:        tc.serializeType,
				allowedSerializeType: tc.allowed,
			}
			data, err := session.Marshal([]any{tc.payload})
			assert.Nil(t, err)

			// Inspect the wrapper: Data must hold the scalar, not the container.
			var wrapper interoperability.TripleResponseWrapper
			assert.Nil(t, proto.Unmarshal(data, &wrapper))
			assert.Equal(t, wrapper.SerializeType, tc.serializeType)
			assert.True(t, len(wrapper.Data) > 0)

			// Client-side decode via protoWrapperCodec into a typed pointer, the
			// way a real caller (which knows the return type) would.
			clientCodec := newProtoWrapperCodec(resolveInnerCodecOrFail(t, tc.serializeType))
			dest := tc.newDest()
			assert.Nil(t, clientCodec.Unmarshal(data, dest))
			tc.assertDecoded(t, dest)
		})
	}
}

func resolveInnerCodecOrFail(t *testing.T, serializeType string) Codec {
	t.Helper()
	c, err := resolveInnerCodec(serializeType)
	if err != nil {
		t.Fatalf("resolveInnerCodec(%q): %v", serializeType, err)
	}
	return c
}

// TestServerCodecSession_Marshal_VoidResponse covers null/void responses: an
// empty or nil container element must produce an empty Data field (decoded as
// void by the peer), NOT an attempt to serialize nil.
func TestServerCodecSession_Marshal_VoidResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  any
	}{
		{"empty-slice", []any{}},
		{"nil-element", []any{nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := &tripleServerCodecSession{
				delegate:      &protoBinaryCodec{},
				serializeType: codecNameHessian2,
			}
			data, err := session.Marshal(tc.msg)
			assert.Nil(t, err)

			var wrapper interoperability.TripleResponseWrapper
			assert.Nil(t, proto.Unmarshal(data, &wrapper))
			assert.Equal(t, len(wrapper.Data), 0)

			// Client decodes empty Data as void (no error, no value).
			clientCodec := newProtoWrapperCodec(&hessian2Codec{})
			var got any
			assert.Nil(t, clientCodec.Unmarshal(data, &got))
		})
	}
}

// TestProtoWrapperCodec_Unmarshal_EmptyData_UnknownSerializeType covers P1: a
// corrupt response {SerializeType:"unknown", Data:nil} must be rejected even
// though Data is empty. Previously it was silently treated as void.
func TestProtoWrapperCodec_Unmarshal_EmptyData_UnknownSerializeType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		serializeType string
		wantErr       bool
	}{
		// Unknown/disabled types must error regardless of Data emptiness.
		{"unknown-empty-data", "unknown", true},
		{"disabled-empty-data", "fastjson", true},
		// Valid types with empty Data are legitimate void responses.
		{"hessian2-empty-data", codecNameHessian2, false},
		{"msgpack-empty-data", codecNameMsgPack, false},
		// hessian4 aliases hessian2; empty Data is void.
		{"hessian4-empty-data", "hessian4", false},
		// Empty SerializeType defaults to hessian2 (backward compat); void OK.
		{"blank-serialize-type", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			codec := newProtoWrapperCodec(&hessian2Codec{})
			wrapper := &interoperability.TripleResponseWrapper{
				SerializeType: tc.serializeType,
				// Data intentionally nil/empty.
			}
			data, _ := proto.Marshal(wrapper)

			var got any
			err := codec.Unmarshal(data, &got)
			if tc.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestServerCodecSession_Marshal_MultiElementError pins the defined semantics
// for a multi-element response container: it is a programming error (the
// production handler always packs exactly one return value), NOT a silent
// truncation to the first element.
func TestServerCodecSession_Marshal_MultiElementError(t *testing.T) {
	t.Parallel()

	session := &tripleServerCodecSession{
		delegate:      &protoBinaryCodec{},
		serializeType: codecNameHessian2,
	}
	_, err := session.Marshal([]any{"a", "b"})
	assert.NotNil(t, err)
}

// TestNonIDLUnary_PublicEntry_EndToEnd drives a non-IDL unary RPC through the
// public entry points (server: NewUnaryHandler; client: NewClient/CallUnary)
// over HTTP. The handler returns NewResponse([]any{result}), mirroring the
// production shape built by server.go wrapTripleResponse, so the session must
// unwrap the one-element container before serialization. Covers the P0 review
// matrix: {Hessian2, MsgPack} x {concrete type pointer, *any} destinations.
func TestNonIDLUnary_PublicEntry_EndToEnd(t *testing.T) {
	t.Parallel()

	const (
		service = "/test.NonIDLGreeter"
		method  = "SayHello"
	)
	cases := []struct {
		name          string
		clientOption  ClientOption
		handlerOption HandlerOption // nil for hessian2: always on the allowlist
		newDest       func() any
	}{
		{"hessian2-concrete-pointer", WithHessian2(), nil, func() any { return new(string) }},
		{"hessian2-any-pointer", WithHessian2(), nil, func() any { var v any; return &v }},
		{"msgpack-concrete-pointer", WithMsgPack(), WithExpectedCodecName(codecNameMsgPack), func() any { return new(string) }},
		{"msgpack-any-pointer", WithMsgPack(), WithExpectedCodecName(codecNameMsgPack), func() any { var v any; return &v }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handlerOpts := []HandlerOption{}
			if tc.handlerOption != nil {
				handlerOpts = append(handlerOpts, tc.handlerOption)
			}
			mux := http.NewServeMux()
			mux.Handle(service+"/"+method, NewUnaryHandler(
				service+"/"+method,
				func() any { return []any{new(string)} },
				func(_ context.Context, req *Request) (*Response, error) {
					arg := req.Msg.([]any)[0].(*string)
					// Production shape: wrapTripleResponse packs []any{result}.
					return NewResponse([]any{"hello:" + *arg}), nil
				},
				handlerOpts...,
			))
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)

			client := NewClient(server.Client(), server.URL+service, WithTriple(), tc.clientOption)
			resp := &Response{Msg: tc.newDest()}
			assert.Nil(t, client.CallUnary(context.Background(), NewRequest([]any{"world"}), method, resp))

			var result string
			switch dest := resp.Msg.(type) {
			case *string:
				result = *dest
			case *any:
				switch v := (*dest).(type) {
				case string:
					result = v
				case []byte:
					// ugorji/codec decodes a msgpack str into []byte when the
					// destination is *any.
					result = string(v)
				}
			}
			assert.Equal(t, result, "hello:world")
		})
	}
}

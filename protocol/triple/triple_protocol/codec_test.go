// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

func convertMapToInterface(stringMap map[string]string) map[string]any {
	interfaceMap := make(map[string]any)
	for key, value := range stringMap {
		interfaceMap[key] = value
	}
	return interfaceMap
}

func TestCodecRoundTrips(t *testing.T) {
	t.Parallel()
	makeRoundtrip := func(codec Codec) func(string, int64) bool {
		return func(text string, number int64) bool {
			got := pingv1.PingRequest{}
			want := pingv1.PingRequest{Text: text, Number: number}
			data, err := codec.Marshal(&want)
			if err != nil {
				t.Fatal(err)
			}
			err = codec.Unmarshal(data, &got)
			if err != nil {
				t.Fatal(err)
			}
			return proto.Equal(&got, &want)
		}
	}
	if err := quick.Check(makeRoundtrip(&protoBinaryCodec{}), nil /* config */); err != nil {
		t.Error(err)
	}
	if err := quick.Check(makeRoundtrip(&protoJSONCodec{}), nil /* config */); err != nil {
		t.Error(err)
	}
}

func TestStableCodec(t *testing.T) {
	t.Parallel()
	makeRoundtrip := func(codec stableCodec) func(map[string]string) bool {
		return func(input map[string]string) bool {
			initialProto, err := structpb.NewStruct(convertMapToInterface(input))
			if err != nil {
				t.Fatal(err)
			}
			want, err := codec.MarshalStable(initialProto)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 10; i++ {
				roundtripProto := &structpb.Struct{}
				err = codec.Unmarshal(want, roundtripProto)
				if err != nil {
					t.Fatal(err)
				}
				got, err := codec.MarshalStable(roundtripProto)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					return false
				}
			}
			return true
		}
	}
	if err := quick.Check(makeRoundtrip(&protoBinaryCodec{}), nil /* config */); err != nil {
		t.Error(err)
	}
	if err := quick.Check(makeRoundtrip(&protoJSONCodec{}), nil /* config */); err != nil {
		t.Error(err)
	}
}

func TestJSONCodec(t *testing.T) {
	t.Parallel()

	var empty emptypb.Empty
	codec := &protoJSONCodec{name: "json"}
	err := codec.Unmarshal([]byte{}, &empty)
	assert.NotNil(t, err)
	assert.True(
		t,
		strings.Contains(err.Error(), "valid JSON"),
		assert.Sprintf(`error message should explain that "" is not a valid JSON object`),
	)
}

func TestMsgpackCodec(t *testing.T) {
	t.Parallel()

	want := &pingv1.PingRequest{
		Number: 1234,
		Text:   "5678",
	}
	codec := &msgpackCodec{}
	binary, err := codec.Marshal(want)
	assert.Nil(t, err)
	var got pingv1.PingRequest
	err = codec.Unmarshal(binary, &got)
	assert.Nil(t, err)
	assert.Equal(t, got.Number, want.Number)
	assert.Equal(t, got.Text, want.Text)
}

// TestProtoBinaryCodec tests protoBinaryCodec comprehensively
func TestProtoBinaryCodec(t *testing.T) {
	t.Parallel()

	codec := &protoBinaryCodec{}

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, codec.Name(), codecNameProto)
	})

	t.Run("IsBinary", func(t *testing.T) {
		t.Parallel()
		assert.True(t, codec.IsBinary())
	})

	t.Run("Marshal_Success", func(t *testing.T) {
		t.Parallel()
		msg := &pingv1.PingRequest{Text: "hello", Number: 42}
		data, err := codec.Marshal(msg)
		assert.Nil(t, err)
		assert.NotNil(t, data)
	})

	t.Run("Marshal_NonProtoMessage", func(t *testing.T) {
		t.Parallel()
		_, err := codec.Marshal("not a proto message")
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "doesn't implement proto.Message"))
	})

	t.Run("Unmarshal_NonProtoMessage", func(t *testing.T) {
		t.Parallel()
		var s string
		err := codec.Unmarshal([]byte{}, &s)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "doesn't implement proto.Message"))
	})

	t.Run("MarshalStable_Success", func(t *testing.T) {
		t.Parallel()
		msg := &pingv1.PingRequest{Text: "hello", Number: 42}
		data1, err := codec.MarshalStable(msg)
		assert.Nil(t, err)
		data2, err := codec.MarshalStable(msg)
		assert.Nil(t, err)
		assert.True(t, bytes.Equal(data1, data2))
	})

	t.Run("MarshalStable_NonProtoMessage", func(t *testing.T) {
		t.Parallel()
		_, err := codec.MarshalStable("not a proto message")
		assert.NotNil(t, err)
	})
}

// TestProtoJSONCodec tests protoJSONCodec comprehensively
func TestProtoJSONCodec(t *testing.T) {
	t.Parallel()

	codec := &protoJSONCodec{name: codecNameJSON}

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, codec.Name(), codecNameJSON)
	})

	t.Run("IsBinary", func(t *testing.T) {
		t.Parallel()
		assert.False(t, codec.IsBinary())
	})

	t.Run("Marshal_Success", func(t *testing.T) {
		t.Parallel()
		msg := &pingv1.PingRequest{Text: "hello", Number: 42}
		data, err := codec.Marshal(msg)
		assert.Nil(t, err)
		assert.NotNil(t, data)
		assert.True(t, strings.Contains(string(data), "hello"))
	})

	t.Run("Marshal_NonProtoMessage", func(t *testing.T) {
		t.Parallel()
		_, err := codec.Marshal("not a proto message")
		assert.NotNil(t, err)
	})

	t.Run("Unmarshal_Success", func(t *testing.T) {
		t.Parallel()
		jsonData := []byte(`{"text":"hello","number":"42"}`)
		var msg pingv1.PingRequest
		err := codec.Unmarshal(jsonData, &msg)
		assert.Nil(t, err)
		assert.Equal(t, msg.Text, "hello")
	})

	t.Run("Unmarshal_NonProtoMessage", func(t *testing.T) {
		t.Parallel()
		var s string
		err := codec.Unmarshal([]byte(`{}`), &s)
		assert.NotNil(t, err)
	})

	t.Run("Unmarshal_EmptyPayload", func(t *testing.T) {
		t.Parallel()
		var msg pingv1.PingRequest
		err := codec.Unmarshal([]byte{}, &msg)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "valid JSON"))
	})

	t.Run("MarshalStable_Success", func(t *testing.T) {
		t.Parallel()
		msg := &pingv1.PingRequest{Text: "hello", Number: 42}
		data1, err := codec.MarshalStable(msg)
		assert.Nil(t, err)
		data2, err := codec.MarshalStable(msg)
		assert.Nil(t, err)
		assert.True(t, bytes.Equal(data1, data2))
	})

	t.Run("MarshalStable_NonProtoMessage", func(t *testing.T) {
		t.Parallel()
		_, err := codec.MarshalStable("not a proto message")
		assert.NotNil(t, err)
	})
}

// TestHessian2Codec tests hessian2Codec
func TestHessian2Codec(t *testing.T) {
	t.Parallel()

	codec := &hessian2Codec{}

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, codec.Name(), codecNameHessian2)
	})

	t.Run("Marshal_Unmarshal_String", func(t *testing.T) {
		t.Parallel()
		want := "hello world"
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		assert.NotNil(t, data)

		var got any
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, got, want)
	})

	t.Run("Marshal_Unmarshal_Int", func(t *testing.T) {
		t.Parallel()
		want := int32(12345)
		data, err := codec.Marshal(want)
		assert.Nil(t, err)

		var got any
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, got, want)
	})

	t.Run("Marshal_Unmarshal_Map", func(t *testing.T) {
		t.Parallel()
		want := map[any]any{"key": "value", "num": int32(42)}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)

		var got any
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		gotMap, ok := got.(map[any]any)
		assert.True(t, ok)
		assert.Equal(t, gotMap["key"], "value")
	})

	t.Run("Marshal_Unmarshal_Slice", func(t *testing.T) {
		t.Parallel()
		want := []any{"a", "b", "c"}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)

		var got any
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
	})

	t.Run("Unmarshal_EmptyData", func(t *testing.T) {
		t.Parallel()
		var got any
		// Empty data should cause decode to return nil or error
		err := codec.Unmarshal([]byte{}, &got)
		// hessian2 may return nil for empty data, which is acceptable
		if err != nil {
			assert.True(t, strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "nil"))
		}
	})
}

// TestMsgpackCodecExtended tests msgpackCodec with more cases
func TestMsgpackCodecExtended(t *testing.T) {
	t.Parallel()

	codec := &msgpackCodec{}

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, codec.Name(), codecNameMsgPack)
	})

	t.Run("Marshal_Unmarshal_Map", func(t *testing.T) {
		t.Parallel()
		want := map[string]any{"key": "value", "num": int64(42)}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)

		var got map[string]any
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		// msgpack may decode string as []byte
		assert.NotNil(t, got["key"])
		assert.NotNil(t, got["num"])
	})

	t.Run("Marshal_Unmarshal_Slice", func(t *testing.T) {
		t.Parallel()
		want := []string{"a", "b", "c"}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)

		var got []string
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, len(got), 3)
	})

	t.Run("Marshal_Unmarshal_Struct", func(t *testing.T) {
		t.Parallel()
		type testStruct struct {
			Name  string
			Value int
		}
		want := testStruct{Name: "test", Value: 42}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)

		var got testStruct
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, got.Name, want.Name)
		assert.Equal(t, got.Value, want.Value)
	})
}

// TestCodecMap tests codecMap methods
func TestCodecMap(t *testing.T) {
	t.Parallel()

	t.Run("Get_ExistingCodec", func(t *testing.T) {
		t.Parallel()
		protoCodec := &protoBinaryCodec{}
		jsonCodec := &protoJSONCodec{name: codecNameJSON}
		cm := newReadOnlyCodecs(map[string]Codec{
			codecNameProto: protoCodec,
			codecNameJSON:  jsonCodec,
		})

		got := cm.Get(codecNameProto)
		assert.NotNil(t, got)
		assert.Equal(t, got.Name(), codecNameProto)

		got = cm.Get(codecNameJSON)
		assert.NotNil(t, got)
		assert.Equal(t, got.Name(), codecNameJSON)
	})

	t.Run("Get_NonExistingCodec", func(t *testing.T) {
		t.Parallel()
		cm := newReadOnlyCodecs(map[string]Codec{})
		got := cm.Get("nonexistent")
		assert.Nil(t, got)
	})

	t.Run("Protobuf_WithCustomCodec", func(t *testing.T) {
		t.Parallel()
		customProto := &protoBinaryCodec{}
		cm := newReadOnlyCodecs(map[string]Codec{
			codecNameProto: customProto,
		})
		got := cm.Protobuf()
		assert.NotNil(t, got)
		assert.Equal(t, got.Name(), codecNameProto)
	})

	t.Run("Protobuf_WithoutCustomCodec", func(t *testing.T) {
		t.Parallel()
		cm := newReadOnlyCodecs(map[string]Codec{})
		got := cm.Protobuf()
		assert.NotNil(t, got)
		assert.Equal(t, got.Name(), codecNameProto)
	})

	t.Run("Names", func(t *testing.T) {
		t.Parallel()
		cm := newReadOnlyCodecs(map[string]Codec{
			codecNameProto: &protoBinaryCodec{},
			codecNameJSON:  &protoJSONCodec{name: codecNameJSON},
		})
		names := cm.Names()
		assert.Equal(t, len(names), 2)
		// Check both names are present (order may vary)
		hasProto := false
		hasJSON := false
		for _, name := range names {
			if name == codecNameProto {
				hasProto = true
			}
			if name == codecNameJSON {
				hasJSON = true
			}
		}
		assert.True(t, hasProto)
		assert.True(t, hasJSON)
	})

	t.Run("Names_Empty", func(t *testing.T) {
		t.Parallel()
		cm := newReadOnlyCodecs(map[string]Codec{})
		names := cm.Names()
		assert.Equal(t, len(names), 0)
	})
}

// TestErrNotProto tests errNotProto function
func TestErrNotProto(t *testing.T) {
	t.Parallel()

	t.Run("NonProtoType", func(t *testing.T) {
		t.Parallel()
		err := errNotProto("string value")
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "doesn't implement proto.Message"))
	})

	t.Run("IntType", func(t *testing.T) {
		t.Parallel()
		err := errNotProto(123)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "int"))
	})

	t.Run("StructType", func(t *testing.T) {
		t.Parallel()
		type customStruct struct{ Name string }
		err := errNotProto(customStruct{Name: "test"})
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "customStruct"))
	})
}

// TestGetArgType tests getArgType function with all type branches
func TestGetArgType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		// nil cases
		{"nil", nil, "V"},

		// boolean
		{"bool_true", true, "boolean"},
		{"bool_false", false, "boolean"},
		{"bool_slice", []bool{true, false}, "[Z"},

		// byte
		{"byte", byte(1), "byte"},
		{"byte_slice", []byte{1, 2, 3}, "[B"},

		// int8
		{"int8", int8(1), "byte"},
		{"int8_slice", []int8{1, 2, 3}, "[B"},

		// int16
		{"int16", int16(1), "short"},
		{"int16_slice", []int16{1, 2, 3}, "[S"},

		// uint16 (char)
		{"uint16", uint16(1), "char"},
		{"uint16_slice", []uint16{1, 2, 3}, "[C"},

		// int (long)
		{"int", int(1), "long"},
		{"int_slice", []int{1, 2, 3}, "[J"},

		// int32
		{"int32", int32(1), "int"},
		{"int32_slice", []int32{1, 2, 3}, "[I"},

		// int64
		{"int64", int64(1), "long"},
		{"int64_slice", []int64{1, 2, 3}, "[J"},

		// time.Time
		{"time", time.Now(), "java.util.Date"},
		{"time_slice", []time.Time{time.Now()}, "[Ljava.util.Date"},

		// float32
		{"float32", float32(1.0), "float"},
		{"float32_slice", []float32{1.0, 2.0}, "[F"},

		// float64
		{"float64", float64(1.0), "double"},
		{"float64_slice", []float64{1.0, 2.0}, "[D"},

		// string
		{"string", "hello", "java.lang.String"},
		{"string_slice", []string{"a", "b"}, "[Ljava.lang.String;"},

		// hessian.Object slice
		{"hessian_object_slice", []hessian.Object{}, "[Ljava.lang.Object;"},

		// map[any]any
		{"map_any_any", map[any]any{"key": "value"}, "java.util.Map"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getArgType(tc.input)
			assert.Equal(t, result, tc.expected)
		})
	}
}

// TestGetArgTypeComplex tests getArgType with complex types
func TestGetArgTypeComplex(t *testing.T) {
	t.Parallel()

	t.Run("Struct_NonPOJO", func(t *testing.T) {
		t.Parallel()
		type simpleStruct struct{ Name string }
		result := getArgType(simpleStruct{Name: "test"})
		assert.Equal(t, result, "java.lang.Object")
	})

	t.Run("Struct_Pointer_NonPOJO", func(t *testing.T) {
		t.Parallel()
		type simpleStruct struct{ Name string }
		result := getArgType(&simpleStruct{Name: "test"})
		assert.Equal(t, result, "java.lang.Object")
	})

	t.Run("Slice_Of_Struct", func(t *testing.T) {
		t.Parallel()
		type simpleStruct struct{ Name string }
		result := getArgType([]simpleStruct{{Name: "test"}})
		assert.Equal(t, result, "[Ljava.lang.Object;")
	})

	t.Run("Slice_Of_Int", func(t *testing.T) {
		t.Parallel()
		// This goes through the default case for slice of non-struct
		result := getArgType([]any{1, 2, 3})
		assert.Equal(t, result, "java.util.List")
	})

	t.Run("Map_String_Int", func(t *testing.T) {
		t.Parallel()
		result := getArgType(map[string]int{"a": 1})
		assert.Equal(t, result, "java.util.Map")
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()
		result := getArgType([3]int{1, 2, 3})
		assert.Equal(t, result, "java.util.List")
	})

	t.Run("Unknown_Type", func(t *testing.T) {
		t.Parallel()
		// Channel type should return empty string
		ch := make(chan int)
		result := getArgType(ch)
		assert.Equal(t, result, "")
	})
}

// TestReflectResponse tests reflectResponse function
func TestReflectResponse(t *testing.T) {
	t.Parallel()

	t.Run("NilInput", func(t *testing.T) {
		t.Parallel()
		var out any
		err := reflectResponse(nil, &out)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "@in is nil"))
	})

	t.Run("NilOutput", func(t *testing.T) {
		t.Parallel()
		err := reflectResponse("input", nil)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "@out is nil"))
	})

	t.Run("NonPointerOutput", func(t *testing.T) {
		t.Parallel()
		var out string
		err := reflectResponse("input", out)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "@out should be a pointer"))
	})

	t.Run("InterfaceOutput", func(t *testing.T) {
		t.Parallel()
		var out any
		err := reflectResponse("hello", &out)
		assert.Nil(t, err)
		assert.Equal(t, out, "hello")
	})

	t.Run("StringValue", func(t *testing.T) {
		t.Parallel()
		var out any
		err := reflectResponse("test string", &out)
		assert.Nil(t, err)
		assert.Equal(t, out, "test string")
	})

	t.Run("IntValue", func(t *testing.T) {
		t.Parallel()
		var out any
		err := reflectResponse(int32(42), &out)
		assert.Nil(t, err)
		assert.Equal(t, out, int32(42))
	})

	t.Run("SliceValue", func(t *testing.T) {
		t.Parallel()
		inSlice := []int{1, 2, 3}
		outSlice := make([]int, 0)
		err := reflectResponse(inSlice, &outSlice)
		assert.Nil(t, err)
		assert.Equal(t, len(outSlice), 3)
	})

	t.Run("MapValue", func(t *testing.T) {
		t.Parallel()
		inMap := map[string]int{"a": 1, "b": 2}
		outMap := make(map[string]int)
		err := reflectResponse(inMap, &outMap)
		assert.Nil(t, err)
		assert.Equal(t, len(outMap), 2)
	})

	t.Run("PointerInterfaceOutput", func(t *testing.T) {
		t.Parallel()
		var out any
		outPtr := &out
		err := reflectResponse("hello", &outPtr)
		assert.Nil(t, err)
	})
}

// TestCopySlice tests copySlice function
func TestCopySlice(t *testing.T) {
	t.Parallel()

	t.Run("NilSlice", func(t *testing.T) {
		t.Parallel()
		var inSlice []int
		outSlice := make([]int, 0)
		err := copySlice(reflect.ValueOf(inSlice), reflect.ValueOf(&outSlice))
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "@in is nil"))
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		inSlice := []int{1, 2, 3}
		outSlice := make([]int, 0)
		err := copySlice(reflect.ValueOf(inSlice), reflect.ValueOf(&outSlice).Elem())
		assert.Nil(t, err)
		assert.Equal(t, len(outSlice), 3)
	})

	t.Run("Success_WithPointer", func(t *testing.T) {
		t.Parallel()
		inSlice := []int{1, 2, 3}
		outSlice := make([]int, 0)
		err := copySlice(reflect.ValueOf(inSlice), reflect.ValueOf(&outSlice))
		assert.Nil(t, err)
		assert.Equal(t, len(outSlice), 3)
	})

	t.Run("TypeMismatch", func(t *testing.T) {
		t.Parallel()
		inSlice := []string{"a", "b"}
		outSlice := make([]int, 0)
		err := copySlice(reflect.ValueOf(inSlice), reflect.ValueOf(&outSlice).Elem())
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "can not assign"))
	})
}

// TestCopyMap tests copyMap function
func TestCopyMap(t *testing.T) {
	t.Parallel()

	t.Run("NilMap", func(t *testing.T) {
		t.Parallel()
		var inMap map[string]int
		outMap := make(map[string]int)
		err := copyMap(reflect.ValueOf(inMap), reflect.ValueOf(&outMap))
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "@in is nil"))
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		inMap := map[string]int{"a": 1, "b": 2}
		outMap := make(map[string]int)
		err := copyMap(reflect.ValueOf(inMap), reflect.ValueOf(&outMap))
		assert.Nil(t, err)
		assert.Equal(t, len(outMap), 2)
	})

	t.Run("KeyTypeMismatch", func(t *testing.T) {
		t.Parallel()
		inMap := map[int]string{1: "a", 2: "b"}
		outMap := make(map[string]string)
		err := copyMap(reflect.ValueOf(inMap), reflect.ValueOf(&outMap))
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "can not assign"))
	})

	t.Run("ValueTypeMismatch", func(t *testing.T) {
		t.Parallel()
		inMap := map[string]int{"a": 1, "b": 2}
		outMap := make(map[string]string)
		err := copyMap(reflect.ValueOf(inMap), reflect.ValueOf(&outMap))
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "can not assign"))
	})
}

// TestProtoWrapperCodec tests protoWrapperCodec
func TestProtoWrapperCodec(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		innerCodec := &hessian2Codec{}
		codec := newProtoWrapperCodec(innerCodec)
		assert.Equal(t, codec.Name(), codecNameHessian2)
	})

	t.Run("Marshal_SingleMessage", func(t *testing.T) {
		t.Parallel()
		innerCodec := &hessian2Codec{}
		codec := newProtoWrapperCodec(innerCodec)

		data, err := codec.Marshal("hello")
		assert.Nil(t, err)
		assert.NotNil(t, data)
	})

	t.Run("Marshal_MultipleMessages", func(t *testing.T) {
		t.Parallel()
		innerCodec := &hessian2Codec{}
		codec := newProtoWrapperCodec(innerCodec)

		messages := []any{"hello", int32(42)}
		data, err := codec.Marshal(messages)
		assert.Nil(t, err)
		assert.NotNil(t, data)
	})

	t.Run("Unmarshal_SingleMessage", func(t *testing.T) {
		t.Parallel()
		innerCodec := &hessian2Codec{}
		codec := newProtoWrapperCodec(innerCodec)

		// First marshal
		data, err := codec.Marshal("hello")
		assert.Nil(t, err)

		// Then unmarshal
		var result any
		err = codec.Unmarshal(data, &result)
		assert.Nil(t, err)
	})

	t.Run("Unmarshal_MultipleMessages", func(t *testing.T) {
		t.Parallel()
		innerCodec := &hessian2Codec{}
		codec := newProtoWrapperCodec(innerCodec)

		// First marshal
		messages := []any{"hello", int32(42)}
		data, err := codec.Marshal(messages)
		assert.Nil(t, err)

		// Then unmarshal
		var result1 any
		var result2 any
		results := []any{&result1, &result2}
		err = codec.Unmarshal(data, results)
		assert.Nil(t, err)
	})

	t.Run("Unmarshal_ParamCountMismatch", func(t *testing.T) {
		t.Parallel()
		innerCodec := &hessian2Codec{}
		codec := newProtoWrapperCodec(innerCodec)

		// Marshal with 2 messages
		messages := []any{"hello", int32(42)}
		data, err := codec.Marshal(messages)
		assert.Nil(t, err)

		// Try to unmarshal with 1 param
		var result any
		err = codec.Unmarshal(data, &result)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "request params len"))
	})

	t.Run("Unmarshal_InvalidData", func(t *testing.T) {
		t.Parallel()
		innerCodec := &hessian2Codec{}
		codec := newProtoWrapperCodec(innerCodec)

		var result any
		err := codec.Unmarshal([]byte{0xFF, 0xFF}, &result)
		assert.NotNil(t, err)
	})
}

// TestCodecConstants tests codec name constants
func TestCodecConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, codecNameProto, "proto")
	assert.Equal(t, codecNameJSON, "json")
	assert.Equal(t, codecNameHessian2, "hessian2")
	assert.Equal(t, codecNameMsgPack, "msgpack")
	assert.Equal(t, codecNameJSONCharsetUTF8, "json; charset=utf-8")
}

// mockPOJO implements hessian.POJO for testing
type mockPOJO struct {
	Name string
}

func (m mockPOJO) JavaClassName() string {
	return "com.example.MockPOJO"
}

// TestGetArgType_POJO tests getArgType with POJO types
func TestGetArgType_POJO(t *testing.T) {
	t.Parallel()

	t.Run("POJO", func(t *testing.T) {
		t.Parallel()
		pojo := mockPOJO{Name: "test"}
		result := getArgType(pojo)
		assert.Equal(t, result, "com.example.MockPOJO")
	})

	t.Run("POJO_Pointer", func(t *testing.T) {
		t.Parallel()
		pojo := &mockPOJO{Name: "test"}
		result := getArgType(pojo)
		assert.Equal(t, result, "com.example.MockPOJO")
	})
}

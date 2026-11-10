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
	"bytes"
	"strings"
	"testing"
	"testing/quick"
)

import (
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
			for range 10 {
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

	t.Run("roundtrip proto message", func(t *testing.T) {
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
	})

	t.Run("name returns msgpack", func(t *testing.T) {
		t.Parallel()
		codec := &msgpackCodec{}
		assert.Equal(t, codecNameMsgPack, codec.Name())
	})

	t.Run("marshal nil returns empty bytes", func(t *testing.T) {
		t.Parallel()
		codec := &msgpackCodec{}
		data, err := codec.Marshal(nil)
		assert.Nil(t, err)
		assert.Equal(t, []byte{0xc0}, data)
	})

	t.Run("unmarshal into nil returns error", func(t *testing.T) {
		t.Parallel()
		codec := &msgpackCodec{}
		err := codec.Unmarshal([]byte{0xc0}, nil)
		assert.NotNil(t, err)
	})

	t.Run("unmarshal invalid data returns error", func(t *testing.T) {
		t.Parallel()
		codec := &msgpackCodec{}
		var got pingv1.PingRequest
		err := codec.Unmarshal([]byte{0xff, 0xff, 0xff}, &got)
		assert.NotNil(t, err)
	})

	t.Run("roundtrip with empty string", func(t *testing.T) {
		t.Parallel()
		want := &pingv1.PingRequest{Text: "", Number: 0}
		codec := &msgpackCodec{}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got pingv1.PingRequest
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.True(t, proto.Equal(&got, want))
	})
}

func TestHessian2Codec(t *testing.T) {
	t.Parallel()

	t.Run("name returns hessian2", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		assert.Equal(t, codecNameHessian2, codec.Name())
	})

	t.Run("roundtrip string", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := "hello dubbo-go"
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got string
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("roundtrip int32", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := int32(42)
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got int32
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("roundtrip int64", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := int64(9223372036854775807)
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got int64
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("roundtrip bool true", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := true
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got bool
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.True(t, got)
	})

	t.Run("roundtrip bool false", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := false
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got bool
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.False(t, got)
	})

	t.Run("roundtrip byte slice", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := []byte{0x01, 0x02, 0x03, 0x04}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got []byte
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("roundtrip map", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := map[any]any{"key1": "value1", "key2": int64(42)}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		got := make(map[any]any)
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("roundtrip string slice", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := []string{"a", "b", "c"}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got []string
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("marshal nil returns nil bytes", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		data, err := codec.Marshal(nil)
		assert.Nil(t, err)
		assert.Equal(t, []byte{'N'}, data)
	})

	t.Run("unmarshal into non-pointer returns error", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		var got string
		err := codec.Unmarshal([]byte{'N'}, got)
		assert.NotNil(t, err)
	})

	t.Run("unmarshal nil pointer returns error", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		var got *string
		err := codec.Unmarshal([]byte{'N'}, got)
		assert.NotNil(t, err)
	})

	t.Run("unmarshal into nil returns error", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		err := codec.Unmarshal([]byte{'N'}, nil)
		assert.NotNil(t, err)
	})

	t.Run("roundtrip empty string", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := ""
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		var got string
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("roundtrip empty map", func(t *testing.T) {
		t.Parallel()
		codec := &hessian2Codec{}
		want := map[string]string{}
		data, err := codec.Marshal(want)
		assert.Nil(t, err)
		got := make(map[string]string)
		err = codec.Unmarshal(data, &got)
		assert.Nil(t, err)
		assert.Equal(t, len(want), len(got))
	})

	t.Run("implements Codec interface", func(t *testing.T) {
		t.Parallel()
		var _ Codec = (*hessian2Codec)(nil)
	})
}

func TestResolveInnerCodec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		serializeType string
		wantOK        bool
		// wantName is checked only when wantOK is true. Empty means skip the
		// name assertion (kept simple for cases that only care about resolve).
		wantName string
	}{
		{"hessian2", "hessian2", true, "hessian2"},
		{"msgpack", "msgpack", true, "msgpack"},
		{"empty-defaults-hessian2", "", true, "hessian2"},
		// Dubbo Java writes "hessian4" into the wrapper (TripleConstants.HESSIAN4);
		// it denotes the same on-wire Hessian2 encoding and must resolve to the
		// hessian2 codec. Mirrors Java's ReflectionPackableMethod.convertHessianFromWrapper.
		{"hessian4-alias", "hessian4", true, "hessian2"},
		{"unknown", "unknown", false, ""},
		// Bare "hessian" is not a value Dubbo writes; it must be rejected rather
		// than silently normalized, so misbehaving peers surface clearly.
		{"hessian-not-aliased", "hessian", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := resolveInnerCodec(tc.serializeType)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("resolveInnerCodec(%q) err: %v", tc.serializeType, err)
				}
				if s == nil {
					t.Fatalf("got nil serializer")
				}
				if tc.wantName != "" && s.Name() != tc.wantName {
					t.Fatalf("resolveInnerCodec(%q) name = %q, want %q",
						tc.serializeType, s.Name(), tc.wantName)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.serializeType)
				}
			}
		})
	}
}

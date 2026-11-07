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

package definition

import (
	"context"
	"reflect"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/fieldmaskpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition/testdata/fixture"
)

func TestWellKnownProtoJSONSchema(t *testing.T) {
	file := wktFile(t)
	payload := file.Messages().ByName("Payload")
	inner := file.Messages().ByName("Inner")
	anyHolder := file.Messages().ByName("AnyHolder")
	structHolder := file.Messages().ByName("StructHolder")
	valueHolder := file.Messages().ByName("ValueHolder")
	listHolder := file.Messages().ByName("ListHolder")
	nullHolder := file.Messages().ByName("NullHolder")
	require.NotNil(t, payload)
	require.NotNil(t, inner)

	dynamicReq := func(md protoreflect.MessageDescriptor) func() any {
		return func() any { return dynamicpb.NewMessage(md) }
	}
	info := &common.ServiceInfo{InterfaceName: "wktfix.Api", Methods: []common.MethodInfo{
		{Name: "Send", Type: constant.CallUnary, ReqInitFunc: dynamicReq(payload)},
		{Name: "Note", Type: constant.CallUnary, ReqInitFunc: dynamicReq(inner)},
		{Name: "Stamp", Type: constant.CallUnary, ReqInitFunc: func() any { return &timestamppb.Timestamp{} }},
		{Name: "Wrap", Type: constant.CallUnary, ReqInitFunc: func() any { return &wrapperspb.StringValue{} }},
		{Name: "Any", Type: constant.CallUnary, ReqInitFunc: dynamicReq(anyHolder)},
		{Name: "DirectAny", Type: constant.CallUnary, ReqInitFunc: func() any { return &anypb.Any{} }},
		{Name: "Struct", Type: constant.CallUnary, ReqInitFunc: dynamicReq(structHolder)},
		{Name: "Value", Type: constant.CallUnary, ReqInitFunc: dynamicReq(valueHolder)},
		{Name: "List", Type: constant.CallUnary, ReqInitFunc: dynamicReq(listHolder)},
		{Name: "Null", Type: constant.CallUnary, ReqInitFunc: dynamicReq(nullHolder)},
		{Name: "When", Type: constant.CallUnary, ReqInitFunc: dynamicReq(inner)},
		{Name: "Watch", Type: constant.CallServerStream, ReqInitFunc: dynamicReq(payload)},
	}}
	u := common.NewURLWithOptions(common.WithProtocol(constant.TriProtocol), common.WithParamsValue(constant.InterfaceKey, "wktfix.Api"))
	u.SetAttribute(constant.ServiceInfoKey, info)

	def, skipped, err := BuildFromURL(u, reflect.TypeOf(struct{}{}))
	require.NoError(t, err)
	again, skippedAgain, err := BuildFromURL(u, reflect.TypeOf(struct{}{}))
	require.NoError(t, err)
	require.Equal(t, def.Methods, again.Methods)
	require.Equal(t, def.Types, again.Types)
	require.Equal(t, skipped, skippedAgain)

	require.Equal(t, []MethodDefinition{
		{Name: "Note", ParameterTypes: []string{"wktfix.Inner"}, ReturnType: "google.protobuf.Empty"},
		{Name: "Send", ParameterTypes: []string{"wktfix.Payload"}, ReturnType: "google.protobuf.Empty"},
	}, def.Methods)

	got := make(map[string]TypeDefinition, len(def.Types))
	for _, d := range def.Types {
		got[d.Type] = d
	}
	require.Equal(t, map[string]string{
		"ts":         "java.lang.String",
		"dur":        "java.lang.String",
		"mask":       "java.lang.String",
		"str":        "java.lang.String",
		"flag":       "boolean",
		"data":       "bytes",
		"i32":        "int",
		"i64":        "long",
		"u32":        "int",
		"u64":        "long",
		"f32":        "float",
		"f64":        "double",
		"inner":      "wktfix.Inner",
		"times":      "java.util.List<java.lang.String>",
		"counts":     "java.util.List<java.lang.Integer>",
		"byName":     "java.util.Map<java.lang.String,java.lang.String>",
		"byCode":     "java.util.Map<java.lang.Integer,java.lang.String>",
		"chosenTime": "java.lang.String",
	}, got["wktfix.Payload"].Properties)
	require.Equal(t, map[string]string{"when": "java.lang.String"}, got["wktfix.Inner"].Properties)
	require.Equal(t, []string{"java.lang.String"}, got["java.util.List<java.lang.String>"].Items)
	require.Equal(t, []string{"java.lang.Integer"}, got["java.util.List<java.lang.Integer>"].Items)
	require.Equal(t, []string{"java.lang.String", "java.lang.String"}, got["java.util.Map<java.lang.String,java.lang.String>"].Items)
	require.Equal(t, []string{"java.lang.Integer", "java.lang.String"}, got["java.util.Map<java.lang.Integer,java.lang.String>"].Items)
	for _, absent := range []string{
		"google.protobuf.Timestamp", "google.protobuf.Duration", "google.protobuf.FieldMask",
		"google.protobuf.StringValue", "google.protobuf.BytesValue", "google.protobuf.BoolValue",
		"google.protobuf.Int32Value", "google.protobuf.Int64Value", "google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value", "google.protobuf.FloatValue", "google.protobuf.DoubleValue",
		"google.protobuf.Any", "google.protobuf.Struct", "google.protobuf.Value",
		"google.protobuf.ListValue", "google.protobuf.NullValue",
		"wktfix.AnyHolder", "wktfix.StructHolder", "wktfix.ValueHolder", "wktfix.ListHolder", "wktfix.NullHolder",
	} {
		require.NotContains(t, got, absent)
	}

	reasons := make(map[string]string, len(skipped))
	for _, skip := range skipped {
		reasons[skip.Name] = skip.Reason
	}
	require.Equal(t, map[string]string{
		"Stamp":     "google.protobuf.Timestamp has scalar JSON representation java.lang.String, which ServiceDefinition cannot describe as a method body",
		"Wrap":      "google.protobuf.StringValue has scalar JSON representation java.lang.String, which ServiceDefinition cannot describe as a method body",
		"Any":       "google.protobuf.Any cannot be represented: protojson emits @type plus the packed message, which is not a fixed schema",
		"DirectAny": "google.protobuf.Any cannot be represented: protojson emits @type plus the packed message, which is not a fixed schema",
		"Struct":    "google.protobuf.Struct cannot be represented: protojson emits an arbitrary JSON object",
		"Value":     "google.protobuf.Value cannot be represented: protojson emits any JSON value",
		"List":      "google.protobuf.ListValue cannot be represented: protojson emits an arbitrary JSON array",
		"Null":      "google.protobuf.NullValue cannot be represented: protojson emits JSON null",
		"When":      "google.protobuf.Timestamp has scalar JSON representation java.lang.String, which ServiceDefinition cannot describe as a method body",
		"Watch":     "streaming is not supported",
	}, reasons)

	const accepted = `{
		"ts":"2017-01-15T01:30:15.01Z",
		"dur":"1.000340012s",
		"mask":"user.displayName,photo",
		"str":"hello",
		"flag":true,
		"data":"YQ==",
		"i32":1,
		"i64":64,
		"u32":2,
		"u64":65,
		"f32":1.5,
		"f64":2.5,
		"inner":{"when":"2017-01-15T01:30:15.01Z"},
		"times":["2017-01-15T01:30:15.01Z"],
		"counts":[3,4],
		"byName":{"a":"1s"},
		"byCode":{"7":"seven"},
		"chosenTime":"2017-01-15T01:30:15.01Z"
	}`
	first := dynamicpb.NewMessage(payload)
	require.NoError(t, protojson.Unmarshal([]byte(accepted), first))
	raw, err := protojson.Marshal(first)
	require.NoError(t, err)
	second := dynamicpb.NewMessage(payload)
	require.NoError(t, protojson.Unmarshal(raw, second))
	require.True(t, proto.Equal(first, second))

	// protojson.Unmarshal accepts both numeric and decimal-string int64 inputs.
	i64 := dynamicpb.NewMessage(payload)
	require.NoError(t, protojson.Unmarshal([]byte(`{"i64":"64","u64":"65"}`), i64))

	rejected := []string{
		`{"ts":{"seconds":"1","nanos":1}}`,
		`{"dur":{"seconds":"1","nanos":340012}}`,
		`{"mask":{"paths":["user.displayName"]}}`,
		`{"str":{"value":"hello"}}`,
		`{"flag":{"value":true}}`,
		`{"data":{"value":"YQ=="}}`,
		`{"i32":{"value":1}}`,
		`{"i64":{"value":"64"}}`,
		`{"f64":{"value":2.5}}`,
		`{"inner":{"when":{"seconds":"1","nanos":1}}}`,
		`{"times":[{"seconds":"1","nanos":1}]}`,
		`{"counts":[{"value":3}]}`,
		`{"byName":{"a":{"seconds":"1","nanos":0}}}`,
		`{"byCode":{"7":{"value":"seven"}}}`,
		`{"chosenTime":{"seconds":"1","nanos":1}}`,
	}
	for _, doc := range rejected {
		unmarshalErr := protojson.Unmarshal([]byte(doc), dynamicpb.NewMessage(payload))
		require.Error(t, unmarshalErr, doc)
	}

	// Scalar bodies are valid protojson, but ServiceDefinition cannot describe their named method types.
	require.NoError(t, protojson.Unmarshal([]byte(`"2017-01-15T01:30:15.01Z"`), &timestamppb.Timestamp{}))
	require.Error(t, protojson.Unmarshal([]byte(`{"seconds":"1","nanos":1}`), &timestamppb.Timestamp{}))
	require.NoError(t, protojson.Unmarshal([]byte(`"hello"`), &wrapperspb.StringValue{}))
	require.Error(t, protojson.Unmarshal([]byte(`{"value":"hello"}`), &wrapperspb.StringValue{}))
	require.NoError(t, protojson.Unmarshal([]byte(`{}`), &emptypb.Empty{}))

	body, err := anypb.New(&timestamppb.Timestamp{})
	require.NoError(t, err)
	anyJSON, err := protojson.Marshal(body)
	require.NoError(t, err)
	require.Contains(t, string(anyJSON), "@type")
	require.Error(t, protojson.Unmarshal([]byte(`{"typeUrl":"type.googleapis.com/google.protobuf.Timestamp","value":""}`), &anypb.Any{}))

	st, err := structpb.NewStruct(map[string]any{"n": float64(1)})
	require.NoError(t, err)
	structJSON, err := protojson.Marshal(st)
	require.NoError(t, err)
	require.JSONEq(t, `{"n":1}`, string(structJSON))
	require.NoError(t, protojson.Unmarshal([]byte(`{"n":1}`), &structpb.Struct{}))
	number := &structpb.Value{}
	require.NoError(t, protojson.Unmarshal([]byte(`1`), number))
	require.InDelta(t, 1, number.GetNumberValue(), 1e-9)
	// The proto oneof layout is a different JSON object, not the number protojson emits.
	asObject := &structpb.Value{}
	require.NoError(t, protojson.Unmarshal([]byte(`{"numberValue":1}`), asObject))
	require.NotNil(t, asObject.GetStructValue())
	require.Zero(t, asObject.GetNumberValue())
	require.Error(t, protojson.Unmarshal([]byte(`{"values":[1,"x"]}`), &structpb.ListValue{}))
	require.NoError(t, protojson.Unmarshal([]byte(`[1,"x"]`), &structpb.ListValue{}))
}

func TestSkippedSchemaDoesNotLeakStagedTypes(t *testing.T) {
	file := wktFile(t)
	node := file.Messages().ByName("Node")
	solo := file.Messages().ByName("Solo")
	inner := file.Messages().ByName("Inner")
	req := func(md protoreflect.MessageDescriptor) func() any {
		return func() any { return dynamicpb.NewMessage(md) }
	}
	u := common.NewURLWithOptions(common.WithProtocol(constant.TriProtocol), common.WithParamsValue(constant.InterfaceKey, "wktfix.Api"))
	u.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{InterfaceName: "wktfix.Api", Methods: []common.MethodInfo{
		{Name: "Note", Type: constant.CallUnary, ReqInitFunc: req(inner)},
		{Name: "Loop", Type: constant.CallUnary, ReqInitFunc: req(node)},
		{Name: "Leak", Type: constant.CallUnary, ReqInitFunc: req(solo)},
	}})
	def, skipped, err := BuildFromURL(u, reflect.TypeOf(struct{}{}))
	require.NoError(t, err)
	require.Equal(t, []MethodDefinition{{
		Name: "Note", ParameterTypes: []string{"wktfix.Inner"}, ReturnType: "google.protobuf.Empty",
	}}, def.Methods)
	require.Equal(t, []SkippedMethod{
		{Name: "Leak", Reason: "google.protobuf.Any cannot be represented: protojson emits @type plus the packed message, which is not a fixed schema"},
		{Name: "Loop", Reason: "google.protobuf.Any cannot be represented: protojson emits @type plus the packed message, which is not a fixed schema"},
	}, skipped)
	got := make(map[string]TypeDefinition, len(def.Types))
	for _, d := range def.Types {
		got[d.Type] = d
	}
	require.Contains(t, got, "wktfix.Inner")
	require.Contains(t, got, "google.protobuf.Empty")
	for _, absent := range []string{"wktfix.Node", "java.util.List<wktfix.Node>", "wktfix.Solo", "google.protobuf.Any"} {
		require.NotContains(t, got, absent)
	}
}

type sayHelloHandler struct{}

func (*sayHelloHandler) SayHello(context.Context, *fixture.Request) (*fixture.Request, error) {
	return &fixture.Request{}, nil
}

func TestProtobufFallbackSwapsFirstRuneOnly(t *testing.T) {
	handler := reflect.TypeOf(&sayHelloHandler{})
	cases := []struct {
		name    string
		method  string
		want    bool
		wantRet string
	}{
		{name: "lower initial", method: "sayHello", want: true, wantRet: "fixture.Request"},
		{name: "exact", method: "SayHello", want: true, wantRet: "fixture.Request"},
		{name: "not a first-rune swap", method: "sayhello", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := common.NewURLWithOptions(common.WithProtocol(constant.TriProtocol), common.WithParamsValue(constant.InterfaceKey, "local.Unregistered"))
			u.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{
				InterfaceName: "local.Unregistered",
				Methods: []common.MethodInfo{{
					Name:        tc.method,
					Type:        constant.CallUnary,
					ReqInitFunc: func() any { return &fixture.Request{} },
				}},
			})
			def, skipped, err := BuildFromURL(u, handler)
			require.NoError(t, err)
			if !tc.want {
				require.Empty(t, def.Methods)
				require.Equal(t, []SkippedMethod{{Name: tc.method, Reason: "protobuf request or response descriptor unavailable"}}, skipped)
				return
			}
			require.Empty(t, skipped)
			require.Equal(t, []MethodDefinition{{
				Name:           tc.method,
				ParameterTypes: []string{"fixture.Request"},
				ReturnType:     tc.wantRet,
			}}, def.Methods)
		})
	}

	t.Run("streaming stays skipped", func(t *testing.T) {
		u := common.NewURLWithOptions(common.WithProtocol(constant.TriProtocol), common.WithParamsValue(constant.InterfaceKey, "local.Unregistered"))
		u.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{
			InterfaceName: "local.Unregistered",
			Methods: []common.MethodInfo{{
				Name: "sayHello",
				Type: constant.CallServerStream,
			}},
		})
		def, skipped, err := BuildFromURL(u, handler)
		require.NoError(t, err)
		require.Empty(t, def.Methods)
		require.Equal(t, []SkippedMethod{{Name: "sayHello", Reason: "streaming is not supported"}}, skipped)
	})

	t.Run("registered descriptor blocks fallback", func(t *testing.T) {
		u := common.NewURLWithOptions(common.WithProtocol(constant.TriProtocol), common.WithParamsValue(constant.InterfaceKey, "fixture.Greeter"))
		u.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{
			InterfaceName: "fixture.Greeter",
			Methods: []common.MethodInfo{
				{Name: "Greet", Type: constant.CallUnary, ReqInitFunc: func() any { return &fixture.Request{} }},
				{Name: "sayHello", Type: constant.CallUnary, ReqInitFunc: func() any { return &fixture.Request{} }},
			},
		})
		def, skipped, err := BuildFromURL(u, handler)
		require.NoError(t, err)
		require.Equal(t, []MethodDefinition{{
			Name:           "Greet",
			ParameterTypes: []string{"fixture.Request"},
			ReturnType:     "google.protobuf.Empty",
		}}, def.Methods)
		require.Equal(t, []SkippedMethod{{Name: "sayHello", Reason: "protobuf request or response descriptor unavailable"}}, skipped)
	})
}

var (
	wktOnce sync.Once
	wktDesc protoreflect.FileDescriptor
	wktErr  error
)

func wktFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	wktOnce.Do(func() {
		fd, err := protodesc.NewFile(wktFileProto(), protoregistry.GlobalFiles)
		if err != nil {
			wktErr = err
			return
		}
		if err := protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
			wktErr = err
			return
		}
		wktDesc = fd
	})
	require.NoError(t, wktErr)
	return wktDesc
}

func wktFileProto() *descriptorpb.FileDescriptorProto {
	opt := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	rep := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	msg := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	enum := descriptorpb.FieldDescriptorProto_TYPE_ENUM
	str := descriptorpb.FieldDescriptorProto_TYPE_STRING
	i32 := descriptorpb.FieldDescriptorProto_TYPE_INT32
	field := func(name, jsonName string, num int32, label descriptorpb.FieldDescriptorProto_Label, typ descriptorpb.FieldDescriptorProto_Type, typeName string) *descriptorpb.FieldDescriptorProto {
		f := &descriptorpb.FieldDescriptorProto{
			Name:     proto.String(name),
			JsonName: proto.String(jsonName),
			Number:   proto.Int32(num),
			Label:    label.Enum(),
			Type:     typ.Enum(),
		}
		if typeName != "" {
			f.TypeName = proto.String(typeName)
		}
		return f
	}
	mapEntry := func(name, valueType string, valueKind descriptorpb.FieldDescriptorProto_Type) *descriptorpb.DescriptorProto {
		return &descriptorpb.DescriptorProto{
			Name: proto.String(name),
			Field: []*descriptorpb.FieldDescriptorProto{
				field("key", "key", 1, opt, i32, ""),
				field("value", "value", 2, opt, valueKind, valueType),
			},
			Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
		}
	}
	byName := mapEntry("ByNameEntry", ".google.protobuf.Duration", msg)
	byName.Field[0].Type = str.Enum()
	byCode := mapEntry("ByCodeEntry", ".google.protobuf.StringValue", msg)
	holder := func(name, typeName string, kind descriptorpb.FieldDescriptorProto_Type) *descriptorpb.DescriptorProto {
		return &descriptorpb.DescriptorProto{
			Name:  proto.String(name),
			Field: []*descriptorpb.FieldDescriptorProto{field("body", "body", 1, opt, kind, typeName)},
		}
	}
	method := func(name, in string) *descriptorpb.MethodDescriptorProto {
		return &descriptorpb.MethodDescriptorProto{
			Name:       proto.String(name),
			InputType:  proto.String(in),
			OutputType: proto.String(".google.protobuf.Empty"),
		}
	}
	chosen := field("chosen_time", "chosenTime", 18, opt, msg, ".google.protobuf.Timestamp")
	chosen.OneofIndex = proto.Int32(0)
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("wktfix.proto"),
		Package: proto.String("wktfix"),
		Syntax:  proto.String("proto3"),
		Dependency: []string{
			"google/protobuf/timestamp.proto",
			"google/protobuf/duration.proto",
			"google/protobuf/field_mask.proto",
			"google/protobuf/wrappers.proto",
			"google/protobuf/any.proto",
			"google/protobuf/struct.proto",
			"google/protobuf/empty.proto",
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Inner"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("when", "when", 1, opt, msg, ".google.protobuf.Timestamp"),
				},
			},
			{
				Name: proto.String("Payload"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("ts", "ts", 1, opt, msg, ".google.protobuf.Timestamp"),
					field("dur", "dur", 2, opt, msg, ".google.protobuf.Duration"),
					field("mask", "mask", 3, opt, msg, ".google.protobuf.FieldMask"),
					field("str", "str", 4, opt, msg, ".google.protobuf.StringValue"),
					field("flag", "flag", 5, opt, msg, ".google.protobuf.BoolValue"),
					field("data", "data", 6, opt, msg, ".google.protobuf.BytesValue"),
					field("i32", "i32", 7, opt, msg, ".google.protobuf.Int32Value"),
					field("i64", "i64", 8, opt, msg, ".google.protobuf.Int64Value"),
					field("u32", "u32", 9, opt, msg, ".google.protobuf.UInt32Value"),
					field("u64", "u64", 10, opt, msg, ".google.protobuf.UInt64Value"),
					field("f32", "f32", 11, opt, msg, ".google.protobuf.FloatValue"),
					field("f64", "f64", 12, opt, msg, ".google.protobuf.DoubleValue"),
					field("inner", "inner", 13, opt, msg, ".wktfix.Inner"),
					field("times", "times", 14, rep, msg, ".google.protobuf.Timestamp"),
					field("counts", "counts", 15, rep, msg, ".google.protobuf.Int32Value"),
					field("by_name", "byName", 16, rep, msg, ".wktfix.Payload.ByNameEntry"),
					field("by_code", "byCode", 17, rep, msg, ".wktfix.Payload.ByCodeEntry"),
					chosen,
				},
				NestedType: []*descriptorpb.DescriptorProto{byName, byCode},
				OneofDecl:  []*descriptorpb.OneofDescriptorProto{{Name: proto.String("choice")}},
			},
			holder("AnyHolder", ".google.protobuf.Any", msg),
			holder("StructHolder", ".google.protobuf.Struct", msg),
			holder("ValueHolder", ".google.protobuf.Value", msg),
			holder("ListHolder", ".google.protobuf.ListValue", msg),
			holder("NullHolder", ".google.protobuf.NullValue", enum),
			{
				Name: proto.String("Node"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("children", "children", 1, rep, msg, ".wktfix.Node"),
					field("bad", "bad", 2, opt, msg, ".google.protobuf.Any"),
				},
			},
			{
				Name: proto.String("Solo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("label", "label", 1, opt, str, ""),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Api"),
			Method: []*descriptorpb.MethodDescriptorProto{
				method("Send", ".wktfix.Payload"),
				method("Note", ".wktfix.Inner"),
				method("Stamp", ".google.protobuf.Timestamp"),
				method("Wrap", ".google.protobuf.StringValue"),
				method("Any", ".wktfix.AnyHolder"),
				method("DirectAny", ".google.protobuf.Any"),
				method("Struct", ".wktfix.StructHolder"),
				method("Value", ".wktfix.ValueHolder"),
				method("List", ".wktfix.ListHolder"),
				method("Null", ".wktfix.NullHolder"),
				{
					Name:       proto.String("When"),
					InputType:  proto.String(".wktfix.Inner"),
					OutputType: proto.String(".google.protobuf.Timestamp"),
				},
				method("Loop", ".wktfix.Node"),
				{
					Name:       proto.String("Leak"),
					InputType:  proto.String(".wktfix.Solo"),
					OutputType: proto.String(".google.protobuf.Any"),
				},
			},
		}},
	}
}

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
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition/testdata/fixture"
)

func TestProtobufServiceDefinition(t *testing.T) {
	u := common.NewURLWithOptions(common.WithProtocol(constant.TriProtocol), common.WithParamsValue(constant.InterfaceKey, "fixture.Greeter"), common.WithParamsValue(constant.ApplicationKey, "demo"), common.WithParamsValue(constant.VersionKey, "1"), common.WithParamsValue(constant.GroupKey, "group"))
	u.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{InterfaceName: "fixture.Greeter", Methods: []common.MethodInfo{
		{Name: "Greet", Type: constant.CallUnary, ReqInitFunc: func() any { return &fixture.Request{} }},
		{Name: "Watch", Type: constant.CallServerStream},
		{Name: "WrongInput", Type: constant.CallUnary, ReqInitFunc: func() any { return &emptypb.Empty{} }},
	}})
	def, skipped, err := BuildFromURL(u, reflect.TypeOf(struct{}{}))
	require.NoError(t, err)
	require.Equal(t, "fixture.Greeter", def.CanonicalName)
	require.Equal(t, "demo", def.Parameters[constant.ApplicationKey])
	require.Equal(t, "1", def.Parameters[constant.VersionKey])
	require.Equal(t, "group", def.Parameters[constant.GroupKey])
	require.Equal(t, []MethodDefinition{{Name: "Greet", ParameterTypes: []string{"fixture.Request"}, ReturnType: "google.protobuf.Empty"}}, def.Methods)
	require.Len(t, skipped, 2)
	got := make(map[string]TypeDefinition)
	for _, d := range def.Types {
		got[d.Type] = d
	}
	require.Equal(t, map[string]string{
		"displayName": "java.lang.String", "count": "int", "total": "long", "active": "boolean", "data": "bytes", "ratio": "double", "score": "float", "status": "fixture.Status", "children": "java.util.List<fixture.Request.Nested>", "lookup": "java.util.Map<java.lang.String,fixture.Request.Nested>", "text": "java.lang.String", "nested": "fixture.Request.Nested", "timestamp": "google.protobuf.Timestamp", "wrapper": "google.protobuf.StringValue", "next": "fixture.Request",
	}, got["fixture.Request"].Properties)
	require.Equal(t, map[string]string{"value": "java.lang.String"}, got["fixture.Request.Nested"].Properties)
	require.Equal(t, []string{"UNKNOWN", "READY"}, got["fixture.Status"].Enums)
	require.Equal(t, []string{"fixture.Request.Nested"}, got["java.util.List<fixture.Request.Nested>"].Items)
	require.Equal(t, []string{"java.lang.String", "fixture.Request.Nested"}, got["java.util.Map<java.lang.String,fixture.Request.Nested>"].Items)
	require.Equal(t, map[string]string{"seconds": "long", "nanos": "int"}, got["google.protobuf.Timestamp"].Properties)
	require.Equal(t, map[string]string{"value": "java.lang.String"}, got["google.protobuf.StringValue"].Properties)
	require.Contains(t, got, "google.protobuf.Empty")
}

func TestProtobufContainerElementsAreBoxed(t *testing.T) {
	// FileDescriptorProto.public_dependency is "repeated int32". A Java generic
	// cannot hold a primitive, so the element must be the boxed type.
	c := newTypeCollector()
	c.protobufMessage((&descriptorpb.FileDescriptorProto{}).ProtoReflect().Descriptor())
	props := c.defs["google.protobuf.FileDescriptorProto"].Properties
	require.Equal(t, "java.util.List<java.lang.Integer>", props["publicDependency"])
	require.Equal(t, []string{"java.lang.Integer"}, c.defs["java.util.List<java.lang.Integer>"].Items)
	require.Equal(t, "java.util.List<java.lang.String>", props["dependency"])
}

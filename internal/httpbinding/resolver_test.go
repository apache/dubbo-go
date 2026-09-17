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

package httpbinding

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestResolveMethod(t *testing.T) {
	rule := &annotations.HttpRule{
		Pattern:      &annotations.HttpRule_Patch{Patch: "/v1/{name=publishers/*/books/*}:update"},
		Body:         "book",
		ResponseBody: "book",
		AdditionalBindings: []*annotations.HttpRule{{
			Pattern: &annotations.HttpRule_Get{Get: "/v1/shelves/{shelf}/books/{book.id}"},
		}},
	}
	bindings, err := ResolveMethod(newTestMethod(t, rule, false))
	require.NoError(t, err)
	require.Equal(t, []HTTPBinding{
		{RPC: "httpbinding.test.Library.GetBook", Method: http.MethodPatch, PathTemplate: "/v1/{name=publishers/*/books/*}:update", Body: "book", ResponseBody: "book", PathFields: []string{"name"}},
		{RPC: "httpbinding.test.Library.GetBook", Method: http.MethodGet, PathTemplate: "/v1/shelves/{shelf}/books/{book.id}", PathFields: []string{"shelf", "book.id"}, Index: 1},
	}, bindings)

	bindings, err = ResolveMethod(newTestMethod(t, nil, false))
	require.NoError(t, err)
	require.Empty(t, bindings)
}

func TestResolvePattern(t *testing.T) {
	tests := []struct {
		rule   *annotations.HttpRule
		method string
	}{
		{&annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: "/v1"}}, http.MethodGet},
		{&annotations.HttpRule{Pattern: &annotations.HttpRule_Put{Put: "/v1"}}, http.MethodPut},
		{&annotations.HttpRule{Pattern: &annotations.HttpRule_Post{Post: "/v1"}}, http.MethodPost},
		{&annotations.HttpRule{Pattern: &annotations.HttpRule_Delete{Delete: "/v1"}}, http.MethodDelete},
		{&annotations.HttpRule{Pattern: &annotations.HttpRule_Patch{Patch: "/v1"}}, http.MethodPatch},
	}
	for _, test := range tests {
		method, path, err := resolvePattern(test.rule)
		require.NoError(t, err)
		require.Equal(t, test.method, method)
		require.Equal(t, "/v1", path)
	}
}

func TestCanonicalRouteKeyNormalizesPathVariables(t *testing.T) {
	require.Equal(t,
		"GET /v1/books/*:publish",
		CanonicalRouteKey("get", "/v1/books/{book_id=*}:publish"),
	)
	require.Equal(t,
		CanonicalRouteKey("GET", "/v1/books/{name}"),
		CanonicalRouteKey("GET", "/v1/books/{id=*}"),
	)
	require.NotEqual(t,
		CanonicalRouteKey("GET", "/v1/books/{name}"),
		CanonicalRouteKey("GET", "/v1/books/{id=publishers/*/books/*}"),
	)
	require.Equal(t,
		CanonicalRouteKey("GET", "/v1/books/{id=published}"),
		CanonicalRouteKey("GET", "/v1/books/published"),
	)
	require.Equal(t,
		CanonicalRouteKey("GET", "/v1/books/{id=publishers/*/books/*}"),
		CanonicalRouteKey("GET", "/v1/books/publishers/*/books/*"),
	)
	require.Equal(t,
		CanonicalRouteKey("GET", "/v1/books/{id=**}"),
		CanonicalRouteKey("GET", "/v1/books/**"),
	)
}

func TestResolveMethodRejectsInvalidRules(t *testing.T) {
	valid := func(path string) *annotations.HttpRule {
		return &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: path}}
	}
	tests := []struct {
		name      string
		rule      *annotations.HttpRule
		streaming bool
		want      string
	}{
		{"custom method", &annotations.HttpRule{Pattern: &annotations.HttpRule_Custom{Custom: &annotations.CustomHttpPattern{Kind: "REPORT", Path: "/v1"}}}, false, "custom HTTP method"},
		{"missing path field", valid("/v1/{missing}"), false, "invalid path field"},
		{"repeated path field", valid("/v1/{ids}"), false, "must be a non-repeated primitive"},
		{"message path field", valid("/v1/{book}"), false, "must be a non-repeated primitive"},
		{"missing body field", &annotations.HttpRule{Pattern: &annotations.HttpRule_Post{Post: "/v1"}, Body: "missing"}, false, "invalid body field"},
		{"body path overlap", &annotations.HttpRule{Pattern: &annotations.HttpRule_Post{Post: "/v1/{name}"}, Body: "name"}, false, "overlaps path field"},
		{"GET body", &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: "/v1"}, Body: "book"}, false, "GET must not specify"},
		{"DELETE body", &annotations.HttpRule{Pattern: &annotations.HttpRule_Delete{Delete: "/v1"}, Body: "book"}, false, "DELETE must not specify"},
		{"missing response field", &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: "/v1"}, ResponseBody: "missing"}, false, "invalid response_body field"},
		{"nested additional binding", &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: "/v1"}, AdditionalBindings: []*annotations.HttpRule{{Pattern: &annotations.HttpRule_Get{Get: "/v2"}, AdditionalBindings: []*annotations.HttpRule{valid("/v3")}}}}, false, "nested additional_bindings"},
		{"duplicate binding", &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: "/v1"}, AdditionalBindings: []*annotations.HttpRule{valid("/v1")}}, false, "duplicate HTTP binding"},
		{"streaming method", valid("/v1"), true, "is streaming"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveMethod(newTestMethod(t, test.rule, test.streaming))
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestResolveMethodAllowsNestedBodyAndResponseFields(t *testing.T) {
	rule := &annotations.HttpRule{
		Pattern:      &annotations.HttpRule_Post{Post: "/v1/{name}"},
		Body:         "book.id",
		ResponseBody: "book.id",
	}
	bindings, err := ResolveMethod(newTestMethod(t, rule, false))
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	require.Equal(t, "book.id", bindings[0].Body)
	require.Equal(t, "book.id", bindings[0].ResponseBody)
}

func newTestMethod(t *testing.T, rule *annotations.HttpRule, streaming bool) protoreflect.MethodDescriptor {
	t.Helper()
	options := &descriptorpb.MethodOptions{}
	if rule != nil {
		proto.SetExtension(options, annotations.E_Http, rule)
	}
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       proto.String("httpbinding/test.proto"),
		Package:    proto.String("httpbinding.test"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/api/annotations.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Book"), Field: []*descriptorpb.FieldDescriptorProto{testField("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false)}},
			{Name: proto.String("Request"), Field: []*descriptorpb.FieldDescriptorProto{
				testField("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false),
				testField("shelf", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false),
				testField("book", 3, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".httpbinding.test.Book", false),
				testField("ids", 4, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", true),
			}},
			{Name: proto.String("Response"), Field: []*descriptorpb.FieldDescriptorProto{testField("book", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".httpbinding.test.Book", false)}},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Library"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:            proto.String("GetBook"),
				InputType:       proto.String(".httpbinding.test.Request"),
				OutputType:      proto.String(".httpbinding.test.Response"),
				ServerStreaming: proto.Bool(streaming),
				Options:         options,
			}},
		}},
	}, protoregistry.GlobalFiles)
	require.NoError(t, err)
	return file.Services().Get(0).Methods().Get(0)
}

func testField(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type, typeName string, repeated bool) *descriptorpb.FieldDescriptorProto {
	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	if repeated {
		label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	}
	field := &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Label: label.Enum(), Type: kind.Enum()}
	if typeName != "" {
		field.TypeName = proto.String(typeName)
	}
	return field
}

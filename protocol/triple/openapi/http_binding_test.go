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

package openapi

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

import (
	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type openAPIHTTPBook struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type openAPIHTTPRequest struct {
	Book     *openAPIHTTPBook `json:"book"`
	PageSize int32            `json:"pageSize"`
	Filter   string           `json:"filter"`
}

type openAPIHTTPResponse struct {
	Book *openAPIHTTPBook `json:"book"`
}

var registerOpenAPIHTTPDescriptor sync.Once

func TestDefinitionResolver_ResolveHTTPBindings(t *testing.T) {
	registerOpenAPIHTTPDescriptor.Do(func() {
		file, err := protodesc.NewFile(openAPIHTTPFileDescriptor(), protoregistry.GlobalFiles)
		if err != nil {
			t.Fatalf("build HTTP descriptor: %v", err)
		}
		if err := protoregistry.GlobalFiles.RegisterFile(file); err != nil {
			t.Fatalf("register HTTP descriptor: %v", err)
		}
	})

	resolver := NewDefinitionResolver(global.DefaultOpenAPIConfig(), true)
	openAPI := resolver.Resolve("triple.openapi.test.Library", &serviceInfo{Methods: []serviceMethodInfo{{
		Name:        "UpdateBook",
		ReqInitFunc: func() any { return &openAPIHTTPRequest{} },
		Meta:        map[string]any{"response.type": reflect.TypeFor[openAPIHTTPResponse]()},
	}}})

	pathItem := openAPI.Paths["/v1/books/{book.id}"]
	if pathItem == nil {
		t.Fatalf("HTTP path was not generated: %v", openAPI.Paths)
	}
	if pathItem.Patch == nil || pathItem.Get == nil {
		t.Fatalf("expected PATCH and GET operations: %#v", pathItem)
	}
	if pathItem.Patch.OperationId != "triple.openapi.test.Library.UpdateBook" {
		t.Errorf("primary operationId = %q", pathItem.Patch.OperationId)
	}
	if pathItem.Get.OperationId != "triple.openapi.test.Library.UpdateBook.binding1" {
		t.Errorf("additional operationId = %q", pathItem.Get.OperationId)
	}
	if pathItem.Extensions["x-google-path-template"] != "/v1/books/{book.id}" {
		t.Errorf("path extension = %v", pathItem.Extensions)
	}
	if pathItem.Patch.RequestBody == nil {
		t.Fatal("PATCH requestBody is nil")
	}
	if len(pathItem.Patch.Parameters) != 3 {
		t.Fatalf("PATCH parameters = %d, want 3", len(pathItem.Patch.Parameters))
	}
	if pathItem.Patch.Parameters[0].In != "path" || !pathItem.Patch.Parameters[0].Required {
		t.Errorf("path parameter = %#v", pathItem.Patch.Parameters[0])
	}
	if pathItem.Get.RequestBody != nil {
		t.Error("GET additional binding should not have a request body")
	}
	if pathItem.Get.Responses["200"].Content["application/json"].Schema == nil {
		t.Error("response_body schema is nil")
	}
	if _, ok := openAPI.Components.Schemas[runtimeErrorSchemaName]; !ok {
		t.Errorf("missing %q error schema", runtimeErrorSchemaName)
	}
}

func TestDefinitionResolver_UsesCanonicalServiceNameForAlias(t *testing.T) {
	registerOpenAPIHTTPDescriptor.Do(func() {
		file, err := protodesc.NewFile(openAPIHTTPFileDescriptor(), protoregistry.GlobalFiles)
		require.NoError(t, err)
		require.NoError(t, protoregistry.GlobalFiles.RegisterFile(file))
	})

	resolver := NewDefinitionResolver(global.DefaultOpenAPIConfig(), true)
	openAPI, err := resolver.ResolveWithError("alias.Library", &serviceInfo{
		InterfaceName: "triple.openapi.test.Library",
		Methods: []serviceMethodInfo{{
			Name:        "UpdateBook",
			ReqInitFunc: func() any { return &openAPIHTTPRequest{} },
			Meta:        map[string]any{"response.type": reflect.TypeFor[openAPIHTTPResponse]()},
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, openAPI.Paths["/v1/books/{book.id}"])
}

func TestDefinitionResolver_HTTPRulesDisabledKeepsCanonicalRoute(t *testing.T) {
	registerOpenAPIHTTPDescriptor.Do(func() {
		file, err := protodesc.NewFile(openAPIHTTPFileDescriptor(), protoregistry.GlobalFiles)
		if err != nil {
			t.Fatalf("build HTTP descriptor: %v", err)
		}
		if err := protoregistry.GlobalFiles.RegisterFile(file); err != nil {
			t.Fatalf("register HTTP descriptor: %v", err)
		}
	})

	resolver := NewDefinitionResolver(global.DefaultOpenAPIConfig())
	openAPI := resolver.Resolve("triple.openapi.test.Library", &serviceInfo{Methods: []serviceMethodInfo{{
		Name:        "UpdateBook",
		ReqInitFunc: func() any { return &openAPIHTTPRequest{} },
	}}})
	if _, ok := openAPI.Paths["/triple.openapi.test.Library/UpdateBook"]; !ok {
		t.Fatalf("canonical route was not retained: %v", openAPI.Paths)
	}
	if _, ok := openAPI.Paths["/v1/books/{book.id}"]; ok {
		t.Fatal("HTTP rule route should not be generated when disabled")
	}
}

func TestDefinitionResolver_HTTPRulesRejectDuplicateRoutesAcrossMethods(t *testing.T) {
	registerOpenAPIHTTPDescriptor.Do(func() {
		file, err := protodesc.NewFile(openAPIHTTPFileDescriptor(), protoregistry.GlobalFiles)
		if err != nil {
			t.Fatalf("build HTTP descriptor: %v", err)
		}
		if err := protoregistry.GlobalFiles.RegisterFile(file); err != nil {
			t.Fatalf("register HTTP descriptor: %v", err)
		}
	})

	resolver := NewDefinitionResolver(global.DefaultOpenAPIConfig(), true)
	_, err := resolver.ResolveWithError("triple.openapi.test.Library", &serviceInfo{Methods: []serviceMethodInfo{
		{Name: "UpdateBook", ReqInitFunc: func() any { return &openAPIHTTPRequest{} }},
		{Name: "FindBook", ReqInitFunc: func() any { return &openAPIHTTPRequest{} }},
	}})
	if err == nil {
		t.Fatal("expected duplicate HTTP route error")
	}
}

func TestNormalizeHTTPPath(t *testing.T) {
	template := "/v1/{name=publishers/*/books/*}:update"
	if got, want := normalizeHTTPPath(template), "/v1/{name}:update"; got != want {
		t.Fatalf("normalizeHTTPPath() = %q, want %q", got, want)
	}
	if got, want := openAPIPathPattern(template, "name"), "^publishers/[^/]+/books/[^/]+$"; got != want {
		t.Fatalf("openAPIPathPattern() = %q, want %q", got, want)
	}
}

func openAPIHTTPFileDescriptor() *descriptorpb.FileDescriptorProto {
	request := &descriptorpb.DescriptorProto{
		Name: proto.String("UpdateBookRequest"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{Name: proto.String("book"), JsonName: proto.String("book"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".triple.openapi.test.Book")},
			{Name: proto.String("page_size"), JsonName: proto.String("pageSize"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
			{Name: proto.String("filter"), JsonName: proto.String("filter"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
		},
	}
	response := &descriptorpb.DescriptorProto{
		Name:  proto.String("UpdateBookResponse"),
		Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("book"), JsonName: proto.String("book"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".triple.openapi.test.Book")}},
	}
	book := &descriptorpb.DescriptorProto{
		Name:  proto.String("Book"),
		Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("id"), JsonName: proto.String("id"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}, {Name: proto.String("title"), JsonName: proto.String("title"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}},
	}
	rule := &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Patch{Patch: "/v1/books/{book.id}"},
		Body:    "book", ResponseBody: "book",
		AdditionalBindings: []*annotations.HttpRule{{Pattern: &annotations.HttpRule_Get{Get: "/v1/books/{book.id}"}, ResponseBody: "book"}},
	}
	methodOptions := &descriptorpb.MethodOptions{}
	proto.SetExtension(methodOptions, annotations.E_Http, rule)
	return &descriptorpb.FileDescriptorProto{
		Name:        proto.String("triple_openapi_http.proto"),
		Package:     proto.String("triple.openapi.test"),
		Syntax:      proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{book, request, response},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Library"), Method: []*descriptorpb.MethodDescriptorProto{
			{Name: proto.String("UpdateBook"), InputType: proto.String(".triple.openapi.test.UpdateBookRequest"), OutputType: proto.String(".triple.openapi.test.UpdateBookResponse"), Options: methodOptions},
			{Name: proto.String("FindBook"), InputType: proto.String(".triple.openapi.test.UpdateBookRequest"), OutputType: proto.String(".triple.openapi.test.UpdateBookResponse"), Options: methodOptions},
		}}},
	}
}

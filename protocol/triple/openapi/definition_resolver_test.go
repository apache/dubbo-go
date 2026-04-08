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
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

// --- test types for definition resolver ---
type GreetRequest struct {
	Name string `json:"name"`
}

type GreetResponse struct {
	Message string `json:"message"`
}

type EmptyRequest struct{}

// --- Resolve tests ---

func TestDefinitionResolver_Resolve_BasicService(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	cfg.InfoTitle = "Test API"
	cfg.InfoVersion = "1.0.0"
	r := NewDefinitionResolver(cfg)

	info := &serviceInfo{
		Methods: []serviceMethodInfo{
			{
				Name:        "Greet",
				ReqInitFunc: func() any { return &GreetRequest{} },
				Meta: map[string]any{
					"response.type": reflect.TypeOf(GreetResponse{}),
				},
			},
		},
	}

	openAPI := r.Resolve("GreetService", info)

	if openAPI == nil {
		t.Fatal("Resolve should not return nil")
	}
	if openAPI.Info.Title != "Test API" {
		t.Errorf("Info.Title = %q, want %q", openAPI.Info.Title, "Test API")
	}
	if openAPI.Info.Version != "1.0.0" {
		t.Errorf("Info.Version = %q, want %q", openAPI.Info.Version, "1.0.0")
	}
	if openAPI.OpenAPI != model.OpenAPIVersion30 {
		t.Errorf("OpenAPI = %q, want %q", openAPI.OpenAPI, model.OpenAPIVersion30)
	}
}

func TestDefinitionResolver_Resolve_PathGeneration(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	info := &serviceInfo{
		Methods: []serviceMethodInfo{
			{Name: "Greet", ReqInitFunc: func() any { return &GreetRequest{} }},
		},
	}

	openAPI := r.Resolve("GreetService", info)

	expectedPath := "/GreetService/Greet"
	if _, ok := openAPI.Paths[expectedPath]; !ok {
		t.Errorf("path %q not found in Paths, got paths: %v", expectedPath, openAPI.Paths)
	}
}

func TestDefinitionResolver_Resolve_OperationGeneration(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	info := &serviceInfo{
		Methods: []serviceMethodInfo{
			{Name: "Greet", ReqInitFunc: func() any { return &GreetRequest{} }},
		},
	}

	openAPI := r.Resolve("GreetService", info)
	pathItem := openAPI.Paths["/GreetService/Greet"]

	if pathItem.Post == nil {
		t.Fatal("POST operation should be generated")
	}
	if pathItem.Post.OperationId != "GreetService.Greet" {
		t.Errorf("OperationId = %q, want %q", pathItem.Post.OperationId, "GreetService.Greet")
	}
}

func TestDefinitionResolver_Resolve_ComponentsGenerated(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	info := &serviceInfo{
		Methods: []serviceMethodInfo{
			{
				Name:        "Greet",
				ReqInitFunc: func() any { return &GreetRequest{} },
				Meta: map[string]any{
					"response.type": reflect.TypeOf(GreetResponse{}),
				},
			},
		},
	}

	openAPI := r.Resolve("GreetService", info)

	if openAPI.Components == nil {
		t.Fatal("Components should not be nil")
	}
	if len(openAPI.Components.Schemas) == 0 {
		t.Error("Components.Schemas should contain schemas for request/response types")
	}
}

func TestDefinitionResolver_Resolve_DuplicateMethodsDeduped(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	info := &serviceInfo{
		Methods: []serviceMethodInfo{
			{Name: "Greet", ReqInitFunc: func() any { return &GreetRequest{} }},
			{Name: "greet", ReqInitFunc: func() any { return &GreetRequest{} }}, // same name different case
		},
	}

	openAPI := r.Resolve("GreetService", info)
	// Both "Greet" and "greet" should map to same path (case-insensitive dedup)
	pathItem := openAPI.Paths["/GreetService/Greet"]
	if pathItem == nil {
		t.Fatal("path should exist")
	}
	// Only one operation should be generated
	opCount := len(pathItem.GetOperations())
	if opCount != 1 {
		t.Errorf("expected 1 operation (deduped), got %d", opCount)
	}
}

func TestDefinitionResolver_Resolve_NoMethods(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	info := &serviceInfo{Methods: []serviceMethodInfo{}}
	openAPI := r.Resolve("EmptyService", info)

	if openAPI == nil {
		t.Fatal("Resolve should not return nil even with no methods")
	}
	if len(openAPI.Paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(openAPI.Paths))
	}
}

// --- resolveRequestSchema tests ---

func TestDefinitionResolver_resolveRequestSchema_MetaRequestType(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name: "Greet",
		Meta: map[string]any{
			"request.type": reflect.TypeOf(GreetRequest{}),
		},
	}

	schema := r.resolveRequestSchema(method, schemaResolver)
	if schema == nil {
		t.Fatal("resolveRequestSchema should not return nil with request.type meta")
	}
	if schema.Ref == "" {
		t.Error("schema should have a $ref for struct type")
	}
}

func TestDefinitionResolver_resolveRequestSchema_PtrMetaRequestType(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name: "Greet",
		Meta: map[string]any{
			"request.type": reflect.TypeOf((*GreetRequest)(nil)), // *GreetRequest
		},
	}

	schema := r.resolveRequestSchema(method, schemaResolver)
	if schema == nil {
		t.Fatal("resolveRequestSchema should not return nil for ptr type in meta")
	}
	// Ptr should be dereferenced
	if schema.Ref == "" {
		t.Error("schema should have a $ref after ptr dereference")
	}
}

func TestDefinitionResolver_resolveRequestSchema_ReqInitFunc(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name:        "Greet",
		ReqInitFunc: func() any { return &GreetRequest{} },
	}

	schema := r.resolveRequestSchema(method, schemaResolver)
	if schema == nil {
		t.Fatal("resolveRequestSchema should not return nil with ReqInitFunc")
	}
}

func TestDefinitionResolver_resolveRequestSchema_NilReqInitFunc(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name:        "Greet",
		ReqInitFunc: nil,
	}

	schema := r.resolveRequestSchema(method, schemaResolver)
	if schema != nil {
		t.Error("resolveRequestSchema should return nil when ReqInitFunc is nil and no meta")
	}
}

func TestDefinitionResolver_resolveRequestSchema_NilReqReturn(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name:        "Greet",
		ReqInitFunc: func() any { return nil },
	}

	schema := r.resolveRequestSchema(method, schemaResolver)
	if schema != nil {
		t.Error("resolveRequestSchema should return nil when ReqInitFunc returns nil")
	}
}

// --- resolveResponseSchema tests ---

func TestDefinitionResolver_resolveResponseSchema_MetaResponseType(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name: "Greet",
		Meta: map[string]any{
			"response.type": reflect.TypeOf(GreetResponse{}),
		},
	}

	schema := r.resolveResponseSchema(method, schemaResolver)
	if schema == nil {
		t.Fatal("resolveResponseSchema should not return nil with response.type meta")
	}
	if schema.Ref == "" {
		t.Error("schema should have a $ref for struct type")
	}
}

func TestDefinitionResolver_resolveResponseSchema_PtrMetaResponseType(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name: "Greet",
		Meta: map[string]any{
			"response.type": reflect.TypeOf((*GreetResponse)(nil)),
		},
	}

	schema := r.resolveResponseSchema(method, schemaResolver)
	if schema == nil {
		t.Fatal("resolveResponseSchema should not return nil for ptr type in meta")
	}
}

func TestDefinitionResolver_resolveResponseSchema_NoMeta(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name: "Greet",
		Meta: nil,
	}

	schema := r.resolveResponseSchema(method, schemaResolver)
	if schema != nil {
		t.Error("resolveResponseSchema should return nil when no meta")
	}
}

// --- buildPath tests ---

func TestDefinitionResolver_buildPath(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	tests := []struct {
		iface  string
		method string
		want   string
	}{
		{"GreetService", "Greet", "/GreetService/Greet"},
		{"UserService", "GetUser", "/UserService/GetUser"},
		{"", "Method", "//Method"}, // buildPath concatenates "/" + interfaceName + "/" + methodName
	}

	for _, tt := range tests {
		got := r.buildPath(tt.iface, tt.method)
		if got != tt.want {
			t.Errorf("buildPath(%q, %q) = %q, want %q", tt.iface, tt.method, got, tt.want)
		}
	}
}

// --- getStatusDescription tests ---

func TestDefinitionResolver_getStatusDescription(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	tests := []struct {
		code string
		want string
	}{
		{"200", "OK"},
		{"400", "Bad Request"},
		{"401", "Unauthorized"},
		{"403", "Forbidden"},
		{"404", "Not Found"},
		{"500", "Internal Server Error"},
		{"999", "Unknown"},
	}

	for _, tt := range tests {
		got := r.getStatusDescription(tt.code)
		if got != tt.want {
			t.Errorf("getStatusDescription(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// --- resolveVersion tests ---

func TestDefinitionResolver_resolveVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"custom version", "2.0.0", "2.0.0"},
		{"empty version uses default", "", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := global.DefaultOpenAPIConfig()
			cfg.InfoVersion = tt.version
			r := NewDefinitionResolver(cfg)
			got := r.resolveVersion()
			if got != tt.want {
				t.Errorf("resolveVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- determineHttpMethods tests ---

func TestDefinitionResolver_determineHttpMethods(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)

	methods := r.determineHttpMethods()
	if len(methods) != 1 || methods[0] != "POST" {
		t.Errorf("determineHttpMethods() = %v, want [POST]", methods)
	}
}

// --- resolveOperation tests ---

func TestDefinitionResolver_resolveOperation(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name:        "Greet",
		ReqInitFunc: func() any { return &GreetRequest{} },
		Meta: map[string]any{
			"response.type": reflect.TypeOf(GreetResponse{}),
		},
	}

	op := r.resolveOperation(method, "POST", "GreetService", schemaResolver)

	if op.OperationId != "GreetService.Greet" {
		t.Errorf("OperationId = %q, want %q", op.OperationId, "GreetService.Greet")
	}
	if op.RequestBody == nil {
		t.Error("RequestBody should not be nil")
	}
	if len(op.Responses) == 0 {
		t.Error("Responses should not be empty")
	}

	// Check default status codes
	for _, code := range []string{"200", "400", "500"} {
		if _, ok := op.Responses[code]; !ok {
			t.Errorf("response for status %s not found", code)
		}
	}

	// Check tags
	if len(op.Tags) != 1 || op.Tags[0] != "GreetService" {
		t.Errorf("Tags = %v, want [GreetService]", op.Tags)
	}
}

func TestDefinitionResolver_resolveOperation_CustomMediaTypes(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	cfg.DefaultConsumesMediaTypes = []string{"application/json", "application/x-protobuf"}
	cfg.DefaultProducesMediaTypes = []string{"application/json", "application/x-protobuf"}
	cfg.DefaultHttpStatusCodes = []string{"200", "404"}

	r := NewDefinitionResolver(cfg)
	schemaResolver := NewSchemaResolver(cfg)

	method := serviceMethodInfo{
		Name:        "Greet",
		ReqInitFunc: func() any { return &GreetRequest{} },
	}

	op := r.resolveOperation(method, "POST", "GreetService", schemaResolver)

	// Check request body content media types
	if len(op.RequestBody.Content) != 2 {
		t.Errorf("RequestBody Content count = %d, want 2", len(op.RequestBody.Content))
	}
	for _, mt := range cfg.DefaultConsumesMediaTypes {
		if _, ok := op.RequestBody.Content[mt]; !ok {
			t.Errorf("RequestBody Content missing media type %q", mt)
		}
	}

	// Check response for 200
	resp200 := op.Responses["200"]
	if resp200 == nil {
		t.Fatal("200 response should exist")
	}
	if len(resp200.Content) != 2 {
		t.Errorf("200 response Content count = %d, want 2", len(resp200.Content))
	}

	// Check 404 exists but 400/500 should not
	if _, ok := op.Responses["400"]; ok {
		t.Error("400 should not be in responses with custom codes")
	}
	if _, ok := op.Responses["404"]; !ok {
		t.Error("404 should be in responses with custom codes")
	}
}

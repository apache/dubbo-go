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
	"encoding/json"
	"strings"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

// --- Encode tests ---
func TestEncoder_Encode_NilOpenAPI(t *testing.T) {
	e := NewEncoder()
	result, err := e.Encode(nil, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("result should not be empty")
	}

	// Should produce valid JSON with default OpenAPI structure
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result should be valid JSON: %v", err)
	}
	if v, ok := parsed["openapi"]; !ok || v != model.OpenAPIVersion30 {
		t.Errorf("openapi version = %v, want %s", v, model.OpenAPIVersion30)
	}
}

func TestEncoder_Encode_JSONPretty(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()
	openAPI.Info.Title = "Test API"
	openAPI.Info.Version = "1.0.0"

	result, err := e.Encode(openAPI, "json", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "  ") {
		t.Error("pretty JSON should contain indentation")
	}
}

func TestEncoder_Encode_JSONCompact(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()
	openAPI.Info.Title = "Test API"

	result, err := e.Encode(openAPI, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "\n  ") {
		t.Error("compact JSON should not contain indentation")
	}
}

func TestEncoder_Encode_YAML(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()
	openAPI.Info.Title = "Test API"
	openAPI.Info.Version = "1.0.0"

	result, err := e.Encode(openAPI, "yaml", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "openapi:") {
		t.Error("YAML output should contain 'openapi:' key")
	}
	if !strings.Contains(result, "title: Test API") {
		t.Error("YAML output should contain title")
	}
}

func TestEncoder_Encode_YML(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()

	result, err := e.Encode(openAPI, "yml", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "openapi:") {
		t.Error("YML output should contain 'openapi:' key")
	}
}

func TestEncoder_Encode_UnknownFormat_FallbackJSON(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()
	openAPI.Info.Title = "Test API"

	result, err := e.Encode(openAPI, "xml", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fallback to JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("unknown format should fallback to JSON, but got invalid JSON: %v", err)
	}
}

func TestEncoder_Encode_EmptyFormat_DefaultJSON(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()

	result, err := e.Encode(openAPI, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("empty format should default to JSON: %v", err)
	}
}

// --- toMap tests ---

func TestEncoder_toMap_FullOpenAPI(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()
	openAPI.Info.Title = "Test API"
	openAPI.Info.Version = "1.0.0"
	openAPI.Info.Description = "A test API"
	openAPI.AddPath("/test", model.NewPathItem().SetOperation("POST",
		model.NewOperation().SetOperationId("testOp")))
	openAPI.Components = model.NewComponents()
	openAPI.Components.AddSchema("TestSchema", model.NewSchema().SetType(model.SchemaTypeObject))

	m := e.toMap(openAPI)

	if m["openapi"] != model.OpenAPIVersion30 {
		t.Errorf("openapi = %v, want %s", m["openapi"], model.OpenAPIVersion30)
	}
	if _, ok := m["info"]; !ok {
		t.Error("info should be present")
	}
	if _, ok := m["paths"]; !ok {
		t.Error("paths should be present")
	}
	if _, ok := m["components"]; !ok {
		t.Error("components should be present")
	}
}

func TestEncoder_toMap_NoPaths(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()

	m := e.toMap(openAPI)
	if _, ok := m["paths"]; ok {
		t.Error("paths should not be present when empty")
	}
}

func TestEncoder_toMap_NoComponents(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()
	openAPI.Components = nil

	m := e.toMap(openAPI)
	if _, ok := m["components"]; ok {
		t.Error("components should not be present when nil")
	}
}

// --- infoToMap tests ---

func TestEncoder_infoToMap(t *testing.T) {
	e := NewEncoder()
	info := &model.Info{
		Title:       "Test",
		Description: "Desc",
		Version:     "1.0",
	}

	m := e.infoToMap(info)
	if m["title"] != "Test" {
		t.Errorf("title = %v, want Test", m["title"])
	}
	if m["description"] != "Desc" {
		t.Errorf("description = %v, want Desc", m["description"])
	}
	if m["version"] != "1.0" {
		t.Errorf("version = %v, want 1.0", m["version"])
	}
}

func TestEncoder_infoToMap_EmptyFields(t *testing.T) {
	e := NewEncoder()
	info := &model.Info{}

	m := e.infoToMap(info)
	if _, ok := m["title"]; ok {
		t.Error("empty title should not be in map")
	}
	if _, ok := m["description"]; ok {
		t.Error("empty description should not be in map")
	}
	if _, ok := m["version"]; ok {
		t.Error("empty version should not be in map")
	}
}

// --- schemaToMap tests ---

func TestEncoder_schemaToMap_Ref(t *testing.T) {
	e := NewEncoder()
	schema := &model.Schema{Ref: "#/components/schemas/MyType"}

	m := e.schemaToMap(schema)
	if m["$ref"] != "#/components/schemas/MyType" {
		t.Errorf("$ref = %v, want #/components/schemas/MyType", m["$ref"])
	}
	// Ref schemas should only have $ref
	if len(m) != 1 {
		t.Errorf("ref schema should only have $ref field, got %d fields", len(m))
	}
}

func TestEncoder_schemaToMap_FullSchema(t *testing.T) {
	e := NewEncoder()
	schema := model.NewSchema().
		SetType(model.SchemaTypeObject).
		SetFormat("email").
		SetTitle("EmailSchema").
		SetDescription("An email").
		SetRequired(true).
		SetExample("test@example.com")
	schema.Enum = []any{"a", "b"}

	m := e.schemaToMap(schema)

	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}
	if m["format"] != "email" {
		t.Errorf("format = %v, want email", m["format"])
	}
	if m["title"] != "EmailSchema" {
		t.Errorf("title = %v, want EmailSchema", m["title"])
	}
	if m["description"] != "An email" {
		t.Errorf("description = %v", m["description"])
	}
	if m["required"] != true {
		t.Errorf("required = %v, want true", m["required"])
	}
	if m["example"] != "test@example.com" {
		t.Errorf("example = %v", m["example"])
	}
}

func TestEncoder_schemaToMap_ArrayWithItems(t *testing.T) {
	e := NewEncoder()
	schema := model.NewSchema().
		SetType(model.SchemaTypeArray).
		SetItems(model.NewSchema().SetType(model.SchemaTypeString))

	m := e.schemaToMap(schema)
	if m["type"] != "array" {
		t.Errorf("type = %v, want array", m["type"])
	}
	items, ok := m["items"].(map[string]any)
	if !ok {
		t.Fatal("items should be map[string]any")
	}
	if items["type"] != "string" {
		t.Errorf("items.type = %v, want string", items["type"])
	}
}

func TestEncoder_schemaToMap_ObjectWithProperties(t *testing.T) {
	e := NewEncoder()
	schema := model.NewSchema().SetType(model.SchemaTypeObject)
	schema.AddProperty("name", model.NewSchema().SetType(model.SchemaTypeString))
	schema.AddProperty("age", model.NewSchema().SetType(model.SchemaTypeInteger))

	m := e.schemaToMap(schema)
	props, ok := m["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be map[string]any")
	}
	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatal("name property should be map[string]any")
	}
	if nameProp["type"] != "string" {
		t.Errorf("name.type = %v, want string", nameProp["type"])
	}
}

func TestEncoder_schemaToMap_AdditionalProperties_Schema(t *testing.T) {
	e := NewEncoder()
	// This is the bug fix test: additionalProperties with *Schema should be serialized properly
	schema := model.NewSchema().
		SetType(model.SchemaTypeObject).
		SetAdditionalProperties(model.NewSchema().SetType(model.SchemaTypeInteger))

	m := e.schemaToMap(schema)
	ap, ok := m["additionalProperties"]
	if !ok {
		t.Fatal("additionalProperties should be present")
	}

	// After fix, additionalProperties should be a map, not a *model.Schema
	apMap, ok := ap.(map[string]any)
	if !ok {
		t.Fatalf("additionalProperties should be map[string]any, got %T", ap)
	}
	if apMap["type"] != "integer" {
		t.Errorf("additionalProperties.type = %v, want integer", apMap["type"])
	}
}

func TestEncoder_schemaToMap_AdditionalProperties_Bool(t *testing.T) {
	e := NewEncoder()
	schema := model.NewSchema().
		SetType(model.SchemaTypeObject).
		SetAdditionalProperties(true)

	m := e.schemaToMap(schema)
	ap, ok := m["additionalProperties"]
	if !ok {
		t.Fatal("additionalProperties should be present")
	}
	if ap != true {
		t.Errorf("additionalProperties = %v, want true", ap)
	}
}

// --- operationToMap tests ---

func TestEncoder_operationToMap(t *testing.T) {
	e := NewEncoder()
	op := model.NewOperation().
		SetOperationId("testOp").
		AddTag("tag1")
	op.SetRequestBody(model.NewRequestBody().SetRequired(true))
	op.GetOrAddResponse("200").Description = "OK"

	m := e.operationToMap(op)

	if m["operationId"] != "testOp" {
		t.Errorf("operationId = %v, want testOp", m["operationId"])
	}
	if _, ok := m["tags"]; !ok {
		t.Error("tags should be present")
	}
	if _, ok := m["requestBody"]; !ok {
		t.Error("requestBody should be present")
	}
	if _, ok := m["responses"]; !ok {
		t.Error("responses should be present")
	}
}

// --- componentsToMap tests ---

func TestEncoder_componentsToMap(t *testing.T) {
	e := NewEncoder()
	c := model.NewComponents()
	c.AddSchema("User", model.NewSchema().SetType(model.SchemaTypeObject).SetTitle("User"))
	c.AddSchema("Req", model.NewSchema().SetType(model.SchemaTypeObject).SetTitle("Req"))

	m := e.componentsToMap(c)
	schemas, ok := m["schemas"].(map[string]any)
	if !ok {
		t.Fatal("schemas should be map[string]any")
	}
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}
}

func TestEncoder_componentsToMap_Empty(t *testing.T) {
	e := NewEncoder()
	c := model.NewComponents()

	m := e.componentsToMap(c)
	if _, ok := m["schemas"]; ok {
		t.Error("empty schemas should not be in map")
	}
}

// --- round-trip test ---

func TestEncoder_RoundTrip_JSON(t *testing.T) {
	e := NewEncoder()
	openAPI := model.NewOpenAPI()
	openAPI.Info.Title = "Round Trip API"
	openAPI.Info.Version = "1.0.0"
	openAPI.AddPath("/test", model.NewPathItem().SetOperation("POST",
		model.NewOperation().SetOperationId("testOp")))

	jsonStr, err := e.Encode(openAPI, "json", true)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed["openapi"] != model.OpenAPIVersion30 {
		t.Errorf("openapi = %v, want %s", parsed["openapi"], model.OpenAPIVersion30)
	}

	info, ok := parsed["info"].(map[string]any)
	if !ok {
		t.Fatal("info should be map[string]any")
	}
	if info["title"] != "Round Trip API" {
		t.Errorf("title = %v, want Round Trip API", info["title"])
	}
}

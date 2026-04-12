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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

// --- test types for schema resolution ---
type SimpleStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email,omitempty"`
}

type StructWithUnexported struct {
	Exported   string `json:"exported"`
	unexported string //nolint:unused
}

type StructWithJsonIgnore struct {
	Visible string `json:"visible"`
	Hidden  string `json:"-"`
}

type StructWithEmbedded struct {
	SimpleStruct
	Extra bool `json:"extra"`
}

type StructWithSlice struct {
	Items []string `json:"items"`
}

type StructWithMap struct {
	Metadata map[string]int `json:"metadata"`
}

type StructWithTime struct {
	CreatedAt time.Time `json:"created_at"`
}

type StructWithNested struct {
	Inner *SimpleStruct `json:"inner"`
}

type StructWithPtr struct {
	Value *string `json:"value"`
}

type RecursiveStruct struct {
	Name  string           `json:"name"`
	Child *RecursiveStruct `json:"child,omitempty"`
	Items []string         `json:"items"`
}

type EnumStruct struct {
	Status string `json:"status"`
}

// --- resolveType tests ---

func TestSchemaResolver_Resolve_BasicTypes(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	tests := []struct {
		name     string
		input    reflect.Type
		wantType model.SchemaType
		wantFmt  string
	}{
		{"string", reflect.TypeOf(""), model.SchemaTypeString, ""},
		{"bool", reflect.TypeOf(true), model.SchemaTypeBoolean, ""},
		{"int", reflect.TypeOf(int(0)), model.SchemaTypeInteger, ""},
		{"int8", reflect.TypeOf(int8(0)), model.SchemaTypeInteger, ""},
		{"int16", reflect.TypeOf(int16(0)), model.SchemaTypeInteger, ""},
		{"int32", reflect.TypeOf(int32(0)), model.SchemaTypeInteger, "int32"},
		{"int64", reflect.TypeOf(int64(0)), model.SchemaTypeInteger, "int64"},
		{"uint", reflect.TypeOf(uint(0)), model.SchemaTypeInteger, ""},
		{"uint8", reflect.TypeOf(uint8(0)), model.SchemaTypeInteger, ""},
		{"uint16", reflect.TypeOf(uint16(0)), model.SchemaTypeInteger, ""},
		{"uint32", reflect.TypeOf(uint32(0)), model.SchemaTypeInteger, "int32"},
		{"uint64", reflect.TypeOf(uint64(0)), model.SchemaTypeInteger, "int64"},
		{"float32", reflect.TypeOf(float32(0)), model.SchemaTypeNumber, "float"},
		{"float64", reflect.TypeOf(float64(0)), model.SchemaTypeNumber, "double"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := r.Resolve(tt.input)
			if schema.Type != tt.wantType {
				t.Errorf("Resolve(%s).Type = %v, want %v", tt.name, schema.Type, tt.wantType)
			}
			if schema.Format != tt.wantFmt {
				t.Errorf("Resolve(%s).Format = %v, want %v", tt.name, schema.Format, tt.wantFmt)
			}
		})
	}
}

func TestSchemaResolver_Resolve_PtrDereference(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := r.Resolve(reflect.TypeOf((*string)(nil)).Elem())
	// *string should be dereferenced to string
	if schema.Type != model.SchemaTypeString {
		t.Errorf("Resolve(*string).Type = %v, want %v", schema.Type, model.SchemaTypeString)
	}
}

func TestSchemaResolver_Resolve_Slice(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := r.Resolve(reflect.TypeOf([]string{}))
	if schema.Type != model.SchemaTypeArray {
		t.Errorf("Resolve([]string).Type = %v, want %v", schema.Type, model.SchemaTypeArray)
	}
	if schema.Items == nil {
		t.Fatal("Resolve([]string).Items should not be nil")
	}
	if schema.Items.Type != model.SchemaTypeString {
		t.Errorf("Resolve([]string).Items.Type = %v, want %v", schema.Items.Type, model.SchemaTypeString)
	}
}

func TestSchemaResolver_Resolve_Map(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := r.Resolve(reflect.TypeOf(map[string]int{}))
	if schema.Type != model.SchemaTypeObject {
		t.Errorf("Resolve(map[string]int).Type = %v, want %v", schema.Type, model.SchemaTypeObject)
	}
	if schema.AdditionalProperties == nil {
		t.Fatal("Resolve(map[string]int).AdditionalProperties should not be nil")
	}
	apSchema, ok := schema.AdditionalProperties.(*model.Schema)
	if !ok {
		t.Fatal("AdditionalProperties should be *model.Schema")
	}
	if apSchema.Type != model.SchemaTypeInteger {
		t.Errorf("AdditionalProperties.Type = %v, want %v", apSchema.Type, model.SchemaTypeInteger)
	}
}

func TestSchemaResolver_Resolve_Interface(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := r.Resolve(reflect.TypeOf((*any)(nil)).Elem())
	if schema.Type != model.SchemaTypeObject {
		t.Errorf("Resolve(interface{}).Type = %v, want %v", schema.Type, model.SchemaTypeObject)
	}
}

func TestSchemaResolver_Resolve_Nil(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := r.Resolve(nil)
	if schema.Type != model.SchemaTypeObject {
		t.Errorf("Resolve(nil).Type = %v, want %v", schema.Type, model.SchemaTypeObject)
	}
}

// --- resolveStruct tests ---

func TestSchemaResolver_Resolve_Struct(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := r.Resolve(reflect.TypeOf(SimpleStruct{}))

	// Struct should return a $ref
	if schema.Ref == "" {
		t.Fatal("Resolve(SimpleStruct) should return a $ref schema")
	}
	if schema.Ref != "#/components/schemas/"+r.toSchemaName(reflect.TypeOf(SimpleStruct{})) {
		t.Errorf("Ref = %v, unexpected", schema.Ref)
	}
}

func TestSchemaResolver_Resolve_StructProperties(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	// Resolve to populate the internal schema map
	r.Resolve(reflect.TypeOf(SimpleStruct{}))

	schemas := r.GetSchemas()
	schemaName := r.toSchemaName(reflect.TypeOf(SimpleStruct{}))
	schema, ok := schemas[schemaName]
	if !ok {
		t.Fatalf("schema %s not found", schemaName)
	}

	expectedProps := map[string]model.SchemaType{
		"name":  model.SchemaTypeString,
		"age":   model.SchemaTypeInteger,
		"email": model.SchemaTypeString,
	}

	for propName, wantType := range expectedProps {
		prop, exists := schema.Properties[propName]
		if !exists {
			t.Errorf("property %s not found", propName)
			continue
		}
		if prop.Type != wantType {
			t.Errorf("property %s.Type = %v, want %v", propName, prop.Type, wantType)
		}
	}
}

func TestSchemaResolver_Resolve_StructUnexportedFieldsSkipped(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	r.Resolve(reflect.TypeOf(StructWithUnexported{}))
	schemas := r.GetSchemas()
	schemaName := r.toSchemaName(reflect.TypeOf(StructWithUnexported{}))
	schema := schemas[schemaName]

	if _, exists := schema.Properties["unexported"]; exists {
		t.Error("unexported field should be skipped")
	}
	if _, exists := schema.Properties["exported"]; !exists {
		t.Error("exported field should be present")
	}
}

func TestSchemaResolver_Resolve_JsonDashIgnored(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	r.Resolve(reflect.TypeOf(StructWithJsonIgnore{}))
	schemas := r.GetSchemas()
	schemaName := r.toSchemaName(reflect.TypeOf(StructWithJsonIgnore{}))
	schema := schemas[schemaName]

	if _, exists := schema.Properties["Hidden"]; exists {
		t.Error("field with json:\"-\" should be skipped")
	}
	if _, exists := schema.Properties["visible"]; !exists {
		t.Error("field with json:\"visible\" should be present")
	}
}

func TestSchemaResolver_Resolve_TimeTime(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := r.Resolve(reflect.TypeOf(time.Time{}))
	// time.Time should produce string + date-time, not a $ref to a struct
	if schema.Ref != "" {
		t.Errorf("time.Time should not produce a $ref, got %s", schema.Ref)
	}
	if schema.Type != model.SchemaTypeString {
		t.Errorf("time.Time.Type = %v, want %v", schema.Type, model.SchemaTypeString)
	}
	if schema.Format != "date-time" {
		t.Errorf("time.Time.Format = %v, want date-time", schema.Format)
	}
}

func TestSchemaResolver_Resolve_StructWithTime(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	r.Resolve(reflect.TypeOf(StructWithTime{}))
	schemas := r.GetSchemas()
	schemaName := r.toSchemaName(reflect.TypeOf(StructWithTime{}))
	schema := schemas[schemaName]

	createdAt, exists := schema.Properties["created_at"]
	if !exists {
		t.Fatal("created_at property not found")
	}
	// The property should be a string with date-time format, not a $ref
	if createdAt.Type != model.SchemaTypeString {
		t.Errorf("created_at.Type = %v, want %v", createdAt.Type, model.SchemaTypeString)
	}
	if createdAt.Format != "date-time" {
		t.Errorf("created_at.Format = %v, want date-time", createdAt.Format)
	}
}

func TestSchemaResolver_Resolve_CircularReference(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	// Resolving a recursive struct should not loop infinitely
	schema := r.Resolve(reflect.TypeOf(RecursiveStruct{}))
	if schema.Ref == "" {
		t.Error("RecursiveStruct should produce a $ref")
	}
}

// --- toSchemaName / sanitizeSchemaName tests ---

func TestSchemaResolver_toSchemaName(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	typ := reflect.TypeOf(SimpleStruct{})
	name := r.toSchemaName(typ)

	if name == "" {
		t.Error("toSchemaName should not return empty string")
	}
	// Should contain sanitized pkg path + type name
	if name == "SimpleStruct" {
		t.Error("toSchemaName should include package path prefix for external types")
	}
}

func TestSanitizeSchemaName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/example/pkg", "github_com_example_pkg"},
		{"simple", "simple"},
		{"has-dash", "has_dash"},
		{"has.dot", "has_dot"},
	}

	for _, tt := range tests {
		got := sanitizeSchemaName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeSchemaName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- GetSchemaName tests ---

func TestSchemaResolver_GetSchemaName(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	tests := []struct {
		name   string
		schema *model.Schema
		want   string
	}{
		{"nil schema", nil, ""},
		{"empty ref", &model.Schema{}, ""},
		{"valid ref", &model.Schema{Ref: "#/components/schemas/MyType"}, "MyType"},
		{"invalid ref format", &model.Schema{Ref: "#/wrong/path"}, ""},
		{"partial ref", &model.Schema{Ref: "#/components/schemas"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.GetSchemaName(tt.schema)
			if got != tt.want {
				t.Errorf("GetSchemaName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- GetSchemaDefinition tests ---

func TestSchemaResolver_GetSchemaDefinition(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	// nil schema
	if def := r.GetSchemaDefinition(nil); def != nil {
		t.Error("GetSchemaDefinition(nil) should return nil")
	}

	// schema without ref returns itself
	noRefSchema := model.NewSchema().SetType(model.SchemaTypeString)
	if def := r.GetSchemaDefinition(noRefSchema); def != noRefSchema {
		t.Error("GetSchemaDefinition(schema without ref) should return the schema itself")
	}

	// schema with ref pointing to non-existent schema
	refSchema := &model.Schema{Ref: "#/components/schemas/NonExistent"}
	if def := r.GetSchemaDefinition(refSchema); def != nil {
		t.Error("GetSchemaDefinition(non-existent ref) should return nil")
	}
}

// --- GenerateExample tests ---

func TestSchemaResolver_GenerateExample_BasicTypes(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	tests := []struct {
		name   string
		schema *model.Schema
		want   any
	}{
		{"nil", nil, nil},
		{"string", model.NewSchema().SetType(model.SchemaTypeString), "string"},
		{"integer", model.NewSchema().SetType(model.SchemaTypeInteger), 0},
		{"number", model.NewSchema().SetType(model.SchemaTypeNumber), 0.0},
		{"boolean", model.NewSchema().SetType(model.SchemaTypeBoolean), false},
		{"date-time", model.NewSchema().SetType(model.SchemaTypeString).SetFormat("date-time"), "2024-01-01T00:00:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.GenerateExample(tt.schema)
			if got != tt.want {
				t.Errorf("GenerateExample() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchemaResolver_GenerateExample_Array(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := model.NewSchema().
		SetType(model.SchemaTypeArray).
		SetItems(model.NewSchema().SetType(model.SchemaTypeString))

	example := r.GenerateExample(schema)
	arr, ok := example.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", example)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
	if arr[0] != "string" {
		t.Errorf("item = %v, want string", arr[0])
	}
}

func TestSchemaResolver_GenerateExample_ObjectWithProperties(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := model.NewSchema().
		SetType(model.SchemaTypeObject).
		SetTitle("TestObj").
		AddProperty("name", model.NewSchema().SetType(model.SchemaTypeString)).
		AddProperty("age", model.NewSchema().SetType(model.SchemaTypeInteger))

	example := r.GenerateExample(schema)
	m, ok := example.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", example)
	}
	if m["name"] != "string" {
		t.Errorf("name = %v, want string", m["name"])
	}
	if m["age"] != 0 {
		t.Errorf("age = %v, want 0", m["age"])
	}
}

func TestSchemaResolver_GenerateExample_EmptyObject(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	schema := model.NewSchema().SetType(model.SchemaTypeObject)
	example := r.GenerateExample(schema)
	m, ok := example.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", example)
	}
	if len(m) != 0 {
		t.Errorf("empty object example should have 0 properties, got %d", len(m))
	}
}

// --- GetSchemas tests ---

func TestSchemaResolver_GetSchemas(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	// Before resolving anything, schemas should be empty
	schemas := r.GetSchemas()
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(schemas))
	}

	// After resolving a struct
	r.Resolve(reflect.TypeOf(SimpleStruct{}))
	schemas = r.GetSchemas()
	if len(schemas) == 0 {
		t.Error("expected schemas after resolving struct")
	}
}

// --- isExported / getFieldName tests ---

func TestSchemaResolver_isExported(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	typ := reflect.TypeOf(StructWithUnexported{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "Exported" && !r.isExported(field) {
			t.Error("Exported field should be considered exported")
		}
		if field.Name == "unexported" && r.isExported(field) {
			t.Error("unexported field should not be considered exported")
		}
	}
}

func TestSchemaResolver_getFieldName(t *testing.T) {
	r := NewSchemaResolver(global.DefaultOpenAPIConfig())

	typ := reflect.TypeOf(SimpleStruct{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		name := r.getFieldName(field, jsonTag)
		if name == "" {
			t.Errorf("getFieldName(%s) should not return empty", field.Name)
		}
	}

	// Test empty json tag falls back to field name
	typ2 := reflect.TypeOf(StructWithUnexported{})
	exportedField, _ := typ2.FieldByName("Exported")
	name := r.getFieldName(exportedField, "")
	if name != "Exported" {
		t.Errorf("getFieldName with empty json tag = %q, want Exported", name)
	}
}

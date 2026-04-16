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
	"strings"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

type SchemaResolver struct {
	config       *global.OpenAPIConfig
	schemaMap    *sync.Map
	nameToSchema *sync.Map
}

func NewSchemaResolver(config *global.OpenAPIConfig) *SchemaResolver {
	return &SchemaResolver{
		config:       config,
		schemaMap:    &sync.Map{},
		nameToSchema: &sync.Map{},
	}
}

func (r *SchemaResolver) toSchemaName(t reflect.Type) string {
	if t.PkgPath() != "" {
		return sanitizeSchemaName(t.PkgPath()) + "_" + t.Name()
	}
	name := t.Name()
	if name == "" {
		name = t.String()
	}
	return name
}

func sanitizeSchemaName(s string) string {
	var b strings.Builder
	for _, c := range s {
		if isAlphaNum(c) {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func isAlphaNum(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// parseSchemaRef extracts the schema name from a $ref string in the format
// "#/components/schemas/<name>". It returns an empty string if the ref does
// not match the expected format.
func parseSchemaRef(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) == 4 && parts[0] == "#" && parts[1] == "components" && parts[2] == "schemas" {
		return parts[3]
	}
	return ""
}

func (r *SchemaResolver) Resolve(t reflect.Type) *model.Schema {
	return r.resolveType(t)
}

func (r *SchemaResolver) resolveType(t reflect.Type) *model.Schema {
	if t == nil {
		return model.NewSchema().SetType(model.SchemaTypeObject)
	}

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return model.NewSchema().SetType(model.SchemaTypeString)
	case reflect.Bool:
		return model.NewSchema().SetType(model.SchemaTypeBoolean)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s := model.NewSchema().SetType(model.SchemaTypeInteger)
		if t.Kind() == reflect.Int64 {
			s.SetFormat("int64")
		} else if t.Kind() == reflect.Int32 {
			s.SetFormat("int32")
		}
		return s
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s := model.NewSchema().SetType(model.SchemaTypeInteger)
		if t.Kind() == reflect.Uint64 {
			s.SetFormat("int64")
		} else if t.Kind() == reflect.Uint32 {
			s.SetFormat("int32")
		}
		return s
	case reflect.Float32, reflect.Float64:
		s := model.NewSchema().SetType(model.SchemaTypeNumber)
		if t.Kind() == reflect.Float64 {
			s.SetFormat("double")
		} else {
			s.SetFormat("float")
		}
		return s
	case reflect.Array, reflect.Slice:
		return model.NewSchema().
			SetType(model.SchemaTypeArray).
			SetItems(r.resolveType(t.Elem()))
	case reflect.Map:
		return model.NewSchema().
			SetType(model.SchemaTypeObject).
			SetAdditionalProperties(r.resolveType(t.Elem()))
	case reflect.Struct:
		return r.resolveStruct(t)
	case reflect.Interface:
		return model.NewSchema().SetType(model.SchemaTypeObject)
	default:
		return model.NewSchema().SetType(model.SchemaTypeObject)
	}
}

func (r *SchemaResolver) resolveStruct(t reflect.Type) *model.Schema {
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return model.NewSchema().
			SetType(model.SchemaTypeString).
			SetFormat("date-time")
	}

	if cached, ok := r.schemaMap.Load(t); ok {
		if s, ok := cached.(*model.Schema); ok {
			return &model.Schema{
				Ref: "#/components/schemas/" + s.Title,
			}
		}
	}

	schemaName := r.toSchemaName(t)

	schema := model.NewSchema().
		SetType(model.SchemaTypeObject).
		SetTitle(schemaName).
		SetGoType(t.String())

	r.schemaMap.Store(t, schema)
	r.nameToSchema.Store(schemaName, schema)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !r.isExported(field) {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := r.getFieldName(field, jsonTag)
		if fieldName == "" {
			continue
		}

		fieldSchema := r.resolveType(field.Type)
		schema.AddProperty(fieldName, fieldSchema)
	}

	example := r.GenerateExample(schema)
	if example != nil {
		schema.SetExample(example)
	}

	return &model.Schema{
		Ref: "#/components/schemas/" + schema.Title,
	}
}

func (r *SchemaResolver) isExported(f reflect.StructField) bool {
	return f.PkgPath == ""
}

func (r *SchemaResolver) getFieldName(f reflect.StructField, jsonTag string) string {
	if jsonTag != "" {
		name := strings.Split(jsonTag, ",")[0]
		if name != "" {
			return name
		}
	}
	return f.Name
}

func (r *SchemaResolver) GetSchemas() map[string]*model.Schema {
	schemas := make(map[string]*model.Schema)
	r.schemaMap.Range(func(key, value any) bool {
		if t, ok := key.(reflect.Type); ok {
			if s, ok := value.(*model.Schema); ok {
				name := r.toSchemaName(t)
				schemas[name] = s
			}
		}
		return true
	})
	return schemas
}

func (r *SchemaResolver) GetSchemaName(schema *model.Schema) string {
	if schema == nil || schema.Ref == "" {
		return ""
	}
	return parseSchemaRef(schema.Ref)
}

func (r *SchemaResolver) GetSchemaDefinition(schema *model.Schema) *model.Schema {
	if schema == nil {
		return nil
	}
	if schema.Ref == "" {
		return schema
	}
	schemaName := parseSchemaRef(schema.Ref)
	if schemaName != "" {
		if s, ok := r.nameToSchema.Load(schemaName); ok {
			if resolvedSchema, ok := s.(*model.Schema); ok {
				return resolvedSchema
			}
		}
	}
	return nil
}

const maxExampleDepth = 15

func (r *SchemaResolver) GenerateExample(schema *model.Schema) any {
	return r.generateExample(schema, make(map[string]struct{}), 0)
}

func (r *SchemaResolver) generateExample(schema *model.Schema, visiting map[string]struct{}, depth int) any {
	if schema == nil {
		return nil
	}

	if depth > maxExampleDepth {
		return make(map[string]any)
	}

	if schema.Ref != "" {
		schemaName := parseSchemaRef(schema.Ref)
		if schemaName != "" {
			if _, ok := visiting[schemaName]; ok {
				return make(map[string]any)
			}
			if s, ok := r.nameToSchema.Load(schemaName); ok {
				if resolvedSchema, ok := s.(*model.Schema); ok {
					return r.generateExample(resolvedSchema, visiting, depth+1)
				}
			}
		}
		return nil
	}

	switch schema.Type {
	case model.SchemaTypeString:
		if schema.Format == "date-time" {
			return "2024-01-01T00:00:00Z"
		}
		return "string"
	case model.SchemaTypeInteger:
		return 0
	case model.SchemaTypeNumber:
		return 0.0
	case model.SchemaTypeBoolean:
		return false
	case model.SchemaTypeArray:
		if schema.Items != nil {
			return []any{r.generateExample(schema.Items, visiting, depth+1)}
		}
		return []any{}
	case model.SchemaTypeObject:
		if len(schema.Properties) > 0 {
			if schema.Title != "" {
				visiting[schema.Title] = struct{}{}
				defer delete(visiting, schema.Title)
			}
			example := make(map[string]any)
			for name, prop := range schema.Properties {
				example[name] = r.generateExample(prop, visiting, depth+1)
			}
			return example
		}
		return make(map[string]any)
	}

	return nil
}

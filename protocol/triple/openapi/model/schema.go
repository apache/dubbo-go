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

package model

type SchemaType string

const (
	SchemaTypeString  SchemaType = "string"
	SchemaTypeInteger SchemaType = "integer"
	SchemaTypeNumber  SchemaType = "number"
	SchemaTypeBoolean SchemaType = "boolean"
	SchemaTypeObject  SchemaType = "object"
	SchemaTypeArray   SchemaType = "array"
)

type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 SchemaType         `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Required             bool               `json:"required,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty"`
	Example              any                `json:"example,omitempty"`
	GoType               string             `json:"-"`
}

func NewSchema() *Schema {
	return &Schema{
		Properties: make(map[string]*Schema),
	}
}

func (s *Schema) SetType(t SchemaType) *Schema {
	s.Type = t
	return s
}

func (s *Schema) SetFormat(format string) *Schema {
	s.Format = format
	return s
}

func (s *Schema) SetTitle(title string) *Schema {
	s.Title = title
	return s
}

func (s *Schema) SetDescription(desc string) *Schema {
	s.Description = desc
	return s
}

func (s *Schema) SetRequired(required bool) *Schema {
	s.Required = required
	return s
}

func (s *Schema) SetItems(items *Schema) *Schema {
	s.Items = items
	return s
}

func (s *Schema) AddProperty(name string, schema *Schema) *Schema {
	if s.Properties == nil {
		s.Properties = make(map[string]*Schema)
	}
	s.Properties[name] = schema
	return s
}

func (s *Schema) SetAdditionalProperties(ap any) *Schema {
	s.AdditionalProperties = ap
	return s
}

func (s *Schema) SetRef(ref string) *Schema {
	s.Ref = ref
	return s
}

func (s *Schema) SetGoType(goType string) *Schema {
	s.GoType = goType
	return s
}

func (s *Schema) SetExample(example any) *Schema {
	s.Example = example
	return s
}

func (s *Schema) AddEnum(value any) *Schema {
	s.Enum = append(s.Enum, value)
	return s
}

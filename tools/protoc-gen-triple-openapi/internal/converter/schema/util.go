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

package schema

import (
	"fmt"
	"strings"
)

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"

	"go.yaml.in/yaml/v4"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func messageToSchema(tt protoreflect.MessageDescriptor) (string, *base.Schema) {
	s := &base.Schema{
		Title:       string(tt.Name()),
		Description: ProtoDescription(tt),
		Type:        []string{"object"},
	}

	props := orderedmap.New[string, *base.SchemaProxy]()
	parent := base.CreateSchemaProxy(s)

	fields := tt.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		prop := fieldToSchema(parent, field)
		props.Set(field.JSONName(), prop)
	}

	s.Properties = props
	appendOneOfSchemas(s, tt.Oneofs())

	return string(tt.FullName()), s
}

func appendOneOfSchemas(s *base.Schema, oneofs protoreflect.OneofDescriptors) {
	for i := 0; i < oneofs.Len(); i++ {
		oneof := oneofs.Get(i)
		if oneof.IsSynthetic() {
			continue
		}

		s.AllOf = append(s.AllOf, oneofToSchema(oneof))
	}
}

func oneofToSchema(tt protoreflect.OneofDescriptor) *base.SchemaProxy {
	fields := tt.Fields()
	choices := make([]*base.SchemaProxy, 0, fields.Len()+1)
	notAnyOf := make([]*base.SchemaProxy, 0, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		notAnyOf = append(notAnyOf, requiredFieldSchema(fields.Get(i).JSONName()))
	}

	choices = append(choices, base.CreateSchemaProxy(&base.Schema{
		Type: []string{"object"},
		Not: base.CreateSchemaProxy(&base.Schema{
			AnyOf: notAnyOf,
		}),
	}))
	for i := 0; i < fields.Len(); i++ {
		choices = append(choices, requiredFieldSchema(fields.Get(i).JSONName()))
	}

	return base.CreateSchemaProxy(&base.Schema{
		OneOf: choices,
	})
}

func requiredFieldSchema(fieldName string) *base.SchemaProxy {
	return base.CreateSchemaProxy(&base.Schema{
		Type:     []string{"object"},
		Required: []string{fieldName},
	})
}

func fieldToSchema(parent *base.SchemaProxy, tt protoreflect.FieldDescriptor) *base.SchemaProxy {
	if tt.IsMap() {
		root := ScalarFieldToSchema(parent, tt, false)
		root.Title = string(tt.Name())
		root.Type = []string{"object"}
		value := tt.MapValue()
		switch value.Kind() {
		case protoreflect.MessageKind, protoreflect.EnumKind:
			root.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{A: ReferenceFieldToSchema(parent, value)}
		default:
			root.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{A: base.CreateSchemaProxy(ScalarFieldToSchema(parent, value, true))}
		}
		root.Description = ProtoDescription(tt)
		return base.CreateSchemaProxy(root)
	} else if tt.IsList() {
		var itemSchema *base.SchemaProxy
		switch tt.Kind() {
		case protoreflect.MessageKind:
			itemSchema = ReferenceFieldToSchema(parent, tt)
		case protoreflect.EnumKind:
			itemSchema = ReferenceFieldToSchema(parent, tt)
		default:
			itemSchema = base.CreateSchemaProxy(ScalarFieldToSchema(parent, tt, true))
		}
		s := &base.Schema{
			Title:       string(tt.Name()),
			ParentProxy: parent,
			Description: ProtoDescription(tt),
			Type:        []string{"array"},
			Items:       &base.DynamicValue[*base.SchemaProxy, bool]{A: itemSchema},
		}
		return base.CreateSchemaProxy(s)
	} else {
		switch tt.Kind() {
		case protoreflect.MessageKind, protoreflect.EnumKind:
			msg := ScalarFieldToSchema(parent, tt, false)
			ref := ReferenceFieldToSchema(parent, tt)
			extensions := orderedmap.New[string, *yaml.Node]()
			extensions.Set("$ref", CreateStringNode(ref.GetReference()))
			msg.Extensions = extensions
			return base.CreateSchemaProxy(msg)
		}

		s := ScalarFieldToSchema(parent, tt, false)
		return base.CreateSchemaProxy(s)
	}
}

func ScalarFieldToSchema(parent *base.SchemaProxy, tt protoreflect.FieldDescriptor, inContainer bool) *base.Schema {
	s := &base.Schema{
		ParentProxy: parent,
	}
	if !inContainer {
		s.Title = string(tt.Name())
		s.Description = ProtoDescription(tt)
	}

	switch tt.Kind() {
	case protoreflect.BoolKind:
		s.Type = []string{"boolean"}
	case protoreflect.Int32Kind, protoreflect.Sfixed32Kind, protoreflect.Sint32Kind: // int32 types
		s.Type = []string{"integer"}
		s.Format = "int32"
	case protoreflect.Fixed32Kind, protoreflect.Uint32Kind: // uint32 types
		s.Type = []string{"integer"}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind: // int64 types
		// NOTE: 64-bit integer types can be strings or numbers because they sometimes
		//       cannot fit into a JSON number type
		s.Type = []string{"integer", "string"}
		s.Format = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind: // uint64 types
		s.Type = []string{"integer", "string"}
		s.Format = "int64"
	case protoreflect.DoubleKind:
		s.Type = []string{"number"}
		s.Format = "double"
	case protoreflect.FloatKind:
		s.Type = []string{"number"}
		s.Format = "float"
	case protoreflect.StringKind:
		s.Type = []string{"string"}
	case protoreflect.BytesKind:
		s.Type = []string{"string"}
		s.Format = "byte"
	}
	return s
}

func enumToSchema(tt protoreflect.EnumDescriptor) (string, *base.Schema) {
	values := tt.Values()
	enum := make([]*yaml.Node, 0, values.Len())
	for i := 0; i < values.Len(); i++ {
		enum = append(enum, CreateStringNode(string(values.Get(i).Name())))
	}

	return string(tt.FullName()), &base.Schema{
		Title: string(tt.Name()),
		Type:  []string{"string"},
		Enum:  enum,
	}
}

func ReferenceFieldToSchema(parent *base.SchemaProxy, tt protoreflect.FieldDescriptor) *base.SchemaProxy {
	switch tt.Kind() {
	case protoreflect.MessageKind:
		return base.CreateSchemaProxyRef("#/components/schemas/" + string(tt.Message().FullName()))
	case protoreflect.EnumKind:
		return base.CreateSchemaProxyRef("#/components/schemas/" + string(tt.Enum().FullName()))
	default:
		panic(fmt.Errorf("ReferenceFieldToSchema called with unknown kind: %T", tt.Kind()))
	}
}

func CreateStringNode(str string) *yaml.Node {
	n := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: str,
	}
	return n
}

func ProtoDescription(descriptor protoreflect.Descriptor) string {
	return strings.TrimSpace(descriptor.ParentFile().SourceLocations().ByDescriptor(descriptor).LeadingComments)
}

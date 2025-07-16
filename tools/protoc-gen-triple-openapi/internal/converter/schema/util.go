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
)

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"

	"google.golang.org/protobuf/reflect/protoreflect"

	"gopkg.in/yaml.v3"
)

func messageToSchema(tt protoreflect.MessageDescriptor) (string, *base.Schema) {
	s := &base.Schema{
		Title: string(tt.Name()),
		// TODO: add Description
		Description: "",
		Type:        []string{"object"},
	}

	props := orderedmap.New[string, *base.SchemaProxy]()

	fields := tt.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		// TODO: handle oneof
		prop := fieldToSchema(base.CreateSchemaProxy(s), field)
		props.Set(field.JSONName(), prop)
	}

	s.Properties = props

	return string(tt.FullName()), s
}

func fieldToSchema(parent *base.SchemaProxy, tt protoreflect.FieldDescriptor) *base.SchemaProxy {
	if tt.IsMap() {
		// Handle maps
		root := ScalarFieldToSchema(parent, tt, false)
		root.Title = string(tt.Name())
		root.Type = []string{"object"}
		// TODO: todo
		root.Description = ""
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
			// TODO: todo
			Description: "",
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
		// TODO: add description
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

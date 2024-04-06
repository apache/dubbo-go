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

package generate

import (
	"fmt"
)

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func GenHessian2(gen *protogen.Plugin, file *protogen.File) {
	filename := file.GeneratedFilenamePrefix + ".hessian2.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	g.P("package ", file.GoPackageName)
	g.P()

	g.P("import hessian \"github.com/apache/dubbo-go-hessian2\"")
	g.P()

	for _, message := range file.Messages {
		genMessage(g, message)
	}
	genRegisterInitFunc(g, file)
}

func genMessage(g *protogen.GeneratedFile, m *protogen.Message) {
	if m.Desc.IsMapEntry() {
		return
	}
	g.AnnotateSymbol(m.GoIdent.String(), protogen.Annotation{
		Location: m.Location,
		Semantic: descriptorpb.GeneratedCodeInfo_Annotation_SET.Enum(),
	})

	g.P("type ", m.GoIdent, " struct {")

	genMessageFields(g, m)
	g.P("}")
	g.P()

	genMessageRelatedMethods(g, m)
}

func genMessageFields(g *protogen.GeneratedFile, m *protogen.Message) {
	for _, field := range m.Fields {
		genMessageField(g, m, field)
	}
}

func genMessageField(g *protogen.GeneratedFile, m *protogen.Message, field *protogen.Field) {
	goType, pointer := fieldGoType(g, field)
	if pointer {
		goType = "*" + goType
	}

	name := field.GoName
	g.AnnotateSymbol(m.GoIdent.GoName+"."+name, protogen.Annotation{
		Location: field.Location,
		Semantic: descriptorpb.GeneratedCodeInfo_Annotation_SET.Enum(),
	})
	g.P(name, " ", goType)
}

func genMessageRelatedMethods(g *protogen.GeneratedFile, m *protogen.Message) {
	g.P("func ", "(x *", m.GoIdent.GoName, ")", "JavaClassName() string {")
	// TODO(Yuukirn): get class name by extend field
	g.P("	return ", "\"org.test.service.", m.GoIdent.GoName, "\"")
	g.P("}")

	g.P()

	g.P("func ", "(x *", m.GoIdent.GoName, ")", "String() string {")
	g.P("	e := hessian.NewEncoder()")
	g.P("	err := e.Encode(x)")
	g.P("	if err != nil {")
	g.P("		return \"\"")
	g.P("	}")
	g.P("	return string(e.Buffer())")
	g.P("}")
	g.P()
}

func genRegisterInitFunc(g *protogen.GeneratedFile, f *protogen.File) {
	g.P("func init() {")
	for _, message := range f.Messages {
		g.P("hessian.RegisterPOJO(new(", message.GoIdent.GoName, "))")
	}
	g.P("}")
	g.P()
}

// fieldGoType returns the Go type used for a field.
//
// If it returns pointer=true, the struct field is a pointer to the type.
func fieldGoType(g *protogen.GeneratedFile, field *protogen.Field) (goType string, pointer bool) {
	if field.Desc.IsWeak() {
		return "struct{}", false
	}

	pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		goType = g.QualifiedGoIdent(field.Enum.GoIdent)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false // rely on nullability of slices for presence
	case protoreflect.MessageKind, protoreflect.GroupKind:
		goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		pointer = false // pointer captured as part of the type
	}
	switch {
	case field.Desc.IsList():
		return "[]" + goType, false
	case field.Desc.IsMap():
		keyType, _ := fieldGoType(g, field.Message.Fields[0])
		valType, _ := fieldGoType(g, field.Message.Fields[1])
		return fmt.Sprintf("map[%v]%v", keyType, valType), false
	}
	return goType, pointer
}

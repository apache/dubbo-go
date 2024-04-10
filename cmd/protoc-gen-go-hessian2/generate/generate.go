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
	"strconv"
)

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/proto/hessian2_extend"
)

var (
	ErrNoJavaClassName        = "should extend java class name to generate hessian2 code"
	ErrExtendedOptionNotMatch = "extended options not match"
)

func GenHessian2(gen *protogen.Plugin, file *protogen.File) {
	filename := file.GeneratedFilenamePrefix + ".hessian2.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	g.P("package ", file.GoPackageName)
	g.P()

	g.P("import hessian \"github.com/apache/dubbo-go-hessian2\"")
	g.P()

	for _, enum := range file.Enums {
		genEnum(g, enum)
	}

	for _, message := range file.Messages {
		genMessage(g, message, file)
	}
	genRegisterInitFunc(g, file)
}

func genEnum(g *protogen.GeneratedFile, e *protogen.Enum) {
	g.P("type ", e.GoIdent.GoName, " int32")
	g.P()

	genEnumValues(g, e)
	genEnumMaps(g, e)
	genEnumRelatedMethods(g, e)
}

func genEnumValues(g *protogen.GeneratedFile, e *protogen.Enum) {
	g.P("const (")
	for i, v := range e.Values {
		g.P(v.GoIdent.GoName, " ", e.GoIdent.GoName, " = ", i)
	}
	g.P(")")
	g.P()
}

func genEnumMaps(g *protogen.GeneratedFile, e *protogen.Enum) {
	g.P("// Enum value maps for ", e.GoIdent.GoName)
	g.P("var (")
	g.P(e.GoIdent.GoName, "_name = map[int32]string {")
	for i, v := range e.Values {
		g.P(i, ": \"", v.GoIdent.GoName, "\",")
	}
	g.P("}")

	g.P(e.GoIdent.GoName, "_enum_name = map[", e.GoIdent.GoName, "]string {")
	for _, v := range e.Values {
		g.P(v.GoIdent.GoName, ": \"", v.GoIdent.GoName, "\",")
	}
	g.P("}")

	g.P(e.GoIdent.GoName, "_value = map[string]int32 {")
	for i, v := range e.Values {
		g.P("\"", v.GoIdent.GoName, "\": ", i, ",")
	}
	g.P("}")

	g.P(e.GoIdent.GoName, "_enum_value = map[string]", e.GoIdent.GoName, "{")
	for _, v := range e.Values {
		g.P("\"", v.GoIdent.GoName, "\": ", v.GoIdent.GoName, ",")
	}
	g.P("}")

	g.P(")")
}

func genEnumRelatedMethods(g *protogen.GeneratedFile, e *protogen.Enum) {
	g.P("func (x ", e.GoIdent.GoName, ") Enum() *", e.GoIdent.GoName, " {")
	g.P("p := new(", e.GoIdent.GoName, ")")
	g.P("*p = x")
	g.P("return p")
	g.P("}")
	g.P()

	g.P("func (x ", e.GoIdent.GoName, ") String() string {")
	g.P("return ", e.GoIdent.GoName, "_enum_name[x]")
	g.P("}")
	g.P()

	g.P("func (x ", e.GoIdent.GoName, ") Number() int32 {")
	g.P("return ", e.GoIdent.GoName, "_value[x.String()]")
	g.P("}")
	g.P()
}

func genMessage(g *protogen.GeneratedFile, m *protogen.Message, f *protogen.File) {
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

	genMessageRelatedMethods(g, m, f)
	genNestedMessage(g, m, f)
}

func genNestedMessage(g *protogen.GeneratedFile, m *protogen.Message, f *protogen.File) {
	for _, msg := range m.Messages {
		genMessage(g, msg, f)
	}
}

func genMessageFields(g *protogen.GeneratedFile, m *protogen.Message) {
	for _, field := range m.Fields {
		genMessageField(g, m, field)
	}
}

func genMessageField(g *protogen.GeneratedFile, m *protogen.Message, field *protogen.Field) {
	goType := getGoType(g, field)

	name := field.GoName
	g.AnnotateSymbol(m.GoIdent.GoName+"."+name, protogen.Annotation{
		Location: field.Location,
		Semantic: descriptorpb.GeneratedCodeInfo_Annotation_SET.Enum(),
	})
	g.P(name, " ", goType)
}

func genMessageRelatedMethods(g *protogen.GeneratedFile, m *protogen.Message, f *protogen.File) {
	g.P("func ", "(x *", m.GoIdent.GoName, ")", "JavaClassName() string {")
	opts := m.Desc.Options().(*descriptorpb.MessageOptions)
	ext, err := proto.GetExtension(opts, hessian2_extend.E_MessageExtend)
	if errors.Is(err, proto.ErrMissingExtension) || ext == nil {
		panic(ErrNoJavaClassName)
	}
	hessian2MsgOpts, ok := ext.(*hessian2_extend.Hessian2MessageOptions)
	if !ok {
		panic(ErrExtendedOptionNotMatch)
	}

	g.P("	return \"", hessian2MsgOpts.JavaClassName, "\"")
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

	genMessageGetterMethod(g, m, f)
}

func genMessageGetterMethod(g *protogen.GeneratedFile, m *protogen.Message, f *protogen.File) {
	for _, field := range m.Fields {
		genFieldGetterMethod(g, field, m, f)
	}
}

func genFieldGetterMethod(g *protogen.GeneratedFile, field *protogen.Field, m *protogen.Message, f *protogen.File) {
	goType := getGoType(g, field)
	defaultValue := fieldDefaultValue(g, f, m, field)

	g.P("func (x *", m.GoIdent.GoName, ") Get", field.GoName, "() ", goType, "{")
	g.P("if x != nil {")
	g.P("return x.", field.GoName)
	g.P("}")
	g.P("return ", defaultValue)
	g.P("}")
	g.P()
}

func genRegisterInitFunc(g *protogen.GeneratedFile, f *protogen.File) {
	g.P("func init() {")
	for _, message := range f.Messages {
		g.P("hessian.RegisterPOJO(new(", message.GoIdent.GoName, "))")
		for _, inner := range message.Messages {
			g.P("hessian.RegisterPOJO(new(", inner.GoIdent.GoName, "))")
		}
	}
	g.P("}")
	g.P()
}

func getGoType(g *protogen.GeneratedFile, field *protogen.Field) (goType string) {
	goType, pointer := fieldGoType(g, field)
	if pointer {
		goType = "*" + goType
	}
	return
}

// below is helper func that copy from protobuf-gen-go
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

func fieldDefaultValue(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message, field *protogen.Field) string {
	if field.Desc.IsList() {
		return "nil"
	}
	if field.Desc.HasDefault() {
		defVarName := "Default_" + m.GoIdent.GoName + "_" + field.GoName
		if field.Desc.Kind() == protoreflect.BytesKind {
			return "append([]byte(nil), " + defVarName + "...)"
		}
		return defVarName
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return "false"
	case protoreflect.StringKind:
		return `""`
	case protoreflect.MessageKind, protoreflect.GroupKind, protoreflect.BytesKind:
		return "nil"
	case protoreflect.EnumKind:
		val := field.Enum.Values[0]
		if val.GoIdent.GoImportPath == f.GoImportPath {
			return g.QualifiedGoIdent(val.GoIdent)
		} else {
			// If the enum value is declared in a different Go package,
			// reference it by number since the name may not be correct.
			// See https://github.com/golang/protobuf/issues/513.
			return g.QualifiedGoIdent(field.Enum.GoIdent) + "(" + strconv.FormatInt(int64(val.Desc.Number()), 10) + ")"
		}
	default:
		return "0"
	}
}

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

package generator

import (
	"fmt"
	"strconv"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/proto/hessian2_extend"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	ErrNoMessageExtend        = "should extend message options to generate hessian2 code"
	ErrNoJavaClassName        = "should extend java class name to generate hessian2 code"
	ErrExtendedOptionNotMatch = "extended options not match"
)

func GenHessian2(g *protogen.GeneratedFile, hessian2Go *Hessian2Go) {
	genPreamble(g, hessian2Go)
	genPackage(g, hessian2Go)

	g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "dubbo-go-hessian2",
		GoImportPath: "github.com/apache/dubbo-go-hessian2",
	})

	for _, enum := range hessian2Go.Enums {
		genEnum(g, enum)
	}

	for _, message := range hessian2Go.Messages {
		genMessage(g, message)
	}

	genRegisterInitFunc(g, hessian2Go)
}

func genPreamble(g *protogen.GeneratedFile, hessian2Go *Hessian2Go) {
	g.P("// Code generated by protoc-gen-go-dubbo. DO NOT EDIT.")
	g.P()
	g.P("// Source: ", hessian2Go.Source)
	g.P("// Package: ", strings.ReplaceAll(hessian2Go.ProtoPackage, ".", "_"))
	g.P()
}

func genPackage(g *protogen.GeneratedFile, hessian2Go *Hessian2Go) {
	g.P("package ", hessian2Go.GoPackageName)
	g.P()
}

func genEnum(g *protogen.GeneratedFile, e *Enum) {
	if e.JavaClassName != "" {
		g.P("type ", e.GoIdent.GoName, " dubbo_go_hessian2.JavaEnum")
	} else {
		g.P("type ", e.GoIdent.GoName, " int32")
	}
	g.P()

	genEnumValues(g, e)
	genEnumMaps(g, e)
	genEnumRelatedMethods(g, e)
}

func genEnumValues(g *protogen.GeneratedFile, e *Enum) {
	g.P("const (")
	for i, v := range e.Values {
		g.P(v.GoIdent.GoName, " ", e.GoIdent.GoName, " = ", i)
	}
	g.P(")")
	g.P()
}

func genEnumMaps(g *protogen.GeneratedFile, e *Enum) {
	g.P("// Enum value maps for ", e.GoIdent.GoName)
	g.P("var (")
	g.P(e.GoIdent.GoName, "_name = map[int32]string {")
	for i, v := range e.Values {
		g.P(i, ": \"", v.GoIdent.GoName, "\",")
	}
	g.P("}")

	g.P(e.GoIdent.GoName, "_value = map[string]int32 {")
	for i, v := range e.Values {
		g.P("\"", v.GoIdent.GoName, "\": ", i, ",")
	}
	g.P("}")

	g.P(")")
}

func genEnumRelatedMethods(g *protogen.GeneratedFile, e *Enum) {
	g.P("func (x ", e.GoIdent.GoName, ") Enum() *", e.GoIdent.GoName, " {")
	g.P("p := new(", e.GoIdent.GoName, ")")
	g.P("*p = x")
	g.P("return p")
	g.P("}")
	g.P()

	g.P("func (x ", e.GoIdent.GoName, ") String() string {")
	g.P("return ", e.GoIdent.GoName, "_name[int32(x)]")
	g.P("}")
	g.P()

	g.P("func (x ", e.GoIdent.GoName, ") Number() int32 {")
	g.P("return ", e.GoIdent.GoName, "_value[x.String()]")
	g.P("}")
	g.P()

	if e.JavaClassName != "" {
		g.P("func (x ", e.GoIdent.GoName, ") EnumValue(s string) dubbo_go_hessian2.JavaEnum {")
		g.P("return dubbo_go_hessian2.JavaEnum(", e.GoIdent.GoName, "_value[x.String()])")
		g.P("}")
		g.P()

		g.P("func (x ", e.GoIdent.GoName, ") JavaClassName() string {")
		g.P("return \"", e.JavaClassName, "\"")
		g.P("}")
		g.P()
	}
}

func genMessage(g *protogen.GeneratedFile, m *Message) {
	if m.Desc.IsMapEntry() || m.ExtendArgs {
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
	genNestedMessage(g, m)
}

func genNestedMessage(g *protogen.GeneratedFile, m *Message) {
	for _, msg := range m.InnerMessages {
		genMessage(g, msg)
	}
}

func genInheritance(g *protogen.GeneratedFile, m *Message) {
	for _, msg := range m.InnerMessages {
		if !msg.IsInheritance {
			continue
		}
		g.P(msg.Message.GoIdent.GoName)
	}
}

func genMessageFields(g *protogen.GeneratedFile, m *Message) {
	genInheritance(g, m)

	for _, field := range m.Fields {
		genMessageField(g, m, field)
	}
}

func genMessageField(g *protogen.GeneratedFile, m *Message, field *Field) {
	name := field.GoName
	g.AnnotateSymbol(m.GoIdent.GoName+"."+name, protogen.Annotation{
		Location: field.Location,
		Semantic: descriptorpb.GeneratedCodeInfo_Annotation_SET.Enum(),
	})
	g.P(name, " ", field.Type)
}

func genMessageRelatedMethods(g *protogen.GeneratedFile, m *Message) {
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
	g.P("	e := dubbo_go_hessian2.NewEncoder()")
	g.P("	err := e.Encode(x)")
	g.P("	if err != nil {")
	g.P("		return \"\"")
	g.P("	}")
	g.P("	return string(e.Buffer())")
	g.P("}")
	g.P()

	genMessageGetterMethod(g, m)
}

func genMessageGetterMethod(g *protogen.GeneratedFile, m *Message) {
	for _, field := range m.Fields {
		genFieldGetterMethod(g, field, m)
	}
}

func genFieldGetterMethod(g *protogen.GeneratedFile, field *Field, m *Message) {
	g.P("func (x *", m.GoIdent.GoName, ") Get", field.GoName, "() ", field.Type, "{")
	g.P("if x != nil {")
	g.P("return x.", field.GoName)
	g.P("}")
	g.P("return ", field.DefaultValue)
	g.P("}")
	g.P()
}

func genRegisterInitFunc(g *protogen.GeneratedFile, hessian2Go *Hessian2Go) {
	g.P("func init() {")
	for _, message := range hessian2Go.Messages {
		if message.Desc.IsMapEntry() || message.ExtendArgs {
			continue
		}
		g.P("dubbo_go_hessian2.RegisterPOJO(new(", message.GoIdent.GoName, "))")
		for _, inner := range message.Messages {
			if inner.Desc.IsMapEntry() {
				continue
			}
			g.P("dubbo_go_hessian2.RegisterPOJO(new(", inner.GoIdent.GoName, "))")
		}
	}

	for _, e := range hessian2Go.Enums {
		g.P()
		if e.JavaClassName != "" {
			g.P("for v := range ", e.GoIdent.GoName, "_name {")
			for range e.Values {
				g.P("dubbo_go_hessian2.RegisterJavaEnum(", e.GoIdent.GoName, "(v))")
			}
			g.P("}")
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

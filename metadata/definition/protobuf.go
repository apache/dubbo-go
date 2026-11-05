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

package definition

import (
	"fmt"
	"reflect"
	"sort"
)

import (
	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// buildProtobufService describes a Triple IDL service from protobuf descriptors.
//
// Type names: method parameter and return types, and every message and enum,
// use the protobuf full name (greet.GreetRequest, google.protobuf.Empty). That
// is the name Triple IDL generic invocation uses for a message, and the
// definition's purpose is to let Admin build such calls. Scalar and container
// fields use the Java vocabulary the reflection builder already uses
// (java.lang.String, int, long, java.util.List<T>, java.util.Map<K,V>) rather
// than proto names like "string" or "int64". Admin resolves types by exact
// string and its debug form only knows default values for that vocabulary, so
// proto scalar names would render as null. Java Dubbo's ProtobufTypeBuilder
// produces the same shape from the generated Java classes. Unsigned and
// fixed-width integers collapse to int and long, as they do in generated Java
// code. bytes has no Java-vocabulary entry Admin understands and stays "bytes".
//
// Only unary methods are published. Admin's generic invocation is unary-only, so
// listing a streaming method would offer a call that cannot be made. Streaming
// methods are reported as skipped.
//
// The request type comes from the method's ReqInitFunc. The response type comes
// from the service descriptor that the generated .pb.go registers under the
// ServiceInfo interface name. If that descriptor's input type disagrees with
// ReqInitFunc, the method is skipped rather than guessed.
func buildProtobufService(u *common.URL, handler reflect.Type, info *common.ServiceInfo) (*ServiceDefinition, []SkippedMethod, error) {
	types := newTypeCollector()
	methods := make([]MethodDefinition, 0, len(info.Methods))
	var skips []SkippedMethod
	var service protoreflect.ServiceDescriptor
	if d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(info.InterfaceName)); err == nil {
		service, _ = d.(protoreflect.ServiceDescriptor)
	}
	for _, m := range info.Methods {
		if m.Type != constant.CallUnary {
			skips = append(skips, SkippedMethod{Name: m.Name, Reason: "streaming is not supported"})
			continue
		}
		var input, output protoreflect.MessageDescriptor
		if m.ReqInitFunc != nil {
			if req, ok := m.ReqInitFunc().(proto.Message); ok && req != nil && req.ProtoReflect().IsValid() {
				input = req.ProtoReflect().Descriptor()
			}
		}
		if service != nil {
			if method := service.Methods().ByName(protoreflect.Name(m.Name)); method != nil && !method.IsStreamingClient() && !method.IsStreamingServer() && input != nil && method.Input().FullName() == input.FullName() {
				output = method.Output()
			}
		}
		// No registered service descriptor means the ServiceInfo was written by
		// hand, or its interface name is not the proto service's full name. Fall
		// back to the handler's signature, which works only when its first result
		// is a generated message pointer. The fallback is not used when a
		// descriptor exists but disagrees with ReqInitFunc; that case is skipped.
		if output == nil && service == nil {
			if method, ok := handler.MethodByName(m.Name); ok {
				t := method.Type
				if t.NumOut() > 0 {
					result := t.Out(0)
					if result.Kind() == reflect.Pointer && result.Implements(reflect.TypeFor[proto.Message]()) {
						output = reflect.New(result.Elem()).Interface().(proto.Message).ProtoReflect().Descriptor()
					}
				}
			}
		}
		if input == nil || output == nil {
			skips = append(skips, SkippedMethod{Name: m.Name, Reason: "protobuf request or response descriptor unavailable"})
			continue
		}
		types.protobufMessage(input)
		types.protobufMessage(output)
		methods = append(methods, MethodDefinition{Name: m.Name, ParameterTypes: []string{string(input.FullName())}, ReturnType: string(output.FullName())})
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	sort.Slice(skips, func(i, j int) bool { return skips[i].Name < skips[j].Name })
	return &ServiceDefinition{CanonicalName: u.Interface(), Parameters: urlParameters(u), Methods: methods, Types: types.definitions()}, skips, nil
}

// protobufMessage registers a message definition and returns its full name.
//
// The definition is stored before its fields are walked, so a message that
// refers back to itself or an enclosing message terminates. Every field becomes a
// property keyed by its JSON name, including each member of a oneof. The
// document format cannot say "at most one of", so oneof exclusivity is not
// expressed.
func (c *typeCollector) protobufMessage(message protoreflect.MessageDescriptor) string {
	name := string(message.FullName())
	if _, ok := c.defs[name]; ok {
		return name
	}
	def := &TypeDefinition{Type: name, Properties: make(map[string]string)}
	c.defs[name] = def
	fields := message.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		def.Properties[field.JSONName()] = c.protobufField(field)
	}
	return name
}

// protobufField returns a field's type expression and registers any container
// definition it needs.
//
// Repeated fields are java.util.List<T>, not the T[] the reflection builder
// writes for slices, because generated Java code exposes them as List<T> and
// Java Dubbo describes them that way. Container elements are boxed
// (java.lang.Integer, not int), since a Java generic cannot hold a primitive.
// The Map spelling has no space after the comma, matching the reflection builder.
func (c *typeCollector) protobufField(field protoreflect.FieldDescriptor) string {
	if field.IsMap() {
		key := boxed(c.protobufSingular(field.MapKey()))
		value := boxed(c.protobufSingular(field.MapValue()))
		name := fmt.Sprintf("%s<%s,%s>", javaMap, key, value)
		c.defs[name] = &TypeDefinition{Type: name, Items: []string{key, value}}
		return name
	}
	item := c.protobufSingular(field)
	if field.IsList() {
		item = boxed(item)
		name := fmt.Sprintf("java.util.List<%s>", item)
		c.defs[name] = &TypeDefinition{Type: name, Items: []string{item}}
		return name
	}
	return item
}

func (c *typeCollector) protobufSingular(field protoreflect.FieldDescriptor) string {
	switch field.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return c.protobufMessage(field.Message())
	case protoreflect.EnumKind:
		enum := field.Enum()
		name := string(enum.FullName())
		if _, ok := c.defs[name]; !ok {
			def := &TypeDefinition{Type: name}
			for i := 0; i < enum.Values().Len(); i++ {
				def.Enums = append(def.Enums, string(enum.Values().Get(i).Name()))
			}
			c.defs[name] = def
		}
		return name
	case protoreflect.StringKind:
		return javaString
	case protoreflect.BoolKind:
		return javaBoolean
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return javaInt
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return javaLong
	case protoreflect.FloatKind:
		return javaFloat
	case protoreflect.DoubleKind:
		return javaDouble
	case protoreflect.BytesKind:
		return "bytes"
	default:
		return field.Kind().String()
	}
}

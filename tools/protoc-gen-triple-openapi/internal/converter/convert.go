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

package converter

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"github.com/swaggest/jsonschema-go"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	openapimodel "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func ConvertFrom(r io.Reader) (*pluginpb.CodeGeneratorResponse, error) {
	in, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read request: %w", err)
	}

	req := &pluginpb.CodeGeneratorRequest{}
	if err = proto.Unmarshal(in, req); err != nil {
		return nil, fmt.Errorf("can't unmarshal input: %w", err)
	}

	return convert(req)
}

func convert(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	genFiles := make(map[string]struct{}, len(req.FileToGenerate))
	for _, file := range req.FileToGenerate {
		genFiles[file] = struct{}{}
	}

	resolver, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: req.GetProtoFile(),
	})
	if err != nil {
		return nil, err
	}

	// TODO: consider basic OpenAPI file

	openapiDoc := &openapimodel.Document{}
	initializeDoc(openapiDoc)

	files := []*pluginpb.CodeGeneratorResponse_File{}
	for _, fileDesc := range req.GetProtoFile() {
		if _, ok := genFiles[fileDesc.GetName()]; !ok {
			continue
		}

		fd, err := resolver.FindFileByPath(fileDesc.GetName())
		if err != nil {
			return nil, err
		}

		openapiDoc.Info.Title = string(fd.FullName())
		// TODO: add info description
		openapiDoc.Info.Description = ""

		openapiDoc.Components, err = generateComponents(fd)
		if err != nil {
			return nil, err
		}

		items := orderedmap.New[string, *openapimodel.PathItem]()
		tags := []*base.Tag{}

		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)

			tags = append(tags, &base.Tag{
				Name: string(service.FullName()),
				// TODO: add serivce description
				Description: "",
			})

			methods := service.Methods()
			for j := 0; j < methods.Len(); j++ {
				md := methods.Get(j)

				// operation
				op := &openapimodel.Operation{
					Summary:     string(md.Name()),
					OperationId: string(md.FullName()),
					Tags:        []string{string(service.FullName())},
					// TODO: add operation description
					Description: "",
				}

				// Responses
				codeMap := orderedmap.New[string, *openapimodel.Response]()
				mediaType := orderedmap.New[string, *openapimodel.MediaType]()
				outputSchema := base.CreateSchemaProxyRef("#/components/schemas/" + string(md.Output().FullName()))

				mediaType.Set("application/json", &openapimodel.MediaType{Schema: outputSchema})
				codeMap.Set("200", &openapimodel.Response{
					Description: "Success",
					Content:     mediaType,
				})

				op.Responses = &openapimodel.Responses{
					Codes: codeMap,
					Default: &openapimodel.Response{
						Description: "Error",
						Content:     makeMediaTypes(base.CreateSchemaProxyRef("#/components/schemas/triple.error")),
					},
				}

				item := &openapimodel.PathItem{}
				item.Post = op

				items.Set("/"+string(service.FullName())+"/"+string(md.Name()), item)
			}
		}
		openapiDoc.Paths.PathItems = items
		openapiDoc.Tags = tags

		b, err := yaml.Marshal(openapiDoc)
		if err != nil {
			return nil, err
		}

		content := string(b)
		name := *fileDesc.Name
		filename := strings.TrimSuffix(name, filepath.Ext(name)) + ".openapi.yaml"
		files = append(files, &pluginpb.CodeGeneratorResponse_File{
			Name:              &filename,
			Content:           &content,
			GeneratedCodeInfo: &descriptorpb.GeneratedCodeInfo{},
		})
	}

	return &plugin.CodeGeneratorResponse{
		File: files,
	}, nil
}

func resolveJsonSchema(root *jsonschema.Schema, t protoreflect.Descriptor) *jsonschema.Schema {
	switch tt := t.(type) {
	case protoreflect.EnumDescriptor:
	case protoreflect.EnumValueDescriptor:
	case protoreflect.MessageDescriptor:
		s := &jsonschema.Schema{}
		s.WithID(string(t.FullName()))
		// TODO: add description
		s.WithDescription("TODO: message jsonschema description")
		s.WithType(jsonschema.Object.Type())
		fields := tt.Fields()
		children := make(map[string]jsonschema.SchemaOrBool, fields.Len())
		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			child := resolveJsonSchema(root, field)
			children[field.JSONName()] = jsonschema.SchemaOrBool{TypeObject: child}
		}
		s.WithProperties(children)
		return s
	case protoreflect.FieldDescriptor:
		s := &jsonschema.Schema{}
		s.WithID(string(tt.FullName()))
		s.WithDescription("TODO: filed jsonschema description")
		if tt.IsMap() {
			s.AdditionalProperties = &jsonschema.SchemaOrBool{TypeObject: resolveJsonSchema(root, tt.MapValue())}
		}
		switch tt.Kind() {
		case protoreflect.BoolKind:
			s.WithType(jsonschema.Boolean.Type())
		case protoreflect.EnumKind:
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind:
			s.WithType(jsonschema.Integer.Type())
		case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
			s.WithType(jsonschema.Integer.Type())
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind:
			s.WithType(jsonschema.Number.Type())
		case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
			s.WithType(jsonschema.Number.Type())
		case protoreflect.FloatKind:
			s.WithType(jsonschema.Number.Type())
		case protoreflect.DoubleKind:
			s.WithType(jsonschema.Number.Type())
		case protoreflect.StringKind:
			s.WithType(jsonschema.String.Type())
		case protoreflect.BytesKind:
			s.WithType(jsonschema.String.Type())
		case protoreflect.MessageKind:
			s.WithRef("#/components/schemas/" + string(tt.FullName()))
			s.WithType(jsonschema.Object.Type())
		}
		return s
	case protoreflect.OneofDescriptor:

	case protoreflect.FileDescriptor:
		s := &jsonschema.Schema{}
		s.WithID(string(t.FullName()))
		s.WithDescription("TODO: file jsonschema description")
		children := []jsonschema.SchemaOrBool{}
		enums := tt.Enums()
		for i := 0; i < enums.Len(); i++ {
			child := resolveJsonSchema(root, enums.Get(i))
			children = append(children, jsonschema.SchemaOrBool{TypeObject: child})
		}
		messages := tt.Messages()
		for i := 0; i < messages.Len(); i++ {
			child := resolveJsonSchema(root, messages.Get(i))
			children = append(children, jsonschema.SchemaOrBool{TypeObject: child})
		}
		s.WithItems(jsonschema.Items{SchemaArray: children})
		return s

	// We don't use these here
	case protoreflect.ServiceDescriptor:
	case protoreflect.MethodDescriptor:
	}

	return nil
}

func initializeDoc(doc *openapimodel.Document) {
	if doc.Version == "" {
		doc.Version = "3.1.0"
	}
	if doc.Info == nil {
		doc.Info = &base.Info{}
	}
	if doc.Paths == nil {
		doc.Paths = &openapimodel.Paths{}
	}
	if doc.Paths.PathItems == nil {
		doc.Paths.PathItems = orderedmap.New[string, *openapimodel.PathItem]()
	}
	if doc.Paths.Extensions == nil {
		doc.Paths.Extensions = orderedmap.New[string, *yaml.Node]()
	}
	if doc.Security == nil {
		doc.Security = []*base.SecurityRequirement{}
	}
	if doc.Extensions == nil {
		doc.Extensions = orderedmap.New[string, *yaml.Node]()
	}
	if doc.Webhooks == nil {
		doc.Webhooks = orderedmap.New[string, *openapimodel.PathItem]()
	}
	if doc.Index == nil {
		doc.Index = &index.SpecIndex{}
	}
	if doc.Rolodex == nil {
		doc.Rolodex = &index.Rolodex{}
	}
	if doc.Components == nil {
		doc.Components = &openapimodel.Components{}
	}
	initializeComponents(doc.Components)
}

func initializeComponents(components *openapimodel.Components) {
	if components.Schemas == nil {
		components.Schemas = orderedmap.New[string, *base.SchemaProxy]()
	}
	if components.Responses == nil {
		components.Responses = orderedmap.New[string, *openapimodel.Response]()
	}
	if components.Parameters == nil {
		components.Parameters = orderedmap.New[string, *openapimodel.Parameter]()
	}
	if components.Examples == nil {
		components.Examples = orderedmap.New[string, *base.Example]()
	}
	if components.RequestBodies == nil {
		components.RequestBodies = orderedmap.New[string, *openapimodel.RequestBody]()
	}
	if components.Headers == nil {
		components.Headers = orderedmap.New[string, *openapimodel.Header]()
	}
	if components.SecuritySchemes == nil {
		components.SecuritySchemes = orderedmap.New[string, *openapimodel.SecurityScheme]()
	}
	if components.Links == nil {
		components.Links = orderedmap.New[string, *openapimodel.Link]()
	}
	if components.Callbacks == nil {
		components.Callbacks = orderedmap.New[string, *openapimodel.Callback]()
	}
	if components.Extensions == nil {
		components.Extensions = orderedmap.New[string, *yaml.Node]()
	}
}

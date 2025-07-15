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
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	openapimodel "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
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

		// handle openapi info
		openapiDoc.Info = &base.Info{
			Title:       "Dubbo-go OpenAPI",
			Version:     "v1",
			Description: "dubbo-go generate OpenAPI docs.",
		}

		// handle openapi servers
		openapiDoc.Servers = append(openapiDoc.Servers, &openapimodel.Server{
			URL:         "http://0.0.0.0:20000",
			Description: "Dubbo-go Default Server",
		})

		// handle openapi components
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

				// RequestBody
				isRequired := true
				requestBodyMediaType := orderedmap.New[string, *openapimodel.MediaType]()
				requestSchema := base.CreateSchemaProxyRef("#/components/schemas/" + string(md.Input().FullName()))
				requestBodyMediaType.Set("application/json", &openapimodel.MediaType{Schema: requestSchema})
				op.RequestBody = &openapimodel.RequestBody{
					// TODO: description
					Description: "",
					Content:     requestBodyMediaType,
					Required:    &isRequired,
				}

				// Responses
				codeMap := orderedmap.New[string, *openapimodel.Response]()
				responseMediaType := orderedmap.New[string, *openapimodel.MediaType]()
				responseSchema := base.CreateSchemaProxyRef("#/components/schemas/" + string(md.Output().FullName()))

				responseMediaType.Set("application/json", &openapimodel.MediaType{Schema: responseSchema})
				codeMap.Set("200", &openapimodel.Response{
					Description: "Success",
					Content:     responseMediaType,
				})

				op.Responses = &openapimodel.Responses{
					Codes: codeMap,
					Default: &openapimodel.Response{
						Description: "Error",
						Content:     makeMediaTypes(base.CreateSchemaProxyRef("#/components/schemas/ErrorResponse")),
					},
				}

				item := &openapimodel.PathItem{}
				item.Post = op

				items.Set("/"+string(service.FullName())+"/"+string(md.Name()), item)
			}
		}
		openapiDoc.Paths.PathItems = items
		openapiDoc.Tags = tags

		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)

		err = encoder.Encode(openapiDoc)
		if err != nil {
			return nil, err
		}

		b := buf.Bytes()

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

func initializeDoc(doc *openapimodel.Document) {
	if doc.Version == "" {
		doc.Version = "3.0.1"
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

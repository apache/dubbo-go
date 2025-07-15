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
)

import (
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	openapimodel "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/reflect/protodesc"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"gopkg.in/yaml.v3"
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

	openapiDoc := &openapimodel.Document{
		Version: "3.0.1",
		Info:    &base.Info{},
		Paths: &openapimodel.Paths{
			PathItems:  orderedmap.New[string, *openapimodel.PathItem](),
			Extensions: orderedmap.New[string, *yaml.Node](),
		},
		Security:   []*base.SecurityRequirement{},
		Extensions: orderedmap.New[string, *yaml.Node](),
		Webhooks:   orderedmap.New[string, *openapimodel.PathItem](),
		Index:      &index.SpecIndex{},
		Rolodex:    &index.Rolodex{},
		Components: &openapimodel.Components{},
	}

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
			})

			methods := service.Methods()
			for j := 0; j < methods.Len(); j++ {
				md := methods.Get(j)

				// operation
				op := &openapimodel.Operation{
					OperationId: string(md.Name()),
					Tags:        []string{string(service.FullName())},
					// TODO: add operation description
				}

				// RequestBody
				isRequired := true
				requestSchema := base.CreateSchemaProxyRef("#/components/schemas/" + string(md.Input().FullName()))
				op.RequestBody = &openapimodel.RequestBody{
					// TODO: description
					Content:  makeMediaTypes(requestSchema),
					Required: &isRequired,
				}

				// Responses
				codeMap := orderedmap.New[string, *openapimodel.Response]()

				// status code 200
				response200Schema := base.CreateSchemaProxyRef("#/components/schemas/" + string(md.Output().FullName()))
				codeMap.Set("200", &openapimodel.Response{
					Description: "OK",
					Content:     makeMediaTypes(response200Schema),
				})

				codeMap.Set("400", newErrorResponse("Bad Request"))

				// status code 500
				codeMap.Set("500", newErrorResponse("Internal Server Error"))

				op.Responses = &openapimodel.Responses{
					Codes: codeMap,
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

func newErrorResponse(description string) *openapimodel.Response {
	responseSchema := base.CreateSchemaProxyRef("#/components/schemas/ErrorResponse")
	responseMediaType := makeMediaTypes(responseSchema)
	return &openapimodel.Response{
		Description: description,
		Content:     responseMediaType,
	}
}



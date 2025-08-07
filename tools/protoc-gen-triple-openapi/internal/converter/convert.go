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

import (
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi/constant"
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi/internal/options"
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
	opts, err := options.Generate(req.GetParameter())

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

	doc := &openapimodel.Document{
		Version: constant.OpenAPIDocVersion,
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
		doc.Info = &base.Info{
			Title:       constant.OpenAPIDocInfoTitle,
			Version:     constant.OpenAPIDocInfoVersion,
			Description: constant.OpenAPIDocInfoDescription,
		}

		// handle openapi servers
		doc.Servers = append(doc.Servers, &openapimodel.Server{
			URL:         constant.OpenAPIDocServerURL,
			Description: constant.OpenAPIDocServerDesription,
		})

		// handle openapi components
		doc.Components, err = generateComponents(fd)
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
				operation := &openapimodel.Operation{
					OperationId: string(md.Name()),
					Tags:        []string{string(service.FullName())},
					// TODO: add operation description
				}

				// RequestBody
				isRequired := true
				requestSchema := base.CreateSchemaProxyRef(constant.OpenAPIDocComponentsSchemaSuffix +
					string(md.Input().FullName()))
				operation.RequestBody = &openapimodel.RequestBody{
					// TODO: description
					Content:  makeMediaTypes(requestSchema),
					Required: &isRequired,
				}

				// Responses
				codeMap := orderedmap.New[string, *openapimodel.Response]()

				// status code 200
				response200Schema := base.CreateSchemaProxyRef(constant.OpenAPIDocComponentsSchemaSuffix +
					string(md.Output().FullName()))
				codeMap.Set(constant.StatusCode200, &openapimodel.Response{
					Description: constant.StatusCode200Description,
					Content:     makeMediaTypes(response200Schema),
				})

				// status code 400
				codeMap.Set(constant.StatusCode400, newErrorResponse(constant.StatusCode400Description))

				// status code 500
				codeMap.Set(constant.StatusCode500, newErrorResponse(constant.StatusCode500Description))

				operation.Responses = &openapimodel.Responses{
					Codes: codeMap,
				}

				item := &openapimodel.PathItem{}
				item.Post = operation

				items.Set("/"+string(service.FullName())+"/"+string(md.Name()), item)
			}
		}
		doc.Paths.PathItems = items
		doc.Tags = tags

		content, err := formatOpenapiDoc(opts, doc)
		if err != nil {
			return nil, err
		}

		var filename string

		name := *fileDesc.Name
		switch opts.Format {
		case constant.YAMLFormat:
			filename = strings.TrimSuffix(name, filepath.Ext(name)) + constant.YAMLFormatSuffix
		case constant.JSONFormat:
			filename = strings.TrimSuffix(name, filepath.Ext(name)) + constant.JSONFormatSuffix
		}
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

func formatOpenapiDoc(opts options.Options, doc *openapimodel.Document) (string, error) {
	switch opts.Format {
	case constant.YAMLFormat:
		return string(doc.RenderWithIndention(2)), nil
	case constant.JSONFormat:
		b, err := doc.RenderJSON("  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unknown format: %s", opts.Format)
	}
}

func newErrorResponse(description string) *openapimodel.Response {
	responseSchema := base.CreateSchemaProxyRef(constant.OpenAPIDocComponentsSchemaSuffix + "ErrorResponse")
	responseMediaType := makeMediaTypes(responseSchema)
	return &openapimodel.Response{
		Description: description,
		Content:     responseMediaType,
	}
}

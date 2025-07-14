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
	"gopkg.in/yaml.v3"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	openapimodel "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	libopenapiutils "github.com/pb33f/libopenapi/utils"

	"google.golang.org/protobuf/reflect/protoreflect"

	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi/internal/converter/schema"
)

func generateComponents(fd protoreflect.FileDescriptor) (*openapimodel.Components, error) {
	components := &openapimodel.Components{
		Schemas:         orderedmap.New[string, *base.SchemaProxy](),
		Responses:       orderedmap.New[string, *openapimodel.Response](),
		Parameters:      orderedmap.New[string, *openapimodel.Parameter](),
		Examples:        orderedmap.New[string, *base.Example](),
		RequestBodies:   orderedmap.New[string, *openapimodel.RequestBody](),
		Headers:         orderedmap.New[string, *openapimodel.Header](),
		SecuritySchemes: orderedmap.New[string, *openapimodel.SecurityScheme](),
		Links:           orderedmap.New[string, *openapimodel.Link](),
		Callbacks:       orderedmap.New[string, *openapimodel.Callback](),
		Extensions:      orderedmap.New[string, *yaml.Node](),
	}

	components.Schemas = schema.GenerateFileSchemas(fd)

	// triple error component
	tripleErrorProps := orderedmap.New[string, *base.SchemaProxy]()
	tripleErrorProps.Set("code", base.CreateSchemaProxy(&base.Schema{
		Description: "The status code, which should be an enum value of [google.rpc.Code][google.rpc.Code].",
		Type:        []string{"string"},
		Examples:    []*yaml.Node{libopenapiutils.CreateStringNode("unimplemented")},
		Enum: []*yaml.Node{
			libopenapiutils.CreateStringNode("invalid_argument"),
			libopenapiutils.CreateStringNode("unauthenticated"),
			libopenapiutils.CreateStringNode("permission_denied"),
			libopenapiutils.CreateStringNode("unimplemented"),
			libopenapiutils.CreateStringNode("deadline_exceeded"),
			libopenapiutils.CreateStringNode("aborted"),
			libopenapiutils.CreateStringNode("failed_precondition"),
			libopenapiutils.CreateStringNode("resource_exhausted"),
			libopenapiutils.CreateStringNode("internal"),
			libopenapiutils.CreateStringNode("unavailable"),
			libopenapiutils.CreateStringNode("unknown"),
		},
	}))
	tripleErrorProps.Set("message", base.CreateSchemaProxy(&base.Schema{
		Description: "A developer-facing error message, which should be in English. Any user-facing error message should be localized and sent in the [google.rpc.Status.details][google.rpc.Status.details] field, or localized by the client.",
		Type:        []string{"string"},
	}))
	tripleErrorProps.Set("detail", base.CreateSchemaProxyRef("#/components/schemas/google.protobuf.Any"))
	components.Schemas.Set("triple.error", base.CreateSchemaProxy(&base.Schema{
		Title:       "Triple Error",
		Description: `Error type returned by triple: https://cn.dubbo.apache.org/zh-cn/overview/reference/protocols/triple-spec/`,
		Properties:  tripleErrorProps,
		Type:        []string{"object"},
	}))

	googleAnyID, googleAnyShema := newGoogleAny()
	components.Schemas.Set(googleAnyID, base.CreateSchemaProxy(googleAnyShema))

	return components, nil
}

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
	"github.com/pb33f/libopenapi/datamodel/high/base"
	openapimodel "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"

	"google.golang.org/protobuf/reflect/protoreflect"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi/internal/converter/schema"
)

func generateComponents(fd protoreflect.FileDescriptor) (*openapimodel.Components, error) {
	components := &openapimodel.Components{
		Schemas: schema.GenerateFileSchemas(fd),
	}

	// triple error component
	errorResponseProps := orderedmap.New[string, *base.SchemaProxy]()
	errorResponseProps.Set("status", base.CreateSchemaProxy(&base.Schema{
		Description: "The status code.",
		Type:        []string{"string"},
	}))
	errorResponseProps.Set("message", base.CreateSchemaProxy(&base.Schema{
		Description: "A developer-facing error message.",
		Type:        []string{"string"},
	}))
	components.Schemas.Set("ErrorResponse", base.CreateSchemaProxy(&base.Schema{
		Title:       "ErrorResponse",
		Description: `Error type returned by triple: https://cn.dubbo.apache.org/zh-cn/overview/reference/protocols/triple-spec/`,
		Properties:  errorResponseProps,
		Type:        []string{"object"},
	}))

	return components, nil
}

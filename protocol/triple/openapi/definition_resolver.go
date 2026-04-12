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

package openapi

import (
	"reflect"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

type DefinitionResolver struct {
	config *global.OpenAPIConfig
}

func NewDefinitionResolver(cfg *global.OpenAPIConfig) *DefinitionResolver {
	return &DefinitionResolver{
		config: cfg,
	}
}

func (r *DefinitionResolver) Resolve(interfaceName string, info *serviceInfo) *model.OpenAPI {
	openAPI := model.NewOpenAPI()
	schemaResolver := NewSchemaResolver(r.config)

	openAPI.Info.Title = r.config.InfoTitle
	openAPI.Info.Version = r.resolveVersion()
	openAPI.Info.Description = r.config.InfoDescription

	seenMethods := make(map[string]bool)
	for _, method := range info.Methods {
		methodName := method.Name
		if seenMethods[strings.ToLower(methodName)] {
			continue
		}
		seenMethods[strings.ToLower(methodName)] = true

		httpMethods := r.determineHttpMethods()
		for _, httpMethod := range httpMethods {
			op := r.resolveOperation(method, httpMethod, interfaceName, schemaResolver)
			path := r.buildPath(interfaceName, methodName)
			pathItem := openAPI.GetOrAddPath(path)
			pathItem.SetOperation(strings.ToUpper(httpMethod), op)
		}
	}

	allSchemas := schemaResolver.GetSchemas()
	if len(allSchemas) > 0 {
		if openAPI.Components == nil {
			openAPI.Components = model.NewComponents()
		}
		for name, schema := range allSchemas {
			openAPI.Components.AddSchema(name, schema)
		}
	}

	return openAPI
}

func (r *DefinitionResolver) resolveOperation(method serviceMethodInfo, httpMethod string, tagName string, schemaResolver *SchemaResolver) *model.Operation {
	op := model.NewOperation()
	op.SetOperationId(tagName + "." + method.Name)
	op.SetGoMethod(method.Name)
	op.SetHttpMethod(strings.ToUpper(httpMethod))
	op.AddTag(tagName)

	reqSchema := r.resolveRequestSchema(method, schemaResolver)
	if reqSchema != nil {
		mediaTypes := r.config.DefaultConsumesMediaTypes
		if len(mediaTypes) == 0 {
			mediaTypes = []string{"application/json"}
		}

		reqBody := model.NewRequestBody()
		for _, mt := range mediaTypes {
			content := reqBody.GetOrAddContent(mt)
			content.SetSchema(reqSchema)
			example := schemaResolver.GenerateExample(reqSchema)
			if example != nil {
				content.SetExample(example)
			}
		}
		op.SetRequestBody(reqBody)
	}

	statusCodes := r.config.DefaultHttpStatusCodes
	if len(statusCodes) == 0 {
		statusCodes = []string{"200", "400", "500"}
	}

	mediaTypes := r.config.DefaultProducesMediaTypes
	if len(mediaTypes) == 0 {
		mediaTypes = []string{"application/json"}
	}

	for _, code := range statusCodes {
		resp := op.GetOrAddResponse(code)
		resp.Description = r.getStatusDescription(code)
		if code == "200" {
			respSchema := r.resolveResponseSchema(method, schemaResolver)
			for _, mt := range mediaTypes {
				content := resp.GetOrAddContent(mt)
				if respSchema != nil {
					content.SetSchema(respSchema)
					example := schemaResolver.GenerateExample(respSchema)
					if example != nil {
						content.SetExample(example)
					}
				}
			}
		}
	}

	return op
}

func (r *DefinitionResolver) resolveRequestSchema(method serviceMethodInfo, schemaResolver *SchemaResolver) *model.Schema {
	if method.Meta != nil {
		if reqType, ok := method.Meta["request.type"]; ok {
			if t, ok := reqType.(reflect.Type); ok {
				if t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				return schemaResolver.Resolve(t)
			}
		}
	}

	if method.ReqInitFunc == nil {
		return nil
	}

	req := method.ReqInitFunc()
	if req == nil {
		return nil
	}

	reqType := reflect.TypeOf(req)
	if reqType == nil {
		return nil
	}

	if reqType.Kind() == reflect.Slice {
		elemType := reqType.Elem()
		if elemType.Kind() == reflect.Interface {
			return nil
		}
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		if elemType.Kind() == reflect.Struct {
			return schemaResolver.Resolve(elemType)
		}
		return schemaResolver.Resolve(elemType)
	}

	return schemaResolver.Resolve(reqType)
}

func (r *DefinitionResolver) resolveResponseSchema(method serviceMethodInfo, schemaResolver *SchemaResolver) *model.Schema {
	if method.Meta != nil {
		if respType, ok := method.Meta["response.type"]; ok {
			if t, ok := respType.(reflect.Type); ok {
				if t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				return schemaResolver.Resolve(t)
			}
		}
	}

	return nil
}

func (r *DefinitionResolver) determineHttpMethods() []string {
	return []string{"POST"}
}

func (r *DefinitionResolver) buildPath(interfaceName, methodName string) string {
	return "/" + interfaceName + "/" + methodName
}

func (r *DefinitionResolver) getStatusDescription(code string) string {
	switch code {
	case "200":
		return "OK"
	case "400":
		return "Bad Request"
	case "401":
		return "Unauthorized"
	case "403":
		return "Forbidden"
	case "404":
		return "Not Found"
	case "500":
		return "Internal Server Error"
	default:
		return "Unknown"
	}
}

func (r *DefinitionResolver) resolveVersion() string {
	if r.config.InfoVersion != "" {
		return r.config.InfoVersion
	}
	return constant.OpenAPIDefaultVersion
}

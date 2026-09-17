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
	"fmt"
	"reflect"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/internal/httpbinding"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

type DefinitionResolver struct {
	config       *global.OpenAPIConfig
	useHTTPRules bool
}

func NewDefinitionResolver(cfg *global.OpenAPIConfig, useHTTPRules ...bool) *DefinitionResolver {
	if cfg == nil {
		cfg = global.DefaultOpenAPIConfig()
	}
	resolver := &DefinitionResolver{config: cfg}
	if len(useHTTPRules) > 0 {
		resolver.useHTTPRules = useHTTPRules[0]
	}
	return resolver
}

func (r *DefinitionResolver) Resolve(interfaceName string, info *serviceInfo) *model.OpenAPI {
	openAPI, _ := r.ResolveWithError(interfaceName, info)
	return openAPI
}

// ResolveWithError resolves a service definition and reports invalid HTTP
// annotations instead of silently falling back to the canonical Triple route.
func (r *DefinitionResolver) ResolveWithError(interfaceName string, info *serviceInfo) (*model.OpenAPI, error) {
	if info == nil {
		return nil, fmt.Errorf("service info is nil")
	}
	openAPI := model.NewOpenAPI()
	schemaResolver := NewSchemaResolver(r.config)

	openAPI.Info.Title = r.config.InfoTitle
	openAPI.Info.Version = r.resolveVersion()
	openAPI.Info.Description = r.config.InfoDescription

	seenMethods := make(map[string]bool)
	seenHTTPRoutes := make(map[string]string)
	for _, method := range info.Methods {
		methodName := method.Name
		if seenMethods[strings.ToLower(methodName)] {
			continue
		}
		seenMethods[strings.ToLower(methodName)] = true

		if r.useHTTPRules {
			descriptorName := interfaceName
			if info.InterfaceName != "" {
				descriptorName = info.InterfaceName
			}
			bindings, err := r.resolveHTTPBindings(descriptorName, methodName)
			if err != nil {
				return nil, fmt.Errorf("resolve HTTP bindings for %s.%s: %w", interfaceName, methodName, err)
			}
			if len(bindings) > 0 {
				for _, binding := range bindings {
					routeKey := httpbinding.CanonicalRouteKey(binding.Method, binding.PathTemplate)
					if previousRPC, exists := seenHTTPRoutes[routeKey]; exists {
						return nil, fmt.Errorf("duplicate HTTP route %q for RPCs %q and %q", routeKey, previousRPC, binding.RPC)
					}
					seenHTTPRoutes[routeKey] = binding.RPC
					op := r.resolveBindingOperation(method, binding, interfaceName, schemaResolver)
					path := normalizeHTTPPath(binding.PathTemplate)
					pathItem := openAPI.GetOrAddPath(path)
					pathItem.SetExtension("x-google-path-template", binding.PathTemplate)
					httpMethod := strings.ToUpper(binding.Method)
					if pathItem.GetOperation(httpMethod) != nil {
						return nil, fmt.Errorf("duplicate HTTP operation %s %s", httpMethod, path)
					}
					pathItem.SetOperation(httpMethod, op)
				}
				continue
			}
		}

		for _, httpMethod := range r.determineHttpMethods() {
			op := r.resolveOperation(method, httpMethod, interfaceName, schemaResolver)
			pathItem := openAPI.GetOrAddPath(r.buildPath(interfaceName, methodName))
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
	if r.useHTTPRules {
		addRuntimeErrorSchema(openAPI)
	}

	return openAPI, nil
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
				if t.Kind() == reflect.Pointer {
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
		if elemType.Kind() == reflect.Pointer {
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
				if t.Kind() == reflect.Pointer {
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
	case "409":
		return "Conflict"
	case "413":
		return "Payload Too Large"
	case "415":
		return "Unsupported Media Type"
	case "429":
		return "Too Many Requests"
	case "501":
		return "Not Implemented"
	case "503":
		return "Service Unavailable"
	case "504":
		return "Gateway Timeout"
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

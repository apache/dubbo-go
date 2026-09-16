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
	"regexp"
	"strconv"
	"strings"
)

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	openapimodel "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"

	"go.yaml.in/yaml/v4"

	"google.golang.org/protobuf/reflect/protoreflect"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi/constant"
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi/internal/converter/schema"
)

var openAPIHTTPPathVariablePattern = regexp.MustCompile(`\{([^}=]+)(?:=[^}]*)?\}`)
var openAPIHTTPPathConstraintPattern = regexp.MustCompile(`\{([^}=]+)=([^}]*)\}`)

type httpOpenAPIOperation struct {
	Path      string
	Method    string
	Operation *openapimodel.Operation
	Template  string
}

func buildHTTPOperations(service protoreflect.ServiceDescriptor, method protoreflect.MethodDescriptor, errorSchemaID string) ([]httpOpenAPIOperation, error) {
	bindings, err := resolveHTTPBindings(method)
	if err != nil {
		return nil, err
	}
	operations := make([]httpOpenAPIOperation, 0, len(bindings))
	for _, binding := range bindings {
		operation := &openapimodel.Operation{
			OperationId: httpOperationID(service, method, binding.Index),
			Tags:        []string{string(service.FullName())},
			Description: schema.ProtoDescription(method),
		}
		for _, pathField := range binding.PathFields {
			field, _ := validateHTTPFieldPath(method.Input(), pathField)
			required := true
			pathSchema := schema.FieldToSchema(field)
			if pathPattern := staticHTTPPathPattern(binding.PathTemplate, pathField); pathPattern != "" && pathSchema != nil && pathSchema.Schema() != nil {
				pathSchema.Schema().Pattern = pathPattern
			}
			operation.Parameters = append(operation.Parameters, &openapimodel.Parameter{
				Name:          pathField,
				In:            "path",
				Required:      &required,
				AllowReserved: true,
				Schema:        pathSchema,
			})
		}
		for _, queryField := range collectStaticQueryFields(method.Input(), binding) {
			operation.Parameters = append(operation.Parameters, &openapimodel.Parameter{
				Name:   queryField.Name,
				In:     "query",
				Schema: schema.FieldToSchema(queryField.Field),
			})
		}

		if binding.Body != "" {
			bodySchema, err := staticHTTPFieldSchema(method.Input(), binding.Body)
			if err != nil {
				return nil, err
			}
			required := true
			operation.RequestBody = &openapimodel.RequestBody{
				Content:  makeMediaTypes(bodySchema),
				Required: &required,
			}
		}

		codeMap := orderedmap.New[string, *openapimodel.Response]()
		responseSchema, err := staticHTTPResponseSchema(method.Output(), binding.ResponseBody)
		if err != nil {
			return nil, err
		}
		codeMap.Set(constant.StatusCode200, &openapimodel.Response{
			Description: constant.StatusCode200Description,
			Content:     makeMediaTypes(responseSchema),
		})
		codeMap.Set(constant.StatusCode400, newErrorResponse(constant.StatusCode400Description, errorSchemaID))
		codeMap.Set(constant.StatusCode500, newErrorResponse(constant.StatusCode500Description, errorSchemaID))
		operation.Responses = &openapimodel.Responses{Codes: codeMap}
		operations = append(operations, httpOpenAPIOperation{
			Path:      normalizeHTTPPath(binding.PathTemplate),
			Method:    strings.ToLower(binding.Method),
			Operation: operation,
			Template:  binding.PathTemplate,
		})
	}
	return operations, nil
}

func staticHTTPPathPattern(template, field string) string {
	for _, match := range openAPIHTTPPathConstraintPattern.FindAllStringSubmatch(template, -1) {
		if match[1] != field {
			continue
		}
		pattern := regexp.QuoteMeta(match[2])
		pattern = strings.ReplaceAll(pattern, `\*\*`, `.*`)
		pattern = strings.ReplaceAll(pattern, `\*`, `[^/]+`)
		return "^" + pattern + "$"
	}
	return ""
}

func httpOperationID(service protoreflect.ServiceDescriptor, method protoreflect.MethodDescriptor, index int) string {
	id := string(service.FullName()) + "." + string(method.Name())
	if index > 0 {
		id += ".binding" + strconv.Itoa(index)
	}
	return id
}

func normalizeHTTPPath(pathTemplate string) string {
	return openAPIHTTPPathVariablePattern.ReplaceAllString(pathTemplate, "{$1}")
}

func staticHTTPFieldSchema(message protoreflect.MessageDescriptor, path string) (*base.SchemaProxy, error) {
	if path == "*" {
		return base.CreateSchemaProxyRef("#/components/schemas/" + string(message.FullName())), nil
	}
	field, err := validateHTTPFieldPath(message, path)
	if err != nil {
		return nil, err
	}
	return schema.FieldToSchema(field), nil
}

func staticHTTPResponseSchema(message protoreflect.MessageDescriptor, path string) (*base.SchemaProxy, error) {
	if path == "" {
		return base.CreateSchemaProxyRef("#/components/schemas/" + string(message.FullName())), nil
	}
	return staticHTTPFieldSchema(message, path)
}

type staticQueryField struct {
	Name  string
	Field protoreflect.FieldDescriptor
}

func collectStaticQueryFields(message protoreflect.MessageDescriptor, binding httpBinding) []staticQueryField {
	fields := make([]staticQueryField, 0)
	collectStaticQueryFieldsAt(message, "", binding, &fields, 0)
	return fields
}

func collectStaticQueryFieldsAt(message protoreflect.MessageDescriptor, prefix string, binding httpBinding, fields *[]staticQueryField, depth int) {
	if message == nil || depth > 8 {
		return
	}
	for index := 0; index < message.Fields().Len(); index++ {
		field := message.Fields().Get(index)
		name := field.JSONName()
		fieldPath := name
		if prefix != "" {
			fieldPath = prefix + "." + name
		}
		if staticBodyField(fieldPath, binding.Body) || staticExactPathField(fieldPath, binding.PathFields) {
			continue
		}
		if field.Kind() == protoreflect.MessageKind && !field.IsList() && !field.IsMap() && !staticWellKnownMessage(field.Message()) {
			before := len(*fields)
			collectStaticQueryFieldsAt(field.Message(), fieldPath, binding, fields, depth+1)
			if len(*fields) != before {
				continue
			}
		}
		*fields = append(*fields, staticQueryField{Name: fieldPath, Field: field})
	}
}

func staticBodyField(fieldPath, body string) bool {
	if body == "*" {
		return true
	}
	if body == "" {
		return false
	}
	return staticSamePath(fieldPath, body) || staticPathPrefix(body, fieldPath)
}

func staticExactPathField(fieldPath string, pathFields []string) bool {
	for _, pathField := range pathFields {
		if staticSamePath(fieldPath, pathField) {
			return true
		}
	}
	return false
}

func staticPathPrefix(prefix, value string) bool {
	prefixParts := strings.Split(prefix, ".")
	valueParts := strings.Split(value, ".")
	if len(prefixParts) > len(valueParts) {
		return false
	}
	for index := range prefixParts {
		if !staticSamePathPart(prefixParts[index], valueParts[index]) {
			return false
		}
	}
	return true
}

func staticSamePath(first, second string) bool {
	firstParts := strings.Split(first, ".")
	secondParts := strings.Split(second, ".")
	if len(firstParts) != len(secondParts) {
		return false
	}
	for index := range firstParts {
		if !staticSamePathPart(firstParts[index], secondParts[index]) {
			return false
		}
	}
	return true
}

func staticSamePathPart(first, second string) bool {
	first = strings.ReplaceAll(first, "_", "")
	second = strings.ReplaceAll(second, "_", "")
	return strings.EqualFold(first, second)
}

func staticWellKnownMessage(message protoreflect.MessageDescriptor) bool {
	if message == nil {
		return true
	}
	switch message.FullName() {
	case "google.protobuf.Timestamp", "google.protobuf.Duration", "google.protobuf.Any", "google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.ListValue", "google.protobuf.Empty":
		return true
	default:
		return false
	}
}

func setHTTPPathExtension(item *openapimodel.PathItem, template string) {
	if item.Extensions == nil {
		item.Extensions = orderedmap.New[string, *yaml.Node]()
	}
	item.Extensions.Set("x-google-path-template", schema.CreateStringNode(template))
}

func setHTTPPathOperation(item *openapimodel.PathItem, method string, operation *openapimodel.Operation) error {
	var existing *openapimodel.Operation
	switch method {
	case "get":
		existing = item.Get
	case "put":
		existing = item.Put
	case "post":
		existing = item.Post
	case "delete":
		existing = item.Delete
	case "patch":
		existing = item.Patch
	}
	if existing != nil {
		return fmt.Errorf("duplicate HTTP operation %s %s", strings.ToUpper(method), operation.OperationId)
	}
	switch method {
	case "get":
		item.Get = operation
	case "put":
		item.Put = operation
	case "post":
		item.Post = operation
	case "delete":
		item.Delete = operation
	case "patch":
		item.Patch = operation
	default:
		return fmt.Errorf("unsupported HTTP method %q", method)
	}
	return nil
}

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
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

import (
	"dubbo.apache.org/dubbo-go/v3/internal/httpbinding"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

const runtimeErrorSchemaName = "Triple-ErrorResponse"

var openAPIPathVariablePattern = regexp.MustCompile(`\{([^}=]+)(?:=[^}]*)?\}`)
var openAPIPathConstraintPattern = regexp.MustCompile(`\{([^}=]+)=([^}]*)\}`)

func (r *DefinitionResolver) resolveHTTPBindings(interfaceName, methodName string) []httpbinding.HTTPBinding {
	bindings, err := httpbinding.Resolve(protoreflect.FullName(interfaceName + "." + methodName))
	if err != nil {
		return nil
	}
	return bindings
}

func normalizeHTTPPath(template string) string {
	return openAPIPathVariablePattern.ReplaceAllString(template, "{$1}")
}

func (r *DefinitionResolver) resolveBindingOperation(method serviceMethodInfo, binding httpbinding.HTTPBinding, tagName string, schemaResolver *SchemaResolver) *model.Operation {
	op := model.NewOperation().
		SetOperationId(bindingOperationID(tagName, method.Name, binding.Index)).
		SetGoMethod(method.Name).
		SetHttpMethod(binding.Method).
		AddTag(tagName)

	if binding.Body != "" {
		requestSchema := r.resolveBindingFieldSchema(method, binding.Body, schemaResolver)
		if binding.Body == "*" {
			requestSchema = r.resolveRequestSchema(method, schemaResolver)
		}
		if requestSchema != nil {
			op.SetRequestBody(r.newRequestBody(requestSchema, schemaResolver))
		}
	}

	for _, pathField := range binding.PathFields {
		pathSchema := r.resolveBindingFieldSchema(method, pathField, schemaResolver)
		if pathPattern := openAPIPathPattern(binding.PathTemplate, pathField); pathPattern != "" {
			pathSchema.Pattern = pathPattern
		}
		parameter := model.NewParameter(pathField, "path").
			SetRequired(true).
			SetAllowReserved(true).
			SetSchema(pathSchema)
		op.AddParameter(parameter)
	}
	for _, queryField := range r.resolveQueryFields(method, binding, schemaResolver) {
		parameter := model.NewParameter(queryField.Name, "query").
			SetSchema(queryField.Schema).
			SetDescription(queryField.Description)
		op.AddParameter(parameter)
	}

	r.addResponses(op, method, binding.ResponseBody, tagName+"."+method.Name, schemaResolver)
	return op
}

func openAPIPathPattern(template, field string) string {
	for _, match := range openAPIPathConstraintPattern.FindAllStringSubmatch(template, -1) {
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

func bindingOperationID(interfaceName, methodName string, index int) string {
	operationID := interfaceName + "." + methodName
	if index > 0 {
		operationID += ".binding" + formatIndex(index)
	}
	return operationID
}

func formatIndex(index int) string {
	return strconv.Itoa(index)
}

func (r *DefinitionResolver) newRequestBody(schema *model.Schema, schemaResolver *SchemaResolver) *model.RequestBody {
	body := model.NewRequestBody().SetRequired(true)
	for _, mediaType := range r.requestMediaTypes() {
		content := body.GetOrAddContent(mediaType).SetSchema(schema)
		if example := schemaResolver.GenerateExample(schema); example != nil {
			content.SetExample(example)
		}
	}
	return body
}

func (r *DefinitionResolver) requestMediaTypes() []string {
	if len(r.config.DefaultConsumesMediaTypes) > 0 {
		return r.config.DefaultConsumesMediaTypes
	}
	return []string{"application/json"}
}

func (r *DefinitionResolver) responseMediaTypes() []string {
	if len(r.config.DefaultProducesMediaTypes) > 0 {
		return r.config.DefaultProducesMediaTypes
	}
	return []string{"application/json"}
}

func (r *DefinitionResolver) addResponses(op *model.Operation, method serviceMethodInfo, responseBody, rpc string, schemaResolver *SchemaResolver) {
	statusCodes := r.config.DefaultHttpStatusCodes
	if len(statusCodes) == 0 {
		statusCodes = []string{"200", "400", "500"}
	}
	for _, code := range statusCodes {
		response := op.GetOrAddResponse(code)
		response.Description = r.getStatusDescription(code)
		for _, mediaType := range r.responseMediaTypes() {
			content := response.GetOrAddContent(mediaType)
			if code == "200" {
				if responseBody == "" {
					responseSchema := r.resolveResponseSchema(method, schemaResolver)
					if responseSchema == nil {
						responseSchema = schemaResolver.Resolve(responseReflectType(method, rpc))
					}
					content.SetSchema(responseSchema)
				} else {
					content.SetSchema(r.resolveBindingFieldSchemaFromResponse(method, responseBody, rpc, schemaResolver))
				}
			} else {
				content.SetSchema(&model.Schema{Ref: "#/components/schemas/" + runtimeErrorSchemaName})
			}
		}
	}
}

type openAPIQueryField struct {
	Name        string
	Description string
	Schema      *model.Schema
}

func (r *DefinitionResolver) resolveQueryFields(method serviceMethodInfo, binding httpbinding.HTTPBinding, schemaResolver *SchemaResolver) []openAPIQueryField {
	requestType := requestReflectType(method)
	if requestType == nil || binding.Body == "*" {
		return nil
	}
	fields := make([]openAPIQueryField, 0)
	collectQueryFields(requestType, "", binding, schemaResolver, &fields, 0)
	return fields
}

func collectQueryFields(t reflect.Type, prefix string, binding httpbinding.HTTPBinding, schemaResolver *SchemaResolver, fields *[]openAPIQueryField, depth int) {
	if depth > 8 {
		return
	}
	t = indirectType(t)
	if t == nil || t.Kind() != reflect.Struct || isWellKnownStruct(t) {
		return
	}
	for index := 0; index < t.NumField(); index++ {
		field := t.Field(index)
		if field.PkgPath != "" || field.Anonymous {
			continue
		}
		name := reflectJSONName(field)
		if name == "" || name == "-" {
			continue
		}
		fieldPath := name
		if prefix != "" {
			fieldPath = prefix + "." + name
		}
		if isBodyField(fieldPath, binding.Body) || isExactPathField(fieldPath, binding.PathFields) {
			continue
		}
		fieldType := indirectType(field.Type)
		if fieldType != nil && fieldType.Kind() == reflect.Struct && !isWellKnownStruct(fieldType) && hasExportedFields(fieldType) {
			before := len(*fields)
			collectQueryFields(field.Type, fieldPath, binding, schemaResolver, fields, depth+1)
			if len(*fields) != before {
				continue
			}
		}
		*fields = append(*fields, openAPIQueryField{
			Name:   fieldPath,
			Schema: schemaResolver.Resolve(field.Type),
		})
	}
}

func (r *DefinitionResolver) resolveBindingFieldSchema(method serviceMethodInfo, path string, schemaResolver *SchemaResolver) *model.Schema {
	return schemaResolver.Resolve(reflectFieldType(requestReflectType(method), path))
}

func (r *DefinitionResolver) resolveBindingFieldSchemaFromResponse(method serviceMethodInfo, path, rpc string, schemaResolver *SchemaResolver) *model.Schema {
	return schemaResolver.Resolve(reflectFieldType(responseReflectType(method, rpc), path))
}

func requestReflectType(method serviceMethodInfo) reflect.Type {
	if method.Meta != nil {
		if requestType, ok := method.Meta["request.type"].(reflect.Type); ok {
			return requestType
		}
	}
	if method.ReqInitFunc == nil {
		return nil
	}
	request := method.ReqInitFunc()
	if request == nil {
		return nil
	}
	t := reflect.TypeOf(request)
	if t.Kind() == reflect.Slice {
		return nil
	}
	return t
}

func responseReflectType(method serviceMethodInfo, rpc string) reflect.Type {
	if method.Meta != nil {
		if responseType, ok := method.Meta["response.type"].(reflect.Type); ok {
			return responseType
		}
	}
	if rpc == "" {
		return nil
	}
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(rpc))
	if err != nil {
		return nil
	}
	methodDescriptor, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		return nil
	}
	messageType, err := protoregistry.GlobalTypes.FindMessageByName(methodDescriptor.Output().FullName())
	if err != nil || messageType == nil {
		return nil
	}
	return reflect.TypeOf(messageType.New().Interface())
}

func reflectFieldType(t reflect.Type, path string) reflect.Type {
	if t == nil {
		return nil
	}
	for _, part := range strings.Split(path, ".") {
		field, ok := reflectField(t, part)
		if !ok {
			return nil
		}
		t = field.Type
	}
	return t
}

func reflectField(t reflect.Type, name string) (reflect.StructField, bool) {
	t = indirectType(t)
	if t == nil || t.Kind() != reflect.Struct {
		return reflect.StructField{}, false
	}
	if field, ok := t.FieldByName(name); ok {
		return field, true
	}
	if field, ok := t.FieldByName(toGoFieldName(name)); ok {
		return field, true
	}
	for index := 0; index < t.NumField(); index++ {
		field := t.Field(index)
		if reflectJSONName(field) == name {
			return field, true
		}
	}
	return reflect.StructField{}, false
}

func reflectJSONName(field reflect.StructField) string {
	protobufTag := field.Tag.Get("protobuf")
	for _, part := range strings.Split(protobufTag, ",") {
		if strings.HasPrefix(part, "json=") {
			if name := strings.TrimPrefix(part, "json="); name != "" {
				return name
			}
		}
	}
	tag := field.Tag.Get("json")
	if name, _, ok := strings.Cut(tag, ","); ok && name != "" {
		return name
	}
	if tag != "" && !strings.Contains(tag, ",") {
		return tag
	}
	return lowerCamel(field.Name)
}

func toGoFieldName(name string) string {
	var result strings.Builder
	upper := true
	for _, char := range name {
		if char == '_' || char == '-' {
			upper = true
			continue
		}
		if upper {
			result.WriteRune(unicode.ToUpper(char))
			upper = false
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

func lowerCamel(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && (t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface) {
		t = t.Elem()
	}
	return t
}

func hasExportedFields(t reflect.Type) bool {
	for index := 0; index < t.NumField(); index++ {
		if t.Field(index).PkgPath == "" && !t.Field(index).Anonymous {
			return true
		}
	}
	return false
}

func isWellKnownStruct(t reflect.Type) bool {
	return t.PkgPath() == "time" && t.Name() == "Time"
}

func isBodyField(fieldPath, body string) bool {
	if body == "*" {
		return true
	}
	if body == "" {
		return false
	}
	return samePath(fieldPath, body) || pathPrefix(body, fieldPath)
}

func isExactPathField(fieldPath string, pathFields []string) bool {
	for _, pathField := range pathFields {
		if samePath(fieldPath, pathField) {
			return true
		}
	}
	return false
}

func pathPrefix(prefix, value string) bool {
	prefixParts := strings.Split(prefix, ".")
	valueParts := strings.Split(value, ".")
	if len(prefixParts) > len(valueParts) {
		return false
	}
	for index := range prefixParts {
		if !samePathPart(prefixParts[index], valueParts[index]) {
			return false
		}
	}
	return true
}

func samePath(first, second string) bool {
	firstParts := strings.Split(first, ".")
	secondParts := strings.Split(second, ".")
	if len(firstParts) != len(secondParts) {
		return false
	}
	for index := range firstParts {
		if !samePathPart(firstParts[index], secondParts[index]) {
			return false
		}
	}
	return true
}

func samePathPart(first, second string) bool {
	return first == second || toGoFieldName(first) == toGoFieldName(second)
}

func addRuntimeErrorSchema(openAPI *model.OpenAPI) {
	if openAPI.Components == nil {
		openAPI.Components = model.NewComponents()
	}
	if _, exists := openAPI.Components.Schemas[runtimeErrorSchemaName]; exists {
		return
	}
	schema := model.NewSchema().SetType(model.SchemaTypeObject).SetTitle(runtimeErrorSchemaName)
	schema.AddProperty("code", model.NewSchema().SetType(model.SchemaTypeInteger).SetFormat("int32"))
	schema.AddProperty("message", model.NewSchema().SetType(model.SchemaTypeString))
	schema.AddProperty("details", model.NewSchema().SetType(model.SchemaTypeArray).SetItems(model.NewSchema().SetType(model.SchemaTypeObject)))
	openAPI.Components.AddSchema(runtimeErrorSchemaName, schema)
}

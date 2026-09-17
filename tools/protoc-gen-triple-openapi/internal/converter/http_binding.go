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
	"net/http"
	"regexp"
	"strings"
)

import (
	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type httpBinding struct {
	Method       string
	PathTemplate string
	Body         string
	ResponseBody string
	PathFields   []string
	Index        int
}

var httpPathVariablePattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.]*)(?:=[^{}]+)?\}`)
var httpCanonicalVariablePattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.]*)(?:=([^{}]+))?\}`)

func resolveHTTPBindings(method protoreflect.MethodDescriptor) ([]httpBinding, error) {
	options, ok := method.Options().(*descriptorpb.MethodOptions)
	if !ok || options == nil || !proto.HasExtension(options, annotations.E_Http) {
		return nil, nil
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, fmt.Errorf("method %q uses google.api.http but is streaming", method.FullName())
	}
	rule, ok := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)
	if !ok || rule == nil {
		return nil, fmt.Errorf("method %q has an invalid google.api.http option", method.FullName())
	}
	rules := append([]*annotations.HttpRule{rule}, rule.GetAdditionalBindings()...)
	bindings := make([]httpBinding, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for index, current := range rules {
		if current == nil {
			return nil, fmt.Errorf("method %q has a nil HTTP binding", method.FullName())
		}
		if index > 0 && len(current.GetAdditionalBindings()) > 0 {
			return nil, fmt.Errorf("method %q has nested additional_bindings", method.FullName())
		}
		httpMethod, pathTemplate, err := httpPattern(current)
		if err != nil {
			return nil, err
		}
		pathFields, err := validateHTTPPath(method.Input(), pathTemplate)
		if err != nil {
			return nil, err
		}
		if body := current.GetBody(); body != "" {
			if err := validateHTTPBodyMethod(httpMethod, body); err != nil {
				return nil, err
			}
		}
		if body := current.GetBody(); body != "" && body != "*" {
			if _, err := validateHTTPFieldPath(method.Input(), body); err != nil {
				return nil, fmt.Errorf("invalid body field %q: %w", body, err)
			}
			for _, pathField := range pathFields {
				if httpFieldPathsOverlap(body, pathField) {
					return nil, fmt.Errorf("body field %q overlaps path field %q", body, pathField)
				}
			}
		}
		if responseBody := current.GetResponseBody(); responseBody != "" && responseBody != "*" {
			if _, err := validateHTTPFieldPath(method.Output(), responseBody); err != nil {
				return nil, fmt.Errorf("invalid response_body field %q: %w", responseBody, err)
			}
		}
		binding := httpBinding{
			Method:       httpMethod,
			PathTemplate: pathTemplate,
			Body:         current.GetBody(),
			ResponseBody: current.GetResponseBody(),
			PathFields:   pathFields,
			Index:        index,
		}
		key := canonicalHTTPRouteKey(binding.Method, binding.PathTemplate)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("method %q has duplicate HTTP binding %q", method.FullName(), key)
		}
		seen[key] = struct{}{}
		bindings = append(bindings, binding)
	}
	return bindings, nil
}

func canonicalHTTPRouteKey(method, path string) string {
	path = httpCanonicalVariablePattern.ReplaceAllStringFunc(path, func(variable string) string {
		matches := httpCanonicalVariablePattern.FindStringSubmatch(variable)
		pattern := "*"
		if len(matches) > 2 && matches[2] != "" {
			pattern = matches[2]
		}
		return "{" + pattern + "}"
	})
	return strings.ToUpper(strings.TrimSpace(method)) + " " + path
}

func validateHTTPBodyMethod(method, body string) error {
	if body == "" {
		return nil
	}
	switch method {
	case http.MethodGet:
		return fmt.Errorf("HTTP method GET must not specify a request body")
	case http.MethodDelete:
		return fmt.Errorf("HTTP method DELETE must not specify a request body")
	default:
		return nil
	}
}

func httpPattern(rule *annotations.HttpRule) (string, string, error) {
	switch pattern := rule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet, pattern.Get, nil
	case *annotations.HttpRule_Put:
		return http.MethodPut, pattern.Put, nil
	case *annotations.HttpRule_Post:
		return http.MethodPost, pattern.Post, nil
	case *annotations.HttpRule_Delete:
		return http.MethodDelete, pattern.Delete, nil
	case *annotations.HttpRule_Patch:
		return http.MethodPatch, pattern.Patch, nil
	case *annotations.HttpRule_Custom:
		return "", "", fmt.Errorf("custom HTTP method %q is not supported", pattern.Custom.GetKind())
	default:
		return "", "", fmt.Errorf("HTTP method and path are required")
	}
}

func validateHTTPPath(message protoreflect.MessageDescriptor, pathTemplate string) ([]string, error) {
	if pathTemplate == "" || !strings.HasPrefix(pathTemplate, "/") {
		return nil, fmt.Errorf("invalid HTTP path template %q", pathTemplate)
	}
	matches := httpPathVariablePattern.FindAllStringSubmatch(pathTemplate, -1)
	if strings.Count(pathTemplate, "{") != len(matches) || strings.Count(pathTemplate, "}") != len(matches) {
		return nil, fmt.Errorf("invalid HTTP path template %q", pathTemplate)
	}
	fields := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		fieldPath := match[1]
		field, err := validateHTTPFieldPath(message, fieldPath)
		if err != nil {
			return nil, fmt.Errorf("invalid path field %q: %w", fieldPath, err)
		}
		if field.IsList() || field.IsMap() || !isSupportedHTTPPathField(field) {
			return nil, fmt.Errorf("path field %q must be a non-repeated primitive", fieldPath)
		}
		if _, exists := seen[fieldPath]; !exists {
			seen[fieldPath] = struct{}{}
			fields = append(fields, fieldPath)
		}
	}
	return fields, nil
}

func validateHTTPFieldPath(message protoreflect.MessageDescriptor, path string) (protoreflect.FieldDescriptor, error) {
	if path == "" {
		return nil, fmt.Errorf("field path is empty")
	}
	parts := strings.Split(path, ".")
	var field protoreflect.FieldDescriptor
	for index, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("empty field name")
		}
		field = message.Fields().ByName(protoreflect.Name(part))
		if field == nil {
			field = message.Fields().ByJSONName(part)
		}
		if field == nil {
			return nil, fmt.Errorf("field %q not found in %q", part, message.FullName())
		}
		if index == len(parts)-1 {
			return field, nil
		}
		if field.IsList() || field.IsMap() || field.Message() == nil {
			return nil, fmt.Errorf("field %q is not a singular message", field.FullName())
		}
		message = field.Message()
	}
	return field, nil
}

func httpFieldPathsOverlap(first, second string) bool {
	firstParts := strings.Split(first, ".")
	secondParts := strings.Split(second, ".")
	if len(firstParts) != len(secondParts) {
		return false
	}
	for index, part := range firstParts {
		if strings.ToLower(strings.ReplaceAll(part, "_", "")) != strings.ToLower(strings.ReplaceAll(secondParts[index], "_", "")) {
			return false
		}
	}
	return true
}

func isSupportedHTTPPathField(field protoreflect.FieldDescriptor) bool {
	if field == nil {
		return false
	}
	if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
		return true
	}
	if field.Message() == nil {
		return false
	}
	switch field.Message().FullName() {
	case "google.protobuf.Timestamp", "google.protobuf.Duration",
		"google.protobuf.DoubleValue", "google.protobuf.FloatValue",
		"google.protobuf.Int64Value", "google.protobuf.Int32Value",
		"google.protobuf.UInt64Value", "google.protobuf.UInt32Value",
		"google.protobuf.BoolValue", "google.protobuf.StringValue",
		"google.protobuf.BytesValue", "google.protobuf.FieldMask",
		"google.protobuf.Value", "google.protobuf.Struct":
		return true
	default:
		return false
	}
}

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

package httpbinding

import (
	"fmt"
	"net/http"
	"strings"

	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Resolve finds a method in the global protobuf registry and resolves its HTTP bindings.
func Resolve(rpc protoreflect.FullName) ([]HTTPBinding, error) {
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(rpc)
	if err != nil {
		return nil, fmt.Errorf("find RPC descriptor %q: %w", rpc, err)
	}
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is not a protobuf method", rpc)
	}
	return ResolveMethod(method)
}

// ResolveMethod resolves the primary google.api.http rule and its first-level additional bindings.
func ResolveMethod(method protoreflect.MethodDescriptor) ([]HTTPBinding, error) {
	if method == nil {
		return nil, fmt.Errorf("method descriptor is nil")
	}
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
	bindings := make([]HTTPBinding, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for index, current := range rules {
		if current == nil {
			return nil, fmt.Errorf("method %q has a nil HTTP binding", method.FullName())
		}
		if index > 0 && len(current.GetAdditionalBindings()) > 0 {
			return nil, fmt.Errorf("method %q has nested additional_bindings", method.FullName())
		}
		binding, err := resolveRule(method, current, index)
		if err != nil {
			return nil, fmt.Errorf("resolve HTTP binding for %q: %w", method.FullName(), err)
		}
		if _, duplicate := seen[binding.Key()]; duplicate {
			return nil, fmt.Errorf("method %q has duplicate HTTP binding %q", method.FullName(), binding.Key())
		}
		seen[binding.Key()] = struct{}{}
		bindings = append(bindings, binding)
	}
	return bindings, nil
}

func resolveRule(method protoreflect.MethodDescriptor, rule *annotations.HttpRule, index int) (HTTPBinding, error) {
	httpMethod, pathTemplate, err := resolvePattern(rule)
	if err != nil {
		return HTTPBinding{}, err
	}
	pathFields, err := resolvePathFields(httpMethod, pathTemplate)
	if err != nil {
		return HTTPBinding{}, fmt.Errorf("invalid path template %q: %w", pathTemplate, err)
	}
	for _, path := range pathFields {
		field, fieldErr := validateFieldPath(method.Input(), path)
		if fieldErr != nil {
			return HTTPBinding{}, fmt.Errorf("invalid path field %q: %w", path, fieldErr)
		}
		if field.IsList() || field.IsMap() {
			return HTTPBinding{}, fmt.Errorf("path field %q must not be repeated or mapped", path)
		}
	}
	if body := rule.GetBody(); body != "" && body != "*" {
		if _, err = validateFieldPath(method.Input(), body); err != nil {
			return HTTPBinding{}, fmt.Errorf("invalid body field %q: %w", body, err)
		}
	}
	if responseBody := rule.GetResponseBody(); responseBody != "" {
		if _, err = validateFieldPath(method.Output(), responseBody); err != nil {
			return HTTPBinding{}, fmt.Errorf("invalid response_body field %q: %w", responseBody, err)
		}
	}

	return HTTPBinding{
		RPC:          string(method.FullName()),
		Method:       httpMethod,
		PathTemplate: pathTemplate,
		Body:         rule.GetBody(),
		ResponseBody: rule.GetResponseBody(),
		PathFields:   pathFields,
		Index:        index,
	}, nil
}

func resolvePattern(rule *annotations.HttpRule) (string, string, error) {
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

func validateFieldPath(message protoreflect.MessageDescriptor, path string) (protoreflect.FieldDescriptor, error) {
	parts := strings.Split(path, ".")
	for index, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("empty field name")
		}
		fields := message.Fields()
		field := fields.ByName(protoreflect.Name(part))
		if field == nil {
			field = fields.ByJSONName(part)
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
	return nil, fmt.Errorf("field path is empty")
}

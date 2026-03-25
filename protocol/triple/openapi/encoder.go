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
	"encoding/json"
	"strings"
)

import (
	"gopkg.in/yaml.v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

type Encoder struct{}

func NewEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Encode(openAPI *model.OpenAPI, format string, pretty bool) (string, error) {
	if openAPI == nil {
		openAPI = model.NewOpenAPI()
	}

	data := e.toMap(openAPI)

	switch strings.ToLower(format) {
	case "json", "":
		if pretty {
			b, err := json.MarshalIndent(data, "", "  ")
			return string(b), err
		}
		b, err := json.Marshal(data)
		return string(b), err
	case "yaml", "yml":
		b, err := yaml.Marshal(data)
		return string(b), err
	default:
		b, err := json.Marshal(data)
		return string(b), err
	}
}

func (e *Encoder) toMap(openAPI *model.OpenAPI) map[string]any {
	result := make(map[string]any)

	result["openapi"] = openAPI.OpenAPI

	if openAPI.Info != nil {
		result["info"] = e.infoToMap(openAPI.Info)
	}

	if len(openAPI.Paths) > 0 {
		result["paths"] = e.pathsToMap(openAPI.Paths)
	}

	if openAPI.Components != nil {
		result["components"] = e.componentsToMap(openAPI.Components)
	}

	return result
}

func (e *Encoder) pathsToMap(paths map[string]*model.PathItem) map[string]any {
	m := make(map[string]any)
	for path, item := range paths {
		m[path] = e.pathItemToMap(item)
	}
	return m
}

func (e *Encoder) pathItemToMap(item *model.PathItem) map[string]any {
	m := make(map[string]any)
	if item.Get != nil {
		m["get"] = e.operationToMap(item.Get)
	}
	if item.Put != nil {
		m["put"] = e.operationToMap(item.Put)
	}
	if item.Post != nil {
		m["post"] = e.operationToMap(item.Post)
	}
	if item.Delete != nil {
		m["delete"] = e.operationToMap(item.Delete)
	}
	if item.Options != nil {
		m["options"] = e.operationToMap(item.Options)
	}
	if item.Head != nil {
		m["head"] = e.operationToMap(item.Head)
	}
	if item.Patch != nil {
		m["patch"] = e.operationToMap(item.Patch)
	}
	if item.Trace != nil {
		m["trace"] = e.operationToMap(item.Trace)
	}
	return m
}

func (e *Encoder) infoToMap(info *model.Info) map[string]any {
	m := make(map[string]any)
	if info.Title != "" {
		m["title"] = info.Title
	}
	if info.Description != "" {
		m["description"] = info.Description
	}
	if info.Version != "" {
		m["version"] = info.Version
	}
	return m
}

func (e *Encoder) operationToMap(op *model.Operation) map[string]any {
	m := make(map[string]any)

	if len(op.Tags) > 0 {
		m["tags"] = op.Tags
	}
	if op.OperationId != "" {
		m["operationId"] = op.OperationId
	}
	if op.RequestBody != nil {
		m["requestBody"] = e.requestBodyToMap(op.RequestBody)
	}
	if len(op.Responses) > 0 {
		resps := make(map[string]any)
		for code, resp := range op.Responses {
			resps[code] = e.responseToMap(resp)
		}
		m["responses"] = resps
	}

	return m
}

func (e *Encoder) requestBodyToMap(r *model.RequestBody) map[string]any {
	m := make(map[string]any)
	if r.Required {
		m["required"] = true
	}
	if len(r.Content) > 0 {
		content := make(map[string]any)
		for mt, c := range r.Content {
			content[mt] = e.mediaTypeToMap(c)
		}
		m["content"] = content
	}
	return m
}

func (e *Encoder) mediaTypeToMap(m *model.MediaType) map[string]any {
	result := make(map[string]any)
	if m.Schema != nil {
		result["schema"] = e.schemaToMap(m.Schema)
	}
	if m.Example != nil {
		result["example"] = m.Example
	}
	return result
}

func (e *Encoder) responseToMap(r *model.ApiResponse) map[string]any {
	m := make(map[string]any)
	m["description"] = r.Description
	if len(r.Content) > 0 {
		content := make(map[string]any)
		for mt, c := range r.Content {
			content[mt] = e.mediaTypeToMap(c)
		}
		m["content"] = content
	}
	return m
}

func (e *Encoder) schemaToMap(s *model.Schema) map[string]any {
	m := make(map[string]any)

	if s.Ref != "" {
		m["$ref"] = s.Ref
		return m
	}

	if s.Type != "" {
		m["type"] = string(s.Type)
	}
	if s.Format != "" {
		m["format"] = s.Format
	}
	if s.Title != "" {
		m["title"] = s.Title
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if s.Required {
		m["required"] = true
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Items != nil {
		m["items"] = e.schemaToMap(s.Items)
	}
	if len(s.Properties) > 0 {
		props := make(map[string]any)
		for name, prop := range s.Properties {
			props[name] = e.schemaToMap(prop)
		}
		m["properties"] = props
	}
	if s.AdditionalProperties != nil {
		m["additionalProperties"] = s.AdditionalProperties
	}
	if s.Example != nil {
		m["example"] = s.Example
	}

	return m
}

func (e *Encoder) componentsToMap(c *model.Components) map[string]any {
	m := make(map[string]any)

	if len(c.Schemas) > 0 {
		schemas := make(map[string]any)
		for name, s := range c.Schemas {
			schemas[name] = e.schemaToMap(s)
		}
		m["schemas"] = schemas
	}

	return m
}

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
	"net/http"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type RequestHandler struct {
	service *DefaultService
	config  *global.OpenAPIConfig
}

func NewRequestHandler(service *DefaultService, config *global.OpenAPIConfig) *RequestHandler {
	return &RequestHandler{
		service: service,
		config:  config,
	}
}

func (h *RequestHandler) Handle(req *http.Request) (string, string, bool) {
	path := req.URL.Path
	basePath := h.config.Path

	apiDocsPath := basePath + "/api-docs"
	if strings.HasPrefix(path, apiDocsPath) {
		return h.handleAPIDocs(path, basePath)
	}

	if strings.HasPrefix(path, basePath) {
		return h.handleOpenAPI(path, basePath)
	}

	return "", "", false
}

func (h *RequestHandler) handleAPIDocs(path, basePath string) (string, string, bool) {
	apiDocsPath := basePath + "/api-docs"
	group := ""
	format := "json"

	if path == apiDocsPath || path == apiDocsPath+"/" {
		group = DefaultGroup
	} else {
		pathPart := strings.TrimPrefix(path, apiDocsPath+"/")
		if strings.HasSuffix(pathPart, ".json") {
			group = strings.TrimSuffix(pathPart, ".json")
			format = "json"
		} else if strings.HasSuffix(pathPart, ".yaml") || strings.HasSuffix(pathPart, ".yml") {
			group = strings.TrimSuffix(pathPart, ".yaml")
			group = strings.TrimSuffix(group, ".yml")
			format = "yaml"
		} else {
			group = pathPart
		}
	}

	if group == "" {
		group = DefaultGroup
	}

	return h.getOpenAPIContent(group, format)
}

func (h *RequestHandler) handleOpenAPI(path, basePath string) (string, string, bool) {
	group := ""
	format := "json"

	openAPIPathJSON := basePath + "/openapi.json"
	openAPIPathYAML := basePath + "/openapi.yaml"
	openAPIPathYML := basePath + "/openapi.yml"

	switch path {
	case basePath, basePath + "/":
		group = DefaultGroup
	case openAPIPathJSON:
		group = DefaultGroup
		format = "json"
	case openAPIPathYAML:
		group = DefaultGroup
		format = "yaml"
	case openAPIPathYML:
		group = DefaultGroup
		format = "yaml"
	default:
		pathPart := strings.TrimPrefix(path, basePath+"/")
		if strings.HasSuffix(pathPart, ".json") {
			group = strings.TrimSuffix(pathPart, ".json")
			format = "json"
		} else if strings.HasSuffix(pathPart, ".yaml") || strings.HasSuffix(pathPart, ".yml") {
			group = strings.TrimSuffix(pathPart, ".yaml")
			group = strings.TrimSuffix(group, ".yml")
			format = "yaml"
		} else {
			group = pathPart
		}
	}

	if group == "" {
		group = DefaultGroup
	}

	return h.getOpenAPIContent(group, format)
}

func (h *RequestHandler) getOpenAPIContent(group, format string) (string, string, bool) {
	openAPI := h.service.GetOpenAPI(&OpenAPIRequest{
		Group:  group,
		Format: format,
	})

	content, err := h.service.GetEncoder().Encode(openAPI, format, true)
	if err != nil {
		return "", "", false
	}

	contentType := ContentTypeJSON
	if format == "yaml" || format == "yml" {
		contentType = ContentTypeYAML
	}

	return content, contentType, true
}

func (h *RequestHandler) GetService() *DefaultService {
	return h.service
}

func (h *RequestHandler) GetConfig() *global.OpenAPIConfig {
	return h.config
}

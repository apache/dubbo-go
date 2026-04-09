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
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/global"
)

type OpenAPIIntegration struct {
	service        *DefaultService
	requestHandler *RequestHandler
	swaggerHandler *SwaggerUIHandler
	redocHandler   *RedocHandler
}

func NewOpenAPIIntegration(cfg *global.OpenAPIConfig) *OpenAPIIntegration {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	cfg.Init()
	svc := NewDefaultService(cfg)
	integration := &OpenAPIIntegration{
		service:        svc,
		requestHandler: NewRequestHandler(svc, cfg),
		swaggerHandler: NewSwaggerUIHandler(svc, cfg),
		redocHandler:   NewRedocHandler(svc, cfg),
	}
	logger.Info("OpenAPI service initialized")
	return integration
}

func (o *OpenAPIIntegration) RegisterService(interfaceName string, info *common.ServiceInfo, openapiGroup string, dubboGroup string, dubboVersion string) {
	if o != nil && o.service != nil {
		o.service.RegisterService(interfaceName, info, openapiGroup, dubboGroup, dubboVersion)
	}
}

type HTTPHandler struct {
	integration *OpenAPIIntegration
}

func NewHTTPHandler(integration *OpenAPIIntegration) *HTTPHandler {
	return &HTTPHandler{integration: integration}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.integration == nil || h.integration.service == nil || !h.integration.service.config.Enabled {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"OpenAPI is not available","status":"404"}`))
		return
	}

	if content, contentType, ok := h.integration.swaggerHandler.Handle(r); ok {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
		return
	}

	if content, contentType, ok := h.integration.redocHandler.Handle(r); ok {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
		return
	}

	if content, contentType, ok := h.integration.requestHandler.Handle(r); ok {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not Found"))
}

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
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/global"
)

var (
	globalService   *DefaultService
	globalHandler   *RequestHandler
	swaggerHandler  *SwaggerUIHandler
	redocHandler    *RedocHandler
	serviceInitOnce sync.Once
)

func InitService(cfg *global.OpenAPIConfig) {
	serviceInitOnce.Do(func() {
		globalService = NewDefaultService(cfg)
		globalHandler = NewRequestHandler(globalService, cfg)
		swaggerHandler = NewSwaggerUIHandler(globalService, cfg)
		redocHandler = NewRedocHandler(globalService, cfg)
		logger.Info("OpenAPI service initialized")
	})
}

func RegisterService(interfaceName string, info *common.ServiceInfo, openapiGroup string) {
	if globalService != nil {
		globalService.RegisterService(interfaceName, info, openapiGroup)
	}
}

type HTTPHandler struct{}

func NewHTTPHandler() *HTTPHandler {
	return &HTTPHandler{}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if globalService == nil || !globalService.config.Enabled {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"OpenAPI is not available","status":"404"}`))
		return
	}

	if content, contentType, ok := swaggerHandler.Handle(r); ok {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
		return
	}

	if content, contentType, ok := redocHandler.Handle(r); ok {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
		return
	}

	if content, contentType, ok := globalHandler.Handle(r); ok {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not Found"))
}

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
	"net/http"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type RedocHandler struct {
	service *DefaultService
	config  *global.OpenAPIConfig
}

func NewRedocHandler(service *DefaultService, config *global.OpenAPIConfig) *RedocHandler {
	return &RedocHandler{
		service: service,
		config:  config,
	}
}

func (h *RedocHandler) Handle(req *http.Request) (string, string, bool) {
	path := req.URL.Path
	redocPath := h.config.Path + "/redoc"

	if !strings.HasPrefix(path, redocPath) {
		return "", "", false
	}

	resPath := strings.TrimPrefix(path, redocPath)
	if resPath == "" || resPath == "/" {
		group := req.URL.Query().Get("group")
		if group == "" {
			group = DefaultGroup
		}
		return h.handleIndex(group), ContentTypeHTML, true
	}

	resPath = strings.TrimPrefix(resPath, "/")
	requestPath := strings.TrimSuffix(resPath, ".html")

	if requestPath == "index" {
		group := req.URL.Query().Get("group")
		if group == "" {
			group = DefaultGroup
		}
		return h.handleIndex(group), ContentTypeHTML, true
	}

	return "", "", false
}

func (h *RedocHandler) handleIndex(group string) string {
	cdn := h.config.GetSetting(SettingKeyRedocCDN)
	if cdn == "" {
		cdn = DefaultRedocCDN
	}

	return fmt.Sprintf(`<!-- Under a "MIT License" license, see https://github.com/Redocly/redoc/blob/main/LICENSE -->
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Redoc</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
<redoc spec-url="%s/api-docs/%s.json"></redoc>
<script src="%s/redoc.standalone.js"></script>
</body>
</html>
`, h.config.Path, group, cdn)
}

func (h *RedocHandler) GetService() *DefaultService {
	return h.service
}

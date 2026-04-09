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
	"fmt"
	"net/http"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

type SwaggerUIHandler struct {
	service *DefaultService
	config  *global.OpenAPIConfig
}

func NewSwaggerUIHandler(service *DefaultService, config *global.OpenAPIConfig) *SwaggerUIHandler {
	return &SwaggerUIHandler{
		service: service,
		config:  config,
	}
}

func (h *SwaggerUIHandler) Handle(req *http.Request) (string, string, bool) {
	path := req.URL.Path
	swaggerUIPath := h.config.Path + "/swagger-ui"

	if !strings.HasPrefix(path, swaggerUIPath) {
		return "", "", false
	}

	resPath := strings.TrimPrefix(path, swaggerUIPath)
	if resPath == "" || resPath == "/" {
		return h.handleIndex(), constant.OpenAPIContentTypeHTML, true
	}

	resPath = strings.TrimPrefix(resPath, "/")
	requestPath := strings.TrimSuffix(resPath, ".html")
	requestPath = strings.TrimSuffix(requestPath, ".js")

	switch requestPath {
	case "index":
		return h.handleIndex(), constant.OpenAPIContentTypeHTML, true
	case "swagger-config":
		return h.handleSwaggerConfig(), constant.OpenAPIContentTypeJSON, true
	default:
		return "", "", false
	}
}

func (h *SwaggerUIHandler) handleIndex() string {
	cdn := h.config.GetSetting(constant.OpenAPISettingKeySwaggerUICDN)
	if cdn == "" {
		cdn = constant.OpenAPIDefaultSwaggerUICDN
	}

	settings := h.buildSettings()

	return fmt.Sprintf(`<!-- Under an "Apache License 2.0" license, see https://github.com/swagger-api/swagger-ui/blob/master/LICENSE -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Swagger UI</title>
    <link rel="icon" type="image/png" href="%s/favicon-32x32.png" sizes="32x32" />
    <link rel="icon" type="image/png" href="%s/favicon-16x16.png" sizes="16x16" />
    <link rel="stylesheet" type="text/css" href="%s/swagger-ui.css"/>
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin: 0;
            background: #fafafa;
        }
    </style>
</head>

<body>
<div id="swagger-ui"></div>
<script src="%s/swagger-ui-bundle.js" charset="UTF-8"></script>
<script src="%s/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
<script src="%s/swagger-initializer.js" charset="UTF-8"></script>
<script>
    window.onload = function () {
        window.ui = SwaggerUIBundle({
            url: "",
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            layout: "StandaloneLayout",
            "configUrl": "./swagger-config",
            "displayRequestDuration": true,
            "operationsSorter": "method"%s
        });
    };
</script>
</body>
</html>
`, cdn, cdn, cdn, cdn, cdn, cdn, settings)
}

func (h *SwaggerUIHandler) buildSettings() string {
	var sb strings.Builder
	for key, value := range h.config.Settings {
		if strings.HasPrefix(key, constant.OpenAPISettingKeySwaggerUISettings) {
			settingKey := strings.TrimPrefix(key, constant.OpenAPISettingKeySwaggerUISettings)
			sb.WriteString(fmt.Sprintf(",\n            \"%s\": %s", settingKey, value))
		}
	}
	return sb.String()
}

func (h *SwaggerUIHandler) handleSwaggerConfig() string {
	groups := h.service.GetOpenAPIGroups()

	urls := make([]map[string]string, 0, len(groups))
	for _, group := range groups {
		urls = append(urls, map[string]string{
			"name": group,
			"url":  h.config.Path + "/api-docs/" + group + ".json",
		})
	}

	config := map[string]any{
		"urls": urls,
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

func (h *SwaggerUIHandler) GetService() *DefaultService {
	return h.service
}

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
	"net/url"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// helper to create a handler with a registered service
func newTestHandler() *RequestHandler {
	cfg := global.DefaultOpenAPIConfig()
	cfg.Enabled = true
	svc := NewDefaultService(cfg)

	info := &common.ServiceInfo{
		InterfaceName: "TestService",
		Methods: []common.MethodInfo{
			{
				Name:        "Greet",
				ReqInitFunc: func() any { return &TestReq{} },
			},
		},
	}
	svc.RegisterService("TestService", info, "", "", "")

	return NewRequestHandler(svc, cfg)
}

// --- Handle tests ---

func TestRequestHandler_Handle_ApiDocsPath(t *testing.T) {
	h := newTestHandler()

	tests := []struct {
		name        string
		path        string
		wantOK      bool
		wantContent string
	}{
		{"api-docs root", "/dubbo/openapi/api-docs", true, constant.OpenAPIContentTypeJSON},
		{"api-docs with trailing slash", "/dubbo/openapi/api-docs/", true, constant.OpenAPIContentTypeJSON},
		{"api-docs group.json", "/dubbo/openapi/api-docs/default.json", true, constant.OpenAPIContentTypeJSON},
		{"api-docs group.yaml", "/dubbo/openapi/api-docs/default.yaml", true, constant.OpenAPIContentTypeYAML},
		{"api-docs group.yml", "/dubbo/openapi/api-docs/default.yml", true, constant.OpenAPIContentTypeYAML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{URL: &url.URL{Path: tt.path}}
			_, contentType, ok := h.Handle(req)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && contentType != tt.wantContent {
				t.Errorf("contentType = %q, want %q", contentType, tt.wantContent)
			}
		})
	}
}

func TestRequestHandler_Handle_OpenAPIPath(t *testing.T) {
	h := newTestHandler()

	tests := []struct {
		name        string
		path        string
		wantOK      bool
		wantContent string
	}{
		{"openapi base", "/dubbo/openapi", true, constant.OpenAPIContentTypeJSON},
		{"openapi base trailing slash", "/dubbo/openapi/", true, constant.OpenAPIContentTypeJSON},
		{"openapi.json", "/dubbo/openapi/openapi.json", true, constant.OpenAPIContentTypeJSON},
		{"openapi.yaml", "/dubbo/openapi/openapi.yaml", true, constant.OpenAPIContentTypeYAML},
		{"openapi.yml", "/dubbo/openapi/openapi.yml", true, constant.OpenAPIContentTypeYAML},
		{"group.json", "/dubbo/openapi/default.json", true, constant.OpenAPIContentTypeJSON},
		{"group.yaml", "/dubbo/openapi/default.yaml", true, constant.OpenAPIContentTypeYAML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{URL: &url.URL{Path: tt.path}}
			_, contentType, ok := h.Handle(req)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && contentType != tt.wantContent {
				t.Errorf("contentType = %q, want %q", contentType, tt.wantContent)
			}
		})
	}
}

func TestRequestHandler_Handle_UnmatchedPath(t *testing.T) {
	h := newTestHandler()

	req := &http.Request{URL: &url.URL{Path: "/other/path"}}
	_, _, ok := h.Handle(req)
	if ok {
		t.Error("unmatched path should return ok=false")
	}
}

func TestRequestHandler_Handle_CompletelyUnrelatedPath(t *testing.T) {
	h := newTestHandler()

	req := &http.Request{URL: &url.URL{Path: "/healthz"}}
	_, _, ok := h.Handle(req)
	if ok {
		t.Error("completely unrelated path should return ok=false")
	}
}

// --- handleAPIDocs tests ---

func TestRequestHandler_handleAPIDocs_GroupExtraction(t *testing.T) {
	h := newTestHandler()

	tests := []struct {
		name string
		path string
		// We check that the result is ok=true (meaning group was found and content generated)
		wantOK bool
	}{
		{"default group", "/dubbo/openapi/api-docs", true},
		{"default group with slash", "/dubbo/openapi/api-docs/", true},
		{"named group json", "/dubbo/openapi/api-docs/mygroup.json", true},
		{"named group yaml", "/dubbo/openapi/api-docs/mygroup.yaml", true},
		{"named group yml", "/dubbo/openapi/api-docs/mygroup.yml", true},
		{"named group no extension", "/dubbo/openapi/api-docs/mygroup", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := h.handleAPIDocs(tt.path, "/dubbo/openapi")
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

// --- handleOpenAPI tests ---

func TestRequestHandler_handleOpenAPI_GroupExtraction(t *testing.T) {
	h := newTestHandler()

	tests := []struct {
		name   string
		path   string
		wantOK bool
	}{
		{"base path", "/dubbo/openapi", true},
		{"base with slash", "/dubbo/openapi/", true},
		{"openapi.json", "/dubbo/openapi/openapi.json", true},
		{"openapi.yaml", "/dubbo/openapi/openapi.yaml", true},
		{"openapi.yml", "/dubbo/openapi/openapi.yml", true},
		{"group.json", "/dubbo/openapi/v2.json", true},
		{"group.yaml", "/dubbo/openapi/v2.yaml", true},
		{"group no extension", "/dubbo/openapi/v2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := h.handleOpenAPI(tt.path, "/dubbo/openapi")
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

// --- getOpenAPIContent tests ---

func TestRequestHandler_getOpenAPIContent_JSON(t *testing.T) {
	h := newTestHandler()

	content, contentType, ok := h.getOpenAPIContent(constant.OpenAPIDefaultGroup, "json")
	if !ok {
		t.Error("should return ok=true for JSON format")
	}
	if contentType != constant.OpenAPIContentTypeJSON {
		t.Errorf("contentType = %q, want %q", contentType, constant.OpenAPIContentTypeJSON)
	}
	if content == "" {
		t.Error("content should not be empty")
	}
}

func TestRequestHandler_getOpenAPIContent_YAML(t *testing.T) {
	h := newTestHandler()

	content, contentType, ok := h.getOpenAPIContent(constant.OpenAPIDefaultGroup, "yaml")
	if !ok {
		t.Error("should return ok=true for YAML format")
	}
	if contentType != constant.OpenAPIContentTypeYAML {
		t.Errorf("contentType = %q, want %q", contentType, constant.OpenAPIContentTypeYAML)
	}
	if content == "" {
		t.Error("content should not be empty")
	}
}

// --- GetService / GetConfig tests ---

func TestRequestHandler_GetService(t *testing.T) {
	h := newTestHandler()
	if h.GetService() == nil {
		t.Error("GetService should not return nil")
	}
}

func TestRequestHandler_GetConfig(t *testing.T) {
	h := newTestHandler()
	if h.GetConfig() == nil {
		t.Error("GetConfig should not return nil")
	}
}

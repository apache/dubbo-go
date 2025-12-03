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

package rest

import (
    "testing"

    restconfig "dubbo.apache.org/dubbo-go/v3/protocol/rest/config"
)

func TestNewServiceConfigAndApplyProvider(t *testing.T) {
    // Build service config via options
    cfg := NewServiceConfig(
        WithServiceInterface("com.example.Foo"),
        WithServicePath("/foo"),
        WithServiceMethod("Get",
            WithMethodPath("/get"),
            WithMethodHTTPMethod("GET"),
        ),
    )

    // Register as provider
    ApplyProviderServiceConfig("svc1", cfg)

    got := restconfig.GetRestProviderServiceConfig("svc1")
    if got == nil {
        t.Fatalf("expected provider config registered, got nil")
    }

    if len(got.RestMethodConfigs) != 1 {
        t.Fatalf("expected 1 method config, got %d", len(got.RestMethodConfigs))
    }

    m := got.RestMethodConfigs[0]
    if m.Path != "/foo/get" {
        t.Fatalf("unexpected joined path: want '/foo/get', got '%s'", m.Path)
    }
    if m.MethodType != "GET" {
        t.Fatalf("unexpected method type: want 'GET', got '%s'", m.MethodType)
    }
}

func TestNewServiceConfigAndApplyConsumer(t *testing.T) {
    cfg := NewServiceConfig(
        WithServiceInterface("com.example.Bar"),
        WithServicePath("/bar/"),
        WithServiceMethod("List",
            WithMethodPath("items"),
            WithMethodHTTPMethod("POST"),
        ),
    )

    ApplyConsumerServiceConfig("svc2", cfg)

    got := restconfig.GetRestConsumerServiceConfig("svc2")
    if got == nil {
        t.Fatalf("expected consumer config registered, got nil")
    }
    if len(got.RestMethodConfigs) != 1 {
        t.Fatalf("expected 1 method config, got %d", len(got.RestMethodConfigs))
    }
    m := got.RestMethodConfigs[0]
    if m.Path != "/bar/items" {
        t.Fatalf("unexpected joined path: want '/bar/items', got '%s'", m.Path)
    }
    if m.MethodType != "POST" {
        t.Fatalf("unexpected method type: want 'POST', got '%s'", m.MethodType)
    }
}

func TestDuplicateMethodOverwrite(t *testing.T) {
    cfg := NewServiceConfig(
        WithServiceInterface("com.example.Dup"),
        WithServicePath("/dup"),
        WithServiceMethod("Dup",
            WithMethodPath("one"),
            WithMethodHTTPMethod("GET"),
        ),
        WithServiceMethod("Dup",
            WithMethodPath("two"),
            WithMethodHTTPMethod("POST"),
        ),
    )

    ApplyProviderServiceConfig("svc_dup", cfg)
    got := restconfig.GetRestProviderServiceConfig("svc_dup")
    if got == nil {
        t.Fatalf("expected provider config registered, got nil")
    }
    // map should contain the last definition
    mm, ok := got.RestMethodConfigsMap["Dup"]
    if !ok {
        t.Fatalf("expected method map to contain 'Dup'")
    }
    if mm.Path != "/dup/two" {
        t.Fatalf("expected path '/dup/two' from last definition, got '%s'", mm.Path)
    }
    if mm.MethodType != "POST" {
        t.Fatalf("expected method type 'POST' from last definition, got '%s'", mm.MethodType)
    }
}

func TestMethodURLAndPathBehavior(t *testing.T) {
    cfg := NewServiceConfig(
        WithServiceInterface("com.example.URL"),
        WithServicePath("/base"),
        WithServiceMethod("Abs",
            WithMethodURL("http://example.com/abs"),
            WithMethodPath("rel"),
            WithMethodHTTPMethod("PUT"),
        ),
    )

    ApplyConsumerServiceConfig("svc_url", cfg)
    got := restconfig.GetRestConsumerServiceConfig("svc_url")
    if got == nil {
        t.Fatalf("expected consumer config registered, got nil")
    }
    m := got.RestMethodConfigsMap["Abs"]
    if m == nil {
        t.Fatalf("expected method config 'Abs' present")
    }
    // URL should be preserved and Path should be joined as well (current behavior)
    if m.URL != "http://example.com/abs" {
        t.Fatalf("expected URL preserved, got '%s'", m.URL)
    }
    if m.Path != "/base/rel" {
        t.Fatalf("expected joined path '/base/rel', got '%s'", m.Path)
    }
}

func TestParamMappings(t *testing.T) {
    cfg := NewServiceConfig(
        WithServiceInterface("com.example.Params"),
        WithServicePath("/p"),
        WithServiceMethod("Map",
            WithMethodPath("/m/{id}"),
            WithMethodPathParam(0, "id"),
            WithMethodQueryParam(1, "age"),
            WithMethodHeader(2, "X-Trace"),
        ),
    )

    ApplyProviderServiceConfig("svc_params", cfg)
    got := restconfig.GetRestProviderServiceConfig("svc_params")
    if got == nil {
        t.Fatalf("expected provider config registered, got nil")
    }
    m := got.RestMethodConfigsMap["Map"]
    if m == nil {
        t.Fatalf("expected method config 'Map' present")
    }
    if v, ok := m.PathParamsMap[0]; !ok || v != "id" {
        t.Fatalf("expected path param mapping 0->id, got '%v'", m.PathParamsMap)
    }
    if v, ok := m.QueryParamsMap[1]; !ok || v != "age" {
        t.Fatalf("expected query param mapping 1->age, got '%v'", m.QueryParamsMap)
    }
    if v, ok := m.HeadersMap[2]; !ok || v != "X-Trace" {
        t.Fatalf("expected header mapping 2->X-Trace, got '%v'", m.HeadersMap)
    }
}


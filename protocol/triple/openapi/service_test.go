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
	"reflect"
	"sync"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

// --- test types ---
type TestReq struct {
	Name string `json:"name"`
}

type TestResp struct {
	Msg string `json:"msg"`
}

// helper to create a basic ServiceInfo
func newTestServiceInfo(methods []common.MethodInfo) *common.ServiceInfo {
	return &common.ServiceInfo{
		InterfaceName: "TestService",
		Methods:       methods,
	}
}

// --- NewDefaultService tests ---

func TestNewDefaultService_NilConfig(t *testing.T) {
	svc := NewDefaultService(nil)
	if svc == nil {
		t.Fatal("NewDefaultService(nil) should not return nil")
	}
	if svc.config == nil {
		t.Error("config should be set to default when nil is passed")
	}
}

func TestNewDefaultService_CustomConfig(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	cfg.InfoTitle = "Custom API"
	svc := NewDefaultService(cfg)
	if svc.config.InfoTitle != "Custom API" {
		t.Errorf("InfoTitle = %q, want %q", svc.config.InfoTitle, "Custom API")
	}
}

// --- RegisterService tests ---

func TestDefaultService_RegisterService_DefaultGroup(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Test",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	svc.RegisterService("TestService", info, "", "", "")
	// openAPIs cache should be cleared
	if svc.openAPIs != nil {
		t.Error("openAPIs cache should be nil after RegisterService")
	}

	// Service should be registered with default group
	key := serviceKey{
		interfaceName: "TestService",
		group:         constant.OpenAPIDefaultGroup,
		dubboGroup:    "",
		dubboVersion:  "",
	}
	if _, ok := svc.services[key]; !ok {
		t.Error("service should be registered with default group")
	}
}

func TestDefaultService_RegisterService_CustomGroup(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo(nil)

	svc.RegisterService("TestService", info, "v2", "dubboGroup", "1.0.0")

	key := serviceKey{
		interfaceName: "TestService",
		group:         "v2",
		dubboGroup:    "dubboGroup",
		dubboVersion:  "1.0.0",
	}
	if _, ok := svc.services[key]; !ok {
		t.Error("service should be registered with custom group")
	}
}

func TestDefaultService_RegisterService_SameInterfaceDifferentGroups(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo(nil)

	svc.RegisterService("TestService", info, "v1", "", "")
	svc.RegisterService("TestService", info, "v2", "", "")

	if len(svc.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(svc.services))
	}
}

// --- GetOpenAPI tests ---

func TestDefaultService_GetOpenAPI_NilRequest(t *testing.T) {
	svc := NewDefaultService(nil)
	openAPI := svc.GetOpenAPI(nil)
	if openAPI == nil {
		t.Fatal("GetOpenAPI(nil) should not return nil")
	}
}

func TestDefaultService_GetOpenAPI_DefaultGroup(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	svc.RegisterService("TestService", info, "", "", "")
	openAPI := svc.GetOpenAPI(&OpenAPIRequest{Group: constant.OpenAPIDefaultGroup})

	if openAPI == nil {
		t.Fatal("GetOpenAPI should not return nil")
	}
}

func TestDefaultService_GetOpenAPI_NonExistentGroup_ReturnsEmpty(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	cfg.InfoTitle = "Test API"
	svc := NewDefaultService(cfg)

	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	svc.RegisterService("Svc1", info, "group1", "", "")
	svc.RegisterService("Svc2", info, "group2", "", "")

	// Requesting a non-existent group should return an empty OpenAPI document
	openAPI := svc.GetOpenAPI(&OpenAPIRequest{Group: "nonexistent"})
	if openAPI == nil {
		t.Fatal("GetOpenAPI with non-existent group should not return nil")
	}
	if len(openAPI.Paths) != 0 {
		t.Error("non-existent group should return empty paths")
	}
	if openAPI.Info == nil || openAPI.Info.Title != "Test API" {
		t.Error("non-existent group should still have configured info title")
	}
}

func TestDefaultService_GetOpenAPI_CachedResult(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	svc.RegisterService("TestService", info, "", "", "")

	// First call resolves
	openAPI1 := svc.GetOpenAPI(&OpenAPIRequest{Group: constant.OpenAPIDefaultGroup})
	// Second call should use cache
	openAPI2 := svc.GetOpenAPI(&OpenAPIRequest{Group: constant.OpenAPIDefaultGroup})

	if openAPI1 != openAPI2 {
		t.Error("GetOpenAPI should return cached result on second call")
	}
}

// --- GetOpenAPIGroups tests ---

func TestDefaultService_GetOpenAPIGroups(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	svc.RegisterService("Svc1", info, "", "", "")
	svc.RegisterService("Svc2", info, "v2", "", "")
	svc.RegisterService("Svc3", info, "v3", "", "")

	groups := svc.GetOpenAPIGroups()

	// Should always include "default"
	found := false
	for _, g := range groups {
		if g == constant.OpenAPIDefaultGroup {
			found = true
			break
		}
	}
	if !found {
		t.Error("groups should include default group")
	}

	// Should include v2 and v3
	if len(groups) < 3 {
		t.Errorf("expected at least 3 groups, got %d", len(groups))
	}
}

func TestDefaultService_GetOpenAPIGroups_NoDuplicates(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo(nil)

	svc.RegisterService("Svc1", info, "", "", "")
	svc.RegisterService("Svc2", info, "", "", "")

	groups := svc.GetOpenAPIGroups()
	seen := map[string]bool{}
	for _, g := range groups {
		if seen[g] {
			t.Errorf("duplicate group %q found", g)
		}
		seen[g] = true
	}
}

// --- Refresh tests ---

func TestDefaultService_Refresh(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	svc.RegisterService("TestService", info, "", "", "")

	// Trigger caching
	_ = svc.GetOpenAPI(&OpenAPIRequest{Group: constant.OpenAPIDefaultGroup})
	if svc.openAPIs == nil {
		t.Error("openAPIs should be cached after GetOpenAPI")
	}

	// Refresh should clear cache
	svc.Refresh()
	if svc.openAPIs != nil {
		t.Error("openAPIs should be nil after Refresh")
	}
}

// --- mergeOpenAPI tests ---

func TestDefaultService_mergeOpenAPI_Paths(t *testing.T) {
	svc := NewDefaultService(nil)

	target := model.NewOpenAPI()
	source := model.NewOpenAPI()

	source.Paths = map[string]*model.PathItem{
		"/test": model.NewPathItem().SetOperation("POST", model.NewOperation().SetOperationId("test")),
	}

	svc.mergeOpenAPI(target, source)

	if len(target.Paths) != 1 {
		t.Errorf("expected 1 path, got %d", len(target.Paths))
	}
	if _, ok := target.Paths["/test"]; !ok {
		t.Error("path /test should be merged")
	}
}

func TestDefaultService_mergeOpenAPI_Components(t *testing.T) {
	svc := NewDefaultService(nil)

	target := model.NewOpenAPI()
	source := model.NewOpenAPI()

	source.Components = model.NewComponents()
	source.Components.AddSchema("Req", model.NewSchema().SetType(model.SchemaTypeObject))

	svc.mergeOpenAPI(target, source)

	if target.Components == nil {
		t.Fatal("Components should not be nil after merge")
	}
	if _, ok := target.Components.Schemas["Req"]; !ok {
		t.Error("schema Req should be merged")
	}
}

func TestDefaultService_mergeOpenAPI_NilSource(t *testing.T) {
	svc := NewDefaultService(nil)

	target := model.NewOpenAPI()
	source := model.NewOpenAPI()
	// source has no paths, no components

	svc.mergeOpenAPI(target, source)

	if len(target.Paths) != 0 {
		t.Error("target paths should be empty")
	}
}

// --- snapshotServiceInfo tests ---

func TestSnapshotServiceInfo_Nil(t *testing.T) {
	result := snapshotServiceInfo(nil)
	if result != nil {
		t.Error("snapshotServiceInfo(nil) should return nil")
	}
}

func TestSnapshotServiceInfo_DeepCopy(t *testing.T) {
	original := &common.ServiceInfo{
		InterfaceName: "TestService",
		Methods: []common.MethodInfo{
			{
				Name:        "Greet",
				ReqInitFunc: func() any { return &TestReq{} },
				Meta:        map[string]any{"key": "value"},
			},
		},
	}

	snapshot := snapshotServiceInfo(original)

	if snapshot == nil {
		t.Fatal("snapshot should not be nil")
	}
	if len(snapshot.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(snapshot.Methods))
	}
	if snapshot.Methods[0].Name != "Greet" {
		t.Errorf("method name = %q, want %q", snapshot.Methods[0].Name, "Greet")
	}

	// Mutating original should not affect snapshot
	original.Methods[0].Name = "Changed"
	if snapshot.Methods[0].Name != "Greet" {
		t.Error("snapshot should be independent of original")
	}

	// Mutating meta should not affect snapshot
	original.Methods[0].Meta["key"] = "changed"
	if snapshot.Methods[0].Meta["key"] != "value" {
		t.Error("snapshot meta should be independent of original")
	}
}

func TestSnapshotServiceInfo_NoMethods(t *testing.T) {
	original := &common.ServiceInfo{
		InterfaceName: "TestService",
		Methods:       nil,
	}

	snapshot := snapshotServiceInfo(original)
	if snapshot == nil {
		t.Fatal("snapshot should not be nil")
	}
	if len(snapshot.Methods) != 0 {
		t.Errorf("expected 0 methods, got %d", len(snapshot.Methods))
	}
}

// --- concurrency test ---

func TestDefaultService_ConcurrentAccess(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			svc.RegisterService("Svc", info, "", "", "")
			_ = svc.GetOpenAPI(&OpenAPIRequest{Group: constant.OpenAPIDefaultGroup})
			_ = svc.GetOpenAPIGroups()
		}(i)
	}
	wg.Wait()
}

// --- GetEncoder / GetConfig tests ---

func TestDefaultService_GetEncoder(t *testing.T) {
	svc := NewDefaultService(nil)
	if svc.GetEncoder() == nil {
		t.Error("GetEncoder should not return nil")
	}
}

func TestDefaultService_GetConfig(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	svc := NewDefaultService(cfg)
	if svc.GetConfig() != cfg {
		t.Error("GetConfig should return the original config")
	}
}

// --- resolveOpenAPIs tests ---

func TestDefaultService_resolveOpenAPIs_MultipleGroups(t *testing.T) {
	svc := NewDefaultService(nil)
	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
			Meta:        map[string]any{"response.type": reflect.TypeOf(TestResp{})},
		},
	})

	svc.RegisterService("Svc1", info, "group1", "", "")
	svc.RegisterService("Svc2", info, "group2", "", "")

	openAPIs := svc.resolveOpenAPIs()

	if len(openAPIs) != 2 {
		t.Errorf("expected 2 groups, got %d", len(openAPIs))
	}
	if _, ok := openAPIs["group1"]; !ok {
		t.Error("group1 should exist")
	}
	if _, ok := openAPIs["group2"]; !ok {
		t.Error("group2 should exist")
	}
}

// --- mergeOpenAPIs tests ---

func TestDefaultService_mergeOpenAPIs(t *testing.T) {
	cfg := global.DefaultOpenAPIConfig()
	cfg.InfoTitle = "Merged API"
	cfg.InfoVersion = "2.0.0"
	cfg.InfoDescription = "All groups merged"
	svc := NewDefaultService(cfg)

	info := newTestServiceInfo([]common.MethodInfo{
		{
			Name:        "Greet",
			ReqInitFunc: func() any { return &TestReq{} },
		},
	})

	svc.RegisterService("Svc1", info, "group1", "", "")
	svc.RegisterService("Svc2", info, "group2", "", "")

	openAPIs := svc.getOpenAPIs()
	merged := svc.mergeOpenAPIs(openAPIs)

	if merged.Info.Title != "Merged API" {
		t.Errorf("merged title = %q, want %q", merged.Info.Title, "Merged API")
	}
	if merged.Info.Version != "2.0.0" {
		t.Errorf("merged version = %q, want %q", merged.Info.Version, "2.0.0")
	}
	if len(merged.Paths) == 0 {
		t.Error("merged OpenAPI should contain paths from all groups")
	}
}

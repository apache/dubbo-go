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

package polaris

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

import (
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/polarismesh/polaris-go/pkg/plugin"
	v1 "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// --- mock types for Route() tests ---

// mockInvoker implements base.Invoker with a configurable URL.
type mockInvoker struct {
	url *common.URL
}

func (m *mockInvoker) GetURL() *common.URL         { return m.url }
func (m *mockInvoker) IsAvailable() bool            { return true }
func (m *mockInvoker) Destroy()                     {}
func (m *mockInvoker) Invoke(_ context.Context, _ base.Invocation) result.Result {
	return nil
}

// mockInvocation implements base.Invocation with minimal stubs.
type mockInvocation struct{}

func (m *mockInvocation) MethodName() string                                     { return "TestMethod" }
func (m *mockInvocation) ActualMethodName() string                               { return "TestMethod" }
func (m *mockInvocation) ParameterTypeNames() []string                           { return nil }
func (m *mockInvocation) ParameterTypes() []reflect.Type                         { return nil }
func (m *mockInvocation) ParameterValues() []reflect.Value                       { return nil }
func (m *mockInvocation) ParameterRawValues() []any                              { return nil }
func (m *mockInvocation) Arguments() []any                                       { return nil }
func (m *mockInvocation) Reply() any                                             { return nil }
func (m *mockInvocation) Invoker() base.Invoker                                  { return nil }
func (m *mockInvocation) IsGenericInvocation() bool                              { return false }
func (m *mockInvocation) Attachments() map[string]any                            { return map[string]any{} }
func (m *mockInvocation) SetAttachment(_ string, _ any)                          {}
func (m *mockInvocation) GetAttachment(_ string) (string, bool)                  { return "", false }
func (m *mockInvocation) GetAttachmentInterface(_ string) any                    { return nil }
func (m *mockInvocation) GetAttachmentWithDefaultValue(_ string, d string) string { return d }
func (m *mockInvocation) GetAttachmentAsContext() context.Context                { return context.Background() }
func (m *mockInvocation) MergeAttachmentFromContext(_ context.Context)            {}
func (m *mockInvocation) Attributes() map[string]any                             { return nil }
func (m *mockInvocation) SetAttribute(_ string, _ any)                           {}
func (m *mockInvocation) GetAttribute(_ string) (any, bool)                      { return nil, false }
func (m *mockInvocation) GetAttributeWithDefaultValue(_ string, d any) any       { return d }

// mockInstance implements model.Instance with a configurable ID.
type mockInstance struct {
	id string
}

func (m *mockInstance) GetInstanceKey() model.InstanceKey { return model.InstanceKey{} }
func (m *mockInstance) GetId() string                     { return m.id }
func (m *mockInstance) GetHost() string                   { return "127.0.0.1" }
func (m *mockInstance) GetPort() uint32                   { return 20000 }
func (m *mockInstance) GetNamespace() string              { return "default" }
func (m *mockInstance) GetService() string                { return "test-service" }
func (m *mockInstance) GetVpcId() string                  { return "" }
func (m *mockInstance) GetProtocol() string               { return "dubbo" }
func (m *mockInstance) GetVersion() string                { return "1.0.0" }
func (m *mockInstance) GetWeight() int                    { return 100 }
func (m *mockInstance) GetPriority() uint32               { return 0 }
func (m *mockInstance) GetMetadata() map[string]string    { return nil }
func (m *mockInstance) GetLogicSet() string               { return "" }
func (m *mockInstance) GetCircuitBreakerStatus() model.CircuitBreakerStatus { return nil }
func (m *mockInstance) IsHealthy() bool                   { return true }
func (m *mockInstance) IsIsolated() bool                  { return false }
func (m *mockInstance) IsEnableHealthCheck() bool         { return false }
func (m *mockInstance) GetRegion() string                 { return "" }
func (m *mockInstance) GetZone() string                   { return "" }
func (m *mockInstance) GetIDC() string                    { return "" }
func (m *mockInstance) GetCampus() string                 { return "" }
func (m *mockInstance) GetRevision() string               { return "" }

// mockEngine implements model.Engine. Only SyncGetServiceRule is used by Route().
type mockEngine struct {
	model.Engine
	routeRule *model.ServiceRuleResponse
	ruleErr   error
}

func (m *mockEngine) SyncGetServiceRule(_ model.EventType, _ *model.GetServiceRuleRequest) (*model.ServiceRuleResponse, error) {
	return m.routeRule, m.ruleErr
}

// mockSDKContext implements api.SDKContext, returning mockEngine.
type mockSDKContext struct {
	api.SDKContext
	engine model.Engine
}

func (m *mockSDKContext) GetEngine() model.Engine                    { return m.engine }
func (m *mockSDKContext) Destroy()                                   {}
func (m *mockSDKContext) IsDestroyed() bool                         { return false }
func (m *mockSDKContext) GetConfig() config.Configuration           { return nil }
func (m *mockSDKContext) GetPlugins() plugin.Manager                { return nil }
func (m *mockSDKContext) GetValueContext() model.ValueContext        { return nil }

// mockRouterAPI implements polaris.RouterAPI.
type mockRouterAPI struct {
	polaris.RouterAPI
	sdkCtx         api.SDKContext
	processResp    *model.InstancesResponse
	processErr     error
}

func (m *mockRouterAPI) SDKContext() api.SDKContext { return m.sdkCtx }
func (m *mockRouterAPI) ProcessRouters(_ *polaris.ProcessRoutersRequest) (*model.InstancesResponse, error) {
	return m.processResp, m.processErr
}

// mockConsumerAPI implements polaris.ConsumerAPI.
type mockConsumerAPI struct {
	polaris.ConsumerAPI
	allInstancesResp *model.InstancesResponse
	allInstancesErr  error
}

func (m *mockConsumerAPI) SDKContext() api.SDKContext { return nil }
func (m *mockConsumerAPI) GetAllInstances(_ *polaris.GetAllInstancesRequest) (*model.InstancesResponse, error) {
	return m.allInstancesResp, m.allInstancesErr
}

// helper: build a mock invoker with a polaris instance ID in its URL.
func newMockInvokerWithInstanceID(t *testing.T, instanceID string) *mockInvoker {
	t.Helper()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.xxx.Service?" +
		constant.PolarisInstanceID + "=" + instanceID +
		"&interface=com.xxx.Service")
	require.NoError(t, err)
	return &mockInvoker{url: url}
}

// helper: build a polarisRouter with mock APIs for Route() testing.
func newTestRouter(t *testing.T, routerAPI *mockRouterAPI, consumerAPI *mockConsumerAPI) *polarisRouter {
	t.Helper()
	return &polarisRouter{
		openRoute:          true,
		routerAPI:          routerAPI,
		consumerAPI:        consumerAPI,
		currentApplication: "test-app",
		Registries:         map[string]*global.RegistryConfig{},
	}
}

const (
	baseServiceURL    = "dubbo://127.0.0.1:20000/com.xxx.Service"
	baseSubServiceURL = "dubbo://127.0.0.1:20001/com.xxx.Service"
	serviceURLWithApp = baseServiceURL + "?application=param-app"
	errMissingAppName = "polaris router must set application name"
	attrAppName       = "attr-app"
	subAttrAppName    = "sub-attr-app"
)

func mustNewURL(t *testing.T, rawURL string) *common.URL {
	t.Helper()
	url, err := common.NewURL(rawURL)
	require.NoError(t, err)
	return url
}

func TestNewPolarisRouterApplicationNameFromParam(t *testing.T) {
	url := mustNewURL(t, serviceURLWithApp)

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, "param-app", r.currentApplication)
}

func TestNewPolarisRouterApplicationNameFromAttribute(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)
	url.SetAttribute(constant.ApplicationKey, &global.ApplicationConfig{Name: attrAppName})

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, attrAppName, r.currentApplication)
}

func TestNewPolarisRouterApplicationNameFromSubURLAttribute(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)
	subURL := mustNewURL(t, baseSubServiceURL)
	subURL.SetAttribute(constant.ApplicationKey, &global.ApplicationConfig{Name: subAttrAppName})
	url.SubURL = subURL

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, subAttrAppName, r.currentApplication)
}

func TestNewPolarisRouterApplicationNameMissing(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)

	_, err := newPolarisRouter(url)
	require.EqualError(t, err, errMissingAppName)
}

func TestNewPolarisRouterApplicationNameFromSubURLParam(t *testing.T) {
	url := mustNewURL(t, baseServiceURL)
	subURL := mustNewURL(t, serviceURLWithApp)
	url.SubURL = subURL

	r, err := newPolarisRouter(url)
	require.NoError(t, err)
	require.Equal(t, "param-app", r.currentApplication)
}

func TestNewPolarisRouterInvalidRegistriesTypeShouldFallback(t *testing.T) {
	url := mustNewURL(t, serviceURLWithApp)
	url.SetAttribute(constant.RegistriesConfigKey, map[string]global.RegistryConfig{
		"test": {},
	})

	var (
		r   *polarisRouter
		err error
	)
	require.NotPanics(t, func() {
		r, err = newPolarisRouter(url)
	})
	require.NoError(t, err)
	require.NotNil(t, r)
	require.NotNil(t, r.Registries)
	require.Empty(t, r.Registries)
}

// --- Route() regression tests for issue #3303 ---

func TestRouteOpenRouteDisabledReturnsOriginalInvokers(t *testing.T) {
	r := &polarisRouter{
		openRoute:  false,
		Registries: map[string]*global.RegistryConfig{},
	}
	url := mustNewURL(t, baseServiceURL+"?interface=com.xxx.Service")
	invokers := []base.Invoker{newMockInvokerWithInstanceID(t, "inst-1")}

	got := r.Route(invokers, url, &mockInvocation{})
	assert.Equal(t, invokers, got, "should return original invokers when openRoute is false")
}

func TestRouteEmptyInvokersReturnsEmpty(t *testing.T) {
	r := &polarisRouter{
		openRoute:  true,
		Registries: map[string]*global.RegistryConfig{},
	}
	url := mustNewURL(t, baseServiceURL+"?interface=com.xxx.Service")

	got := r.Route(nil, url, &mockInvocation{})
	assert.Empty(t, got, "should return empty when input invokers is nil")

	got = r.Route([]base.Invoker{}, url, &mockInvocation{})
	assert.Empty(t, got, "should return empty when input invokers is empty")
}

func TestRouteGetAllInstancesErrorReturnsOriginalInvokers(t *testing.T) {
	consumerAPI := &mockConsumerAPI{
		allInstancesResp: nil,
		allInstancesErr:  errors.New("polaris unavailable"),
	}
	routerAPI := &mockRouterAPI{}
	r := newTestRouter(t, routerAPI, consumerAPI)
	url := mustNewURL(t, baseServiceURL+"?interface=com.xxx.Service")
	invokers := []base.Invoker{newMockInvokerWithInstanceID(t, "inst-1")}

	got := r.Route(invokers, url, &mockInvocation{})
	assert.Equal(t, invokers, got, "should return original invokers when GetAllInstances fails")
}

func TestRouteProcessRoutersErrorReturnsOriginalInvokers(t *testing.T) {
	inst := &mockInstance{id: "inst-1"}
	consumerAPI := &mockConsumerAPI{
		allInstancesResp: &model.InstancesResponse{
			Instances: []model.Instance{inst},
		},
	}
	engine := &mockEngine{
		routeRule: &model.ServiceRuleResponse{
			Value: &v1.Routing{},
		},
	}
	sdkCtx := &mockSDKContext{engine: engine}
	routerAPI := &mockRouterAPI{
		sdkCtx:     sdkCtx,
		processErr: errors.New("route processing failed"),
	}
	r := newTestRouter(t, routerAPI, consumerAPI)
	url := mustNewURL(t, baseServiceURL+"?interface=com.xxx.Service")
	invokers := []base.Invoker{newMockInvokerWithInstanceID(t, "inst-1")}

	got := r.Route(invokers, url, &mockInvocation{})
	assert.Equal(t, invokers, got, "should return original invokers when ProcessRouters fails")
}

// TestRouteRuleNotMatchFallbackToOriginalInvokers is the key regression test for issue #3303.
// When ProcessRouters returns instances that don't match any known invoker (route rule not match),
// Route() should return the original invokers instead of an empty slice to prevent nil pointer panic.
func TestRouteRuleNotMatchFallbackToOriginalInvokers(t *testing.T) {
	inst := &mockInstance{id: "inst-1"}
	consumerAPI := &mockConsumerAPI{
		allInstancesResp: &model.InstancesResponse{
			Instances: []model.Instance{inst},
		},
	}
	engine := &mockEngine{
		routeRule: &model.ServiceRuleResponse{
			Value: &v1.Routing{},
		},
	}
	sdkCtx := &mockSDKContext{engine: engine}
	// ProcessRouters returns instances with IDs that don't exist in invokersMap
	nonMatchingInst := &mockInstance{id: "unknown-inst-999"}
	routerAPI := &mockRouterAPI{
		sdkCtx: sdkCtx,
		processResp: &model.InstancesResponse{
			Instances: []model.Instance{nonMatchingInst},
		},
	}
	r := newTestRouter(t, routerAPI, consumerAPI)
	url := mustNewURL(t, baseServiceURL+"?interface=com.xxx.Service")
	invokers := []base.Invoker{newMockInvokerWithInstanceID(t, "inst-1")}

	var got []base.Invoker
	require.NotPanics(t, func() {
		got = r.Route(invokers, url, &mockInvocation{})
	}, "should not panic when route rule not match")
	assert.Equal(t, invokers, got, "should fallback to original invokers when route rule returns non-matching instances")
	assert.Len(t, got, 1, "should preserve all original invokers")
}

// TestRouteEmptyProcessRoutersResultFallbackToOriginalInvokers tests the case where
// ProcessRouters returns an empty instance list (e.g., all instances filtered out).
func TestRouteEmptyProcessRoutersResultFallbackToOriginalInvokers(t *testing.T) {
	inst := &mockInstance{id: "inst-1"}
	consumerAPI := &mockConsumerAPI{
		allInstancesResp: &model.InstancesResponse{
			Instances: []model.Instance{inst},
		},
	}
	engine := &mockEngine{
		routeRule: &model.ServiceRuleResponse{
			Value: &v1.Routing{},
		},
	}
	sdkCtx := &mockSDKContext{engine: engine}
	// ProcessRouters returns empty instances
	routerAPI := &mockRouterAPI{
		sdkCtx: sdkCtx,
		processResp: &model.InstancesResponse{
			Instances: []model.Instance{},
		},
	}
	r := newTestRouter(t, routerAPI, consumerAPI)
	url := mustNewURL(t, baseServiceURL+"?interface=com.xxx.Service")
	invokers := []base.Invoker{newMockInvokerWithInstanceID(t, "inst-1")}

	var got []base.Invoker
	require.NotPanics(t, func() {
		got = r.Route(invokers, url, &mockInvocation{})
	}, "should not panic when ProcessRouters returns empty instances")
	assert.Equal(t, invokers, got, "should fallback to original invokers when ProcessRouters returns empty list")
}

// TestRouteHappyPathReturnsMatchedInvokers tests that Route() correctly filters invokers
// when ProcessRouters returns matching instances.
func TestRouteHappyPathReturnsMatchedInvokers(t *testing.T) {
	inst1 := &mockInstance{id: "inst-1"}
	inst2 := &mockInstance{id: "inst-2"}
	consumerAPI := &mockConsumerAPI{
		allInstancesResp: &model.InstancesResponse{
			Instances: []model.Instance{inst1, inst2},
		},
	}
	engine := &mockEngine{
		routeRule: &model.ServiceRuleResponse{
			Value: &v1.Routing{},
		},
	}
	sdkCtx := &mockSDKContext{engine: engine}
	// ProcessRouters returns only inst-1 (route rule matched)
	routerAPI := &mockRouterAPI{
		sdkCtx: sdkCtx,
		processResp: &model.InstancesResponse{
			Instances: []model.Instance{inst1},
		},
	}
	r := newTestRouter(t, routerAPI, consumerAPI)
	url := mustNewURL(t, baseServiceURL+"?interface=com.xxx.Service")
	invoker1 := newMockInvokerWithInstanceID(t, "inst-1")
	invoker2 := newMockInvokerWithInstanceID(t, "inst-2")
	invokers := []base.Invoker{invoker1, invoker2}

	got := r.Route(invokers, url, &mockInvocation{})
	assert.Len(t, got, 1, "should return only matched invokers")
	assert.Equal(t, invoker1, got[0], "should return the invoker matching inst-1")
}

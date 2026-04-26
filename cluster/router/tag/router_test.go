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

package tag

import (
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	common_cfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

var (
	url1        *common.URL
	url2        *common.URL
	url3        *common.URL
	consumerUrl *common.URL
)

func initUrl() {
	url1, _ = common.NewURL("dubbo://192.168.0.1:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0")
	url2, _ = common.NewURL("dubbo://192.168.0.2:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0")
	url3, _ = common.NewURL("dubbo://192.168.0.3:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0")
	consumerUrl, _ = common.NewURL("consumer://127.0.0.1:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0")
}

func TestRouter(t *testing.T) {
	initUrl()
	trueValue := true
	truePointer := &trueValue
	falseValue := false
	falsePointer := &falseValue
	t.Run("staticEmptyTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 3)
	})
	t.Run("staticEmptyTag_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 3)
	})
	t.Run("staticEmptyTag_requestHasTag_force", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag", constant.ForceUseTag: "true"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Empty(t, result)
	})
	t.Run("staticTag_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		ivk2.GetURL().SetParam(constant.Tagkey, "tag")
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 1)
		assert.Equal(t, "tag", result[0].GetURL().GetParam(constant.Tagkey, ""))
	})
	t.Run("staticTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		ivk2.GetURL().SetParam(constant.Tagkey, "tag")
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 2)
	})

	initUrl()
	t.Run("dynamicEmptyTag_requestEmptyTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   truePointer,
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 3)
	})

	t.Run("dynamicEmptyTag_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   truePointer,
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 3)
	})

	t.Run("dynamicTag_requestEmptyTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(strings.Join([]string{consumerUrl.GetParam(constant.ApplicationKey, ""), constant.TagRouterRuleSuffix}, ""), global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   truePointer,
			Tags: []global.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.3:20000"},
			}},
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 2)
	})

	t.Run("dynamicTag_emptyAddress_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   truePointer,
			Tags: []global.Tag{{
				Name: "tag",
			}},
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 3)
	})

	t.Run("dynamicTag_address_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(strings.Join([]string{consumerUrl.GetParam(constant.ApplicationKey, ""), constant.TagRouterRuleSuffix}, ""), global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   truePointer,
			Tags: []global.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.3:20000"},
			}},
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 1)
	})

	t.Run("dynamicTag_twoAddress_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(strings.Join([]string{consumerUrl.GetParam(constant.ApplicationKey, ""), constant.TagRouterRuleSuffix}, ""), global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   truePointer,
			Tags: []global.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.3:20000"},
			}},
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, url3, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 2)
	})

	t.Run("dynamicTag_addressNotMatch_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   truePointer,
			Tags: []global.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.4:20000"},
			}},
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 3)
	})

	t.Run("dynamicTag_notValid", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, global.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   falsePointer,
			Enabled: truePointer,
			Valid:   falsePointer,
			Tags: []global.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.3:20000"},
			}},
		})
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]any{constant.Tagkey: "tag"}
		result := p.Route(invokerList, url3, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 3)
	})

	t.Run("dynamicConfigIsNull", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, url3, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 3)
	})

	t.Run("dynamicTag_withApplicationKey", func(t *testing.T) { //NOSONAR
		p, _ := NewTagPriorityRouter()
		ivk1 := base.NewBaseInvoker(url1)
		ivk1.GetURL().SetParam(constant.ApplicationKey, "test-app") //NOSONAR
		ivk2 := base.NewBaseInvoker(url2)
		ivk2.GetURL().SetParam(constant.ApplicationKey, "test-app") //NOSONAR

		configKey := "test-app" + constant.TagRouterRuleSuffix //NOSONAR
		p.routerConfigs.Store(configKey, global.RouterConfig{
			Enabled: truePointer,
			Valid:   truePointer,
			Tags: []global.Tag{{
				Addresses: []string{"192.168.0.1:20000"}, //NOSONAR
			}},
		})

		result := p.Route([]base.Invoker{ivk1, ivk2}, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 1)
	})
}

func TestNotify(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		initUrl()
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		ivk.GetURL().SetParam(constant.ApplicationKey, "org.apache.dubbo.UserProvider.Test")
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		extension.SetDefaultConfigurator(configurator.NewMockConfigurator)
		ccUrl, _ := common.NewURL("mock://127.0.0.1:1111")
		mockFactory := &config_center.MockDynamicConfigurationFactory{
			Content: `
key: org.apache.dubbo.UserProvider.Test
force: false
enabled: true
runtime: false
tags:
 - name: tag1
   addresses: [192.168.0.1:20881]
 - name: tag2
   addresses: [192.168.0.2:20882]`,
		}
		dc, _ := mockFactory.GetDynamicConfiguration(ccUrl)
		common_cfg.GetEnvInstance().SetDynamicConfiguration(dc)
		p.Notify(invokerList)
		value, ok := p.routerConfigs.Load(url1.GetParam(constant.ApplicationKey, "") + constant.TagRouterRuleSuffix)
		assert.True(t, ok)
		routerCfg := value.(global.RouterConfig)
		assert.Equal(t, "org.apache.dubbo.UserProvider.Test", routerCfg.Key)
		assert.Len(t, routerCfg.Tags, 2)
		assert.True(t, *routerCfg.Enabled)
	})

	t.Run("configNotValid", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)
		initUrl()
		ivk := base.NewBaseInvoker(url1)
		ivk1 := base.NewBaseInvoker(url2)
		ivk2 := base.NewBaseInvoker(url3)
		ivk.GetURL().SetParam(constant.ApplicationKey, "org.apache.dubbo.UserProvider.Test")
		invokerList := make([]base.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		extension.SetDefaultConfigurator(configurator.NewMockConfigurator)
		ccUrl, _ := common.NewURL("mock://127.0.0.1:1111")
		mockFactory := &config_center.MockDynamicConfigurationFactory{
			Content: `xxxxxx`,
		}
		dc, _ := mockFactory.GetDynamicConfiguration(ccUrl)
		common_cfg.GetEnvInstance().SetDynamicConfiguration(dc)
		p.Notify(invokerList)
		value, ok := p.routerConfigs.Load(url1.GetParam(constant.ApplicationKey, "") + constant.TagRouterRuleSuffix)
		assert.False(t, ok)
		assert.Nil(t, value)
	})
}

func TestSetStaticConfig(t *testing.T) {
	t.Run("empty tag config is ignored", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)

		p.SetStaticConfig(&global.RouterConfig{
			Scope: constant.RouterScopeApplication,
			Key:   "test-app",
		})

		value, ok := p.routerConfigs.Load("test-app" + constant.TagRouterRuleSuffix)
		assert.False(t, ok)
		assert.Nil(t, value)
	})

	t.Run("service-scope tag config is ignored", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)

		p.SetStaticConfig(&global.RouterConfig{
			Scope: constant.RouterScopeService,
			Key:   "svc.test",
			Tags: []global.Tag{{
				Name: "gray",
			}},
		})

		value, ok := p.routerConfigs.Load("svc.test" + constant.TagRouterRuleSuffix)
		assert.False(t, ok)
		assert.Nil(t, value)
	})

	t.Run("static tag config is cloned and stored with default enabled", func(t *testing.T) { // NOSONAR
		p, err := NewTagPriorityRouter()
		require.NoError(t, err)

		cfg := &global.RouterConfig{
			Scope: constant.RouterScopeApplication,
			Key:   "test-app",
			Tags: []global.Tag{{
				Name:      "gray",
				Addresses: []string{"192.168.0.1:20000"}, // NOSONAR
			}},
		}

		p.SetStaticConfig(cfg)
		cfg.Tags[0].Addresses[0] = "192.168.0.9:20000" // NOSONAR

		value, ok := p.routerConfigs.Load("test-app" + constant.TagRouterRuleSuffix)
		require.True(t, ok)
		routerCfg := value.(global.RouterConfig)
		assert.True(t, *routerCfg.Enabled)
		assert.True(t, *routerCfg.Valid)
		assert.Equal(t, "192.168.0.1:20000", routerCfg.Tags[0].Addresses[0]) // NOSONAR
	})
}

func TestParseRoute(t *testing.T) {
	t.Run("route with tags is valid", func(t *testing.T) {
		cfg, err := parseRoute(`
key: test-app
tags:
 - name: gray
   addresses: [192.168.0.1:20000]
`)
		require.NoError(t, err)
		require.NotNil(t, cfg.Valid)
		assert.True(t, *cfg.Valid)
	})

	t.Run("route without tags is invalid", func(t *testing.T) {
		cfg, err := parseRoute("key: test-app")
		require.NoError(t, err)
		require.NotNil(t, cfg.Valid)
		assert.False(t, *cfg.Valid)
	})
}

func TestRequestTagNilForceFallback(t *testing.T) {
	initUrl()

	ivk := base.NewBaseInvoker(url1)
	ivk1 := base.NewBaseInvoker(url2)
	ivk2 := base.NewBaseInvoker(url3)
	invokerList := []base.Invoker{ivk, ivk1, ivk2}

	result := requestTag(
		invokerList,
		consumerUrl,
		invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"}),
		global.RouterConfig{
			Tags: []global.Tag{{
				Name: "gray",
			}},
		},
		"gray",
	)

	assert.Len(t, result, 3)
}

func TestRouteNilDefaults(t *testing.T) {
	initUrl()

	p, err := NewTagPriorityRouter()
	require.NoError(t, err)

	ivk := base.NewBaseInvoker(url1)
	ivk1 := base.NewBaseInvoker(url2)
	ivk2 := base.NewBaseInvoker(url3)
	invokerList := []base.Invoker{ivk, ivk1, ivk2}

	p.routerConfigs.Store(constant.TagRouterRuleSuffix, global.RouterConfig{
		Tags: []global.Tag{{
			Name: "gray",
		}},
	})

	result := p.Route(
		invokerList,
		consumerUrl,
		invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"}),
	)

	assert.Len(t, result, 3)
}

type mockCache struct {
	invokers []base.Invoker
	pool     router.AddrPool
}

func (m *mockCache) GetInvokers() []base.Invoker                        { return m.invokers }
func (m *mockCache) FindAddrPool(_ router.Poolable) router.AddrPool     { return m.pool }
func (m *mockCache) FindAddrMeta(_ router.Poolable) router.AddrMetadata { return nil }
func (m *mockCache) FindAddrPoolWithInvokers(_ router.Poolable) ([]base.Invoker, router.AddrPool) {
	return m.invokers, m.pool
}

func newCacheRouter() *PriorityRouter {
	initUrl()
	p, _ := NewTagPriorityRouter()
	return p
}

func makeInvokers(tag1, tag2, tag3 string) []base.Invoker {
	u1, _ := common.NewURL("dubbo://192.168.0.1:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0")
	u2, _ := common.NewURL("dubbo://192.168.0.2:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0")
	u3, _ := common.NewURL("dubbo://192.168.0.3:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0")
	if tag1 != "" {
		u1.SetParam(constant.Tagkey, tag1)
	}
	if tag2 != "" {
		u2.SetParam(constant.Tagkey, tag2)
	}
	if tag3 != "" {
		u3.SetParam(constant.Tagkey, tag3)
	}
	return []base.Invoker{
		base.NewBaseInvoker(u1),
		base.NewBaseInvoker(u2),
		base.NewBaseInvoker(u3),
	}
}

func withCache(p *PriorityRouter, invokers []base.Invoker) {
	pool, _ := p.Pool(invokers)
	p.cache = &mockCache{invokers: invokers, pool: pool}
}

func boolPtr(v bool) *bool { return &v }

func TestRouteBitmapStaticTag(t *testing.T) {
	p := newCacheRouter()
	invokers := makeInvokers("gray", "gray", "")
	withCache(p, invokers)

	t.Run("request has tag, returns only matching invokers", func(t *testing.T) {
		attachments := map[string]any{constant.Tagkey: "gray"}
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 2)
		for _, r := range result {
			assert.Equal(t, "gray", r.GetURL().GetParam(constant.Tagkey, ""))
		}
	})

	t.Run("request has no tag, returns untagged invokers", func(t *testing.T) {
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 1)
		assert.Empty(t, result[0].GetURL().GetParam(constant.Tagkey, ""))
	})

	t.Run("request has non-matching tag with force, returns empty", func(t *testing.T) {
		attachments := map[string]any{constant.Tagkey: "nonexistent", constant.ForceUseTag: "true"}
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Empty(t, result)
	})
}

func TestRouteBitmapDynamicTagAddress(t *testing.T) {
	p := newCacheRouter()
	invokers := makeInvokers("gray", "gray", "")
	withCache(p, invokers)

	p.routerConfigs.Store(consumerUrl.GetParam(constant.ApplicationKey, "")+constant.TagRouterRuleSuffix, global.RouterConfig{
		Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
		Force:   boolPtr(false),
		Enabled: boolPtr(true),
		Valid:   boolPtr(true),
		Tags: []global.Tag{{
			Name:      "gray",
			Addresses: []string{"192.168.0.1:20000"},
		}},
	})

	t.Run("address matches only selected invoker", func(t *testing.T) {
		attachments := map[string]any{constant.Tagkey: "gray"}
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 1)
		assert.Equal(t, "192.168.0.1:20000", result[0].GetURL().Location)
	})
}

func TestRouteBitmapDynamicTagAnyHost(t *testing.T) {
	p := newCacheRouter()
	invokers := makeInvokers("gray", "gray", "gray")
	withCache(p, invokers)

	p.routerConfigs.Store(consumerUrl.GetParam(constant.ApplicationKey, "")+constant.TagRouterRuleSuffix, global.RouterConfig{
		Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
		Force:   boolPtr(true),
		Enabled: boolPtr(true),
		Valid:   boolPtr(true),
		Tags: []global.Tag{{
			Name:      "gray",
			Addresses: []string{constant.AnyHostValue + ":20000"},
		}},
	})

	t.Run("anyhost address matches all invokers on that port", func(t *testing.T) {
		attachments := map[string]any{constant.Tagkey: "gray"}
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 3)
	})
}

func TestRouteBitmapDynamicEmptyTag(t *testing.T) {
	p := newCacheRouter()
	invokers := makeInvokers("gray", "", "")
	withCache(p, invokers)

	p.routerConfigs.Store(consumerUrl.GetParam(constant.ApplicationKey, "")+constant.TagRouterRuleSuffix, global.RouterConfig{
		Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
		Force:   boolPtr(false),
		Enabled: boolPtr(true),
		Valid:   boolPtr(true),
		Tags: []global.Tag{{
			Addresses: []string{"192.168.0.1:20000"},
		}},
	})

	t.Run("empty request tag with address exclusion", func(t *testing.T) {
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.Len(t, result, 2)
		for _, r := range result {
			assert.NotEqual(t, "192.168.0.1:20000", r.GetURL().Location)
		}
	})
}

func TestRouteBitmapDynamicParamExact(t *testing.T) {
	p := newCacheRouter()
	invokers := makeInvokers("gray", "gray", "")
	invokers[1].GetURL().SetParam("version", "v2")
	withCache(p, invokers)

	p.routerConfigs.Store(consumerUrl.GetParam(constant.ApplicationKey, "")+constant.TagRouterRuleSuffix, global.RouterConfig{
		Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
		Force:   boolPtr(false),
		Enabled: boolPtr(true),
		Valid:   boolPtr(true),
		Tags: []global.Tag{{
			Name: "gray",
			Match: []*common.ParamMatch{
				{Key: "version", Value: common.StringMatch{Exact: "v2"}},
			},
		}},
	})

	t.Run("indexed param exact match via bitmap", func(t *testing.T) {
		attachments := map[string]any{constant.Tagkey: "gray"}
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 1)
		assert.Equal(t, "192.168.0.2:20000", result[0].GetURL().Location)
	})
}

func TestRouteBitmapMatchFallback(t *testing.T) {
	p := newCacheRouter()
	invokers := makeInvokers("gray", "gray", "")
	withCache(p, invokers)

	p.routerConfigs.Store(consumerUrl.GetParam(constant.ApplicationKey, "")+constant.TagRouterRuleSuffix, global.RouterConfig{
		Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
		Force:   boolPtr(false),
		Enabled: boolPtr(true),
		Valid:   boolPtr(true),
		Tags: []global.Tag{{
			Name: "gray",
			Match: []*common.ParamMatch{
				{Key: "environment", Value: common.StringMatch{Exact: "prod"}},
			},
		}},
	})

	t.Run("unindexed param key falls back to original path", func(t *testing.T) {
		attachments := map[string]any{constant.Tagkey: "gray"}
		result := p.Route(invokers, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.Len(t, result, 1)
		assert.Empty(t, result[0].GetURL().GetParam(constant.Tagkey, ""))
	})
}

func TestRouteBitmapEquivalence(t *testing.T) {
	// Same scenario via bitmap and original path should produce identical results.
	invokers := makeInvokers("gray", "gray", "")

	pCache := newCacheRouter()
	withCache(pCache, invokers)
	pCache.routerConfigs.Store(consumerUrl.GetParam(constant.ApplicationKey, "")+constant.TagRouterRuleSuffix, global.RouterConfig{
		Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
		Force:   boolPtr(false),
		Enabled: boolPtr(true),
		Valid:   boolPtr(true),
		Tags: []global.Tag{{
			Name:      "gray",
			Addresses: []string{"192.168.0.1:20000"},
		}},
	})

	pFallback := newCacheRouter()
	pFallback.routerConfigs.Store(consumerUrl.GetParam(constant.ApplicationKey, "")+constant.TagRouterRuleSuffix, global.RouterConfig{
		Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
		Force:   boolPtr(false),
		Enabled: boolPtr(true),
		Valid:   boolPtr(true),
		Tags: []global.Tag{{
			Name:      "gray",
			Addresses: []string{"192.168.0.1:20000"},
		}},
	})

	t.Run("bitmap and fallback produce same result", func(t *testing.T) {
		attachments := map[string]any{constant.Tagkey: "gray"}
		invoc := invocation.NewRPCInvocation("GetUser", nil, attachments)

		rCache := pCache.Route(invokers, consumerUrl, invoc)
		rFallback := pFallback.Route(invokers, consumerUrl, invoc)

		assert.Len(t, rCache, len(rFallback))
		for i := range rCache {
			assert.Equal(t, rCache[i].GetURL().Location, rFallback[i].GetURL().Location)
		}
	})
}

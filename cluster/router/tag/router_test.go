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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	common_cfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/protocol"
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
	t.Run("staticEmptyTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.True(t, len(result) == 3)
	})
	t.Run("staticEmptyTag_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 3)
	})
	t.Run("staticEmptyTag_requestHasTag_force", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag", constant.ForceUseTag: "true"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 0)
	})
	t.Run("staticTag_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		ivk2.GetURL().SetParam(constant.Tagkey, "tag")
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 1)
		assert.True(t, result[0].GetURL().GetParam(constant.Tagkey, "") == "tag")
	})
	t.Run("staticTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		ivk2.GetURL().SetParam(constant.Tagkey, "tag")
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.True(t, len(result) == 2)
	})

	initUrl()
	t.Run("dynamicEmptyTag_requestEmptyTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   true,
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.True(t, len(result) == 3)
	})

	t.Run("dynamicEmptyTag_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   true,
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 3)
	})

	t.Run("dynamicTag_requestEmptyTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   true,
			Tags: []config.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.3:20000"},
			}},
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.True(t, len(result) == 2)
	})

	t.Run("dynamicTag_emptyAddress_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   true,
			Tags: []config.Tag{{
				Name: "tag",
			}},
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 3)
	})

	t.Run("dynamicTag_address_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   true,
			Tags: []config.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.3:20000"},
			}},
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 1)
	})

	t.Run("dynamicTag_twoAddress_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   true,
			Tags: []config.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.3:20000"},
			}},
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, url3, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 2)
	})

	t.Run("dynamicTag_addressNotMatch_requestHasTag", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   true,
			Tags: []config.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.4:20000"},
			}},
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, consumerUrl, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 3)
	})

	t.Run("dynamicTag_notValid", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		p.routerConfigs.Store(consumerUrl.Service()+constant.TagRouterRuleSuffix, config.RouterConfig{
			Key:     consumerUrl.Service() + constant.TagRouterRuleSuffix,
			Force:   false,
			Enabled: true,
			Valid:   false,
			Tags: []config.Tag{{
				Name:      "tag",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.3:20000"},
			}},
		})
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		attachments := map[string]interface{}{constant.Tagkey: "tag"}
		result := p.Route(invokerList, url3, invocation.NewRPCInvocation("GetUser", nil, attachments))
		assert.True(t, len(result) == 3)
	})

	t.Run("dynamicConfigIsNull", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
		invokerList = append(invokerList, ivk)
		invokerList = append(invokerList, ivk1)
		invokerList = append(invokerList, ivk2)
		result := p.Route(invokerList, url3, invocation.NewRPCInvocation("GetUser", nil, nil))
		assert.True(t, len(result) == 3)
	})
}

func TestNotify(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		initUrl()
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
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
		value, ok := p.routerConfigs.Load(url3.Service() + constant.TagRouterRuleSuffix)
		assert.True(t, ok)
		routerCfg := value.(config.RouterConfig)
		assert.True(t, routerCfg.Key == "org.apache.dubbo.UserProvider.Test")
		assert.True(t, len(routerCfg.Tags) == 2)
		assert.True(t, routerCfg.Enabled)
	})

	t.Run("configNotValid", func(t *testing.T) {
		p, err := NewTagPriorityRouter()
		assert.Nil(t, err)
		initUrl()
		ivk := protocol.NewBaseInvoker(url1)
		ivk1 := protocol.NewBaseInvoker(url2)
		ivk2 := protocol.NewBaseInvoker(url3)
		invokerList := make([]protocol.Invoker, 0, 3)
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
		value, ok := p.routerConfigs.Load(url3.Service() + constant.TagRouterRuleSuffix)
		assert.True(t, ok == false)
		assert.True(t, value == nil)
	})
}

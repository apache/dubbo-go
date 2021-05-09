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

package v3router

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	mockVSConfigPath = "./test_file/virtual_service.yml"
	mockDRConfigPath = "./test_file/dest_rule.yml"
)

func TestNewUniformRouterChain(t *testing.T) {
	vsBytes, _ := yaml.LoadYMLConfig(mockVSConfigPath)
	drBytes, _ := yaml.LoadYMLConfig(mockDRConfigPath)
	rc, err := NewUniformRouterChain(vsBytes, drBytes, make(chan struct{}))
	assert.Nil(t, err)
	assert.NotNil(t, rc)
}

type ruleTestItemStruct struct {
	name                             string
	matchMethodNameExact             string
	RouterDestHost                   string
	RouterDestSubset                 string
	RouterFallBackDestHost           string
	RouterFallBackDestSubset         string
	RouterFallBackFallBackDestHost   string
	RouterFallBackFallBackDestSubset string
	fallbackLevel                    int
	RouterSize                       int
}

func TestParseConfigFromFile(t *testing.T) {
	vsBytes, _ := yaml.LoadYMLConfig(mockVSConfigPath)
	drBytes, _ := yaml.LoadYMLConfig(mockDRConfigPath)
	routers, err := parseFromConfigToRouters(vsBytes, drBytes, make(chan struct{}, 1))
	fmt.Println(routers, err)
	assert.Equal(t, len(routers), 1)
	assert.NotNil(t, routers[0].dubboRouter)
	assert.Equal(t, len(routers[0].dubboRouter.uniformRules), 2)
	for i, v := range routers[0].dubboRouter.uniformRules {
		if i == 0 {
			assert.Equal(t, len(v.services), 2)
			assert.Equal(t, "com.taobao.hsf.demoService:1.0.0", v.services[0].Exact)
			assert.Equal(t, "", v.services[0].Regex)
			assert.Equal(t, "", v.services[0].NoEmpty)
			assert.Equal(t, "", v.services[0].Empty)
			assert.Equal(t, "", v.services[0].Prefix)

			assert.Equal(t, "com.taobao.hsf.demoService:2.0.0", v.services[1].Exact)
			assert.Equal(t, "", v.services[1].Regex)
			assert.Equal(t, "", v.services[1].NoEmpty)
			assert.Equal(t, "", v.services[1].Empty)
			assert.Equal(t, "", v.services[1].Prefix)

			assert.Equal(t, len(v.virtualServiceRules), 4)

			ruleTestItemStructList := []ruleTestItemStruct{
				{
					name:                             "sayHello-String-method-route",
					matchMethodNameExact:             "sayHello",
					RouterDestHost:                   "demo",
					RouterDestSubset:                 "v1",
					RouterFallBackDestHost:           "demo",
					RouterFallBackDestSubset:         "v2",
					RouterFallBackFallBackDestHost:   "demo",
					RouterFallBackFallBackDestSubset: "v3",
					RouterSize:                       1,
					fallbackLevel:                    3,
				},
				{
					name:                     "sayHello-method-route",
					matchMethodNameExact:     "s-method",
					RouterDestHost:           "demo",
					RouterDestSubset:         "v2",
					RouterFallBackDestHost:   "demo",
					RouterFallBackDestSubset: "v3",
					RouterSize:               1,
					fallbackLevel:            2,
				},
				{
					name:                 "some-method-route",
					matchMethodNameExact: "some-method",
					RouterDestHost:       "demo",
					RouterDestSubset:     "v4",
					RouterSize:           1,
					fallbackLevel:        1,
				},
				{
					name:                             "final",
					matchMethodNameExact:             "GetUser",
					RouterDestHost:                   "demo",
					RouterDestSubset:                 "v1",
					RouterFallBackDestHost:           "demo",
					RouterFallBackDestSubset:         "v2",
					RouterFallBackFallBackDestHost:   "demo",
					RouterFallBackFallBackDestSubset: "v3",
					RouterSize:                       2,
					fallbackLevel:                    3,
				},
			}
			for i, vsRule := range v.virtualServiceRules {
				assert.NotNil(t, v.virtualServiceRules[i].routerItem)
				assert.Equal(t, ruleTestItemStructList[i].name, vsRule.routerItem.Name)
				assert.Equal(t, 1, len(vsRule.routerItem.Match))
				assert.NotNil(t, vsRule.routerItem.Match[0].Method)
				assert.Equal(t, ruleTestItemStructList[i].matchMethodNameExact, vsRule.routerItem.Match[0].Method.NameMatch.Exact)
				assert.Equal(t, ruleTestItemStructList[i].RouterSize, len(vsRule.routerItem.Router))
				assert.NotNil(t, vsRule.routerItem.Router[0].Destination)
				assert.Equal(t, ruleTestItemStructList[i].RouterDestHost, vsRule.routerItem.Router[0].Destination.Host)
				assert.Equal(t, ruleTestItemStructList[i].RouterDestSubset, vsRule.routerItem.Router[0].Destination.Subset)
				if vsRule.routerItem.Router[0].Destination.Fallback == nil {
					assert.Equal(t, 1, ruleTestItemStructList[i].fallbackLevel)
					continue
				}
				newRule := vsRule.routerItem.Router[0].Destination.Fallback
				assert.NotNil(t, newRule.Destination)
				assert.Equal(t, ruleTestItemStructList[i].RouterFallBackDestHost, newRule.Destination.Host)
				assert.Equal(t, ruleTestItemStructList[i].RouterFallBackDestSubset, newRule.Destination.Subset)
				if newRule.Destination.Fallback == nil {
					assert.Equal(t, 2, ruleTestItemStructList[i].fallbackLevel)
					continue
				}

				newRule = newRule.Destination.Fallback
				assert.NotNil(t, newRule.Destination)
				assert.Equal(t, ruleTestItemStructList[i].RouterFallBackFallBackDestHost, newRule.Destination.Host)
				assert.Equal(t, ruleTestItemStructList[i].RouterFallBackFallBackDestSubset, newRule.Destination.Subset)
				if newRule.Destination.Fallback == nil {
					assert.Equal(t, 3, ruleTestItemStructList[i].fallbackLevel)
				}
			}

			destMap := v.DestinationLabelListMap
			v1Val, ok := destMap["v1"]
			assert.True(t, ok)
			v1SigmaVal, ok := v1Val["sigma.ali/mg"]
			assert.True(t, ok)
			assert.Equal(t, "v1-host", v1SigmaVal)
			v1Generic, ok := v1Val["generic"]
			assert.True(t, ok)
			assert.Equal(t, "false", v1Generic)

			v2Val, ok := destMap["v2"]
			assert.True(t, ok)
			v2Generic, ok := v2Val["generic"]
			assert.True(t, ok)
			assert.Equal(t, "false", v2Generic)

			v3Val, ok := destMap["v3"]
			assert.True(t, ok)
			v3SigmaVal, ok := v3Val["sigma.ali/mg"]
			assert.True(t, ok)
			assert.Equal(t, "v3-host", v3SigmaVal)

		}
	}
}

func TestRouterChain_Route(t *testing.T) {
	vsBytes, _ := yaml.LoadYMLConfig(mockVSConfigPath)
	drBytes, _ := yaml.LoadYMLConfig(mockDRConfigPath)
	rc, err := NewUniformRouterChain(vsBytes, drBytes, make(chan struct{}))
	assert.Nil(t, err)
	assert.NotNil(t, rc)
	newGoodURL, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0")
	newBadURL1, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0")
	newBadURL2, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0")
	goodIvk := protocol.NewBaseInvoker(newGoodURL)
	b1 := protocol.NewBaseInvoker(newBadURL1)
	b2 := protocol.NewBaseInvoker(newBadURL2)
	invokerList := make([]protocol.Invoker, 3)
	invokerList = append(invokerList, goodIvk)
	invokerList = append(invokerList, b1)
	invokerList = append(invokerList, b2)
	result := rc.Route(invokerList, newGoodURL, invocation.NewRPCInvocation("GetUser", nil, nil))
	assert.Equal(t, 0, len(result))
	//todo test find target invoker
}

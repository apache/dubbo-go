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

package registry

import (
	"fmt"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
)

func TestDefaultServiceInstance_GetAddressAndWeight(t *testing.T) {
	inst := &DefaultServiceInstance{Host: "127.0.0.1", Port: 20880}
	assert.Equal(t, "127.0.0.1:20880", inst.GetAddress())

	instNoPort := &DefaultServiceInstance{Host: "127.0.0.1"}
	assert.Equal(t, "127.0.0.1", instNoPort.GetAddress())

	instWithAddress := &DefaultServiceInstance{Address: "custom"}
	assert.Equal(t, "custom", instWithAddress.GetAddress())

	assert.EqualValues(t, constant.DefaultWeight, (&DefaultServiceInstance{}).GetWeight())
	assert.Equal(t, int64(50), (&DefaultServiceInstance{Weight: 50}).GetWeight())
}

func TestDefaultServiceInstance_ToURLsWithEndpoints(t *testing.T) {
	serviceURL := common.NewURLWithOptions(
		common.WithProtocol("tri"),
		common.WithIp("127.0.0.1"),
		common.WithPort("2000"),
		common.WithPath("DemoService"),
		common.WithInterface("DemoService"),
		common.WithMethods([]string{"SayHello"}),
		common.WithParams(url.Values{constant.WeightKey: {"0"}}),
	)
	serviceInfo := info.NewServiceInfoWithURL(serviceURL)

	t.Run("pick matching endpoint", func(t *testing.T) {
		instance := &DefaultServiceInstance{
			Host: "127.0.0.1",
			Port: 20880,
			Metadata: map[string]string{
				constant.ServiceInstanceEndpoints: `[{"port":3000,"protocol":"tri"},{"port":3001,"protocol":"rest"}]`,
			},
			Tag: "gray",
		}

		urls := instance.ToURLs(serviceInfo)
		assert.Len(t, urls, 1)
		got := urls[0]
		assert.Equal(t, "tri", got.Protocol)
		assert.Equal(t, "127.0.0.1", got.Ip)
		assert.Equal(t, "3000", got.Port)
		assert.Equal(t, "DemoService", got.Service())
		assert.Equal(t, "gray", got.GetParam(constant.Tagkey, ""))
		assert.Equal(t, fmt.Sprint(constant.DefaultWeight), got.GetParam(constant.WeightKey, ""))
	})

	t.Run("fallback without endpoints", func(t *testing.T) {
		instance := &DefaultServiceInstance{
			Host: "127.0.0.1",
			Port: 20880,
			Metadata: map[string]string{
				constant.ServiceInstanceEndpoints: "invalid-json",
			},
		}
		urls := instance.ToURLs(serviceInfo)
		assert.Len(t, urls, 1)
		assert.Equal(t, "20880", urls[0].Port)
	})
}

func TestDefaultServiceInstance_GetEndPointsAndCopy(t *testing.T) {
	instance := &DefaultServiceInstance{
		Host:    "127.0.0.1",
		Port:    20880,
		Tag:     "blue",
		Enable:  true,
		Healthy: true,
		Metadata: map[string]string{
			constant.ServiceInstanceEndpoints: `[{"port":20880,"protocol":"tri"}]`,
			"custom":                          "value",
		},
		ServiceMetadata: info.NewMetadataInfo("app", ""),
	}

	endpoints := instance.GetEndPoints()
	assert.Len(t, endpoints, 1)
	assert.Equal(t, 20880, endpoints[0].Port)
	assert.Equal(t, "tri", endpoints[0].Protocol)

	copied := instance.Copy(&Endpoint{Port: 3001})
	copiedInstance, ok := copied.(*DefaultServiceInstance)
	if !ok {
		t.Fatalf("expected *DefaultServiceInstance, got %T", copied)
	}
	assert.Equal(t, 3001, copiedInstance.Port)
	assert.Equal(t, instance.GetAddress(), copiedInstance.ID)
	assert.Equal(t, instance.Metadata, copiedInstance.Metadata)
	assert.Equal(t, instance.Tag, copiedInstance.Tag)

	broken := &DefaultServiceInstance{
		Metadata: map[string]string{
			constant.ServiceInstanceEndpoints: "{broken",
		},
	}
	assert.Nil(t, broken.GetEndPoints())
}

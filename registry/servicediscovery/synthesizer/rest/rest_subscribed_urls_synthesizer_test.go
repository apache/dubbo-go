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
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/registry"
)

func TestRestSubscribedURLsSynthesizer_Synthesize(t *testing.T) {
	syn := RestSubscribedURLsSynthesizer{}
	subUrl, _ := common.NewURL("rest://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	instances := []registry.ServiceInstance{
		&registry.DefaultServiceInstance{
			Id:          "test1",
			ServiceName: "test1",
			Host:        "127.0.0.1:80",
			Port:        80,
			Enable:      false,
			Healthy:     false,
			Metadata:    nil,
		},
		&registry.DefaultServiceInstance{
			Id:          "test2",
			ServiceName: "test2",
			Host:        "127.0.0.2:8081",
			Port:        8081,
			Enable:      false,
			Healthy:     false,
			Metadata:    nil,
		},
	}

	var expectUrls []*common.URL
	u1 := common.NewURLWithOptions(common.WithProtocol("rest"), common.WithIp("127.0.0.1"),
		common.WithPort("80"), common.WithPath("org.apache.dubbo-go.mockService"),
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SIDE_KEY, constant.PROVIDER_PROTOCOL),
		common.WithParamsValue(constant.APPLICATION_KEY, "test1"),
		common.WithParamsValue(constant.REGISTRY_KEY, "true"))
	u2 := common.NewURLWithOptions(common.WithProtocol("rest"), common.WithIp("127.0.0.2"),
		common.WithPort("8081"), common.WithPath("org.apache.dubbo-go.mockService"),
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SIDE_KEY, constant.PROVIDER_PROTOCOL),
		common.WithParamsValue(constant.APPLICATION_KEY, "test2"),
		common.WithParamsValue(constant.REGISTRY_KEY, "true"))
	expectUrls = append(expectUrls, u1, u2)
	result := syn.Synthesize(subUrl, instances)
	assert.Equal(t, expectUrls, result)
}

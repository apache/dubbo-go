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

package config

import (
	"maps"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func cloneRPCServiceMap(src map[string]common.RPCService) map[string]common.RPCService {
	return maps.Clone(src)
}

func TestGetConsumerService(t *testing.T) {

	SetConsumerService(&HelloService{})
	SetProviderService(&HelloService{})

	service := GetConsumerService("HelloService")
	reference := service.(*HelloService).Reference()

	assert.Equal(t, "HelloService", reference)

	SetConsumerServiceByInterfaceName("org.apache.dubbo.HelloService", &HelloService{})
	service = GetConsumerServiceByInterfaceName("org.apache.dubbo.HelloService")
	reference = service.(*HelloService).Reference()
	assert.Equal(t, "HelloService", reference)

	callback := GetCallback(reference)
	assert.Nil(t, callback)
}

func TestGetProviderServiceMapReturnsCopy(t *testing.T) {
	proServicesLock.Lock()
	originalProServices := cloneRPCServiceMap(proServices)
	originalProServicesInfo := maps.Clone(proServicesInfo)
	proServices = map[string]common.RPCService{}
	proServicesInfo = map[string]any{}
	proServicesLock.Unlock()

	defer func() {
		proServicesLock.Lock()
		proServices = originalProServices
		proServicesInfo = originalProServicesInfo
		proServicesLock.Unlock()
	}()

	svc := &HelloService{}
	SetProviderService(svc)

	got := GetProviderServiceMap()
	require.Len(t, got, 1)

	got["Injected"] = &HelloService{}
	got["HelloService"] = &HelloService{}

	proServicesLock.Lock()
	_, hasInjected := proServices["Injected"]
	_, hasHelloService := proServices["HelloService"]
	proServicesLock.Unlock()

	assert.False(t, hasInjected)
	assert.True(t, hasHelloService)
}

func TestGetConsumerServiceMapReturnsCopy(t *testing.T) {
	conServicesLock.Lock()
	originalConServices := cloneRPCServiceMap(conServices)
	conServices = map[string]common.RPCService{}
	conServicesLock.Unlock()

	defer func() {
		conServicesLock.Lock()
		conServices = originalConServices
		conServicesLock.Unlock()
	}()

	svc := &HelloService{}
	SetConsumerService(svc)

	got := GetConsumerServiceMap()
	require.Len(t, got, 1)

	got["Injected"] = &HelloService{}
	got["HelloService"] = &HelloService{}

	conServicesLock.Lock()
	_, hasInjected := conServices["Injected"]
	stored := conServices["HelloService"]
	conServicesLock.Unlock()

	assert.False(t, hasInjected)
	assert.Equal(t, svc, stored)
}

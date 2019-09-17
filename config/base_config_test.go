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
	"context"
	"fmt"
	"reflect"
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)
import (
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config_center"
)

func Test_refresh(t *testing.T) {
	c := &BaseConfig{}
	mockMap := map[string]string{}
	mockMap["dubbo.registries.shanghai_reg1.protocol"] = "mock100"
	mockMap["dubbo.reference.com.MockService.MockService.retries"] = "10"
	mockMap["dubbo.com.MockService.MockService.GetUser.retries"] = "10"
	mockMap["dubbo.consumer.check"] = "false"
	mockMap["dubbo.application.name"] = "dubbo"

	config.GetEnvInstance().UpdateExternalConfigMap(mockMap)

	father := &ConsumerConfig{
		Check: &[]bool{true}[0],
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
		Registries: map[string]*RegistryConfig{
			//"shanghai_reg1": {
			//	id:         "shanghai_reg1",
			//	Protocol:   "mock",
			//	TimeoutStr: "2s",
			//	Group:      "shanghai_idc",
			//	Address:    "127.0.0.1:2181",
			//	Username:   "user1",
			//	Password:   "pwd1",
			//},
			"shanghai_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.2:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg1": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.3:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.4:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
		},
		References: map[string]*ReferenceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*MethodConfig{
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser",
						Retries:       "2",
						Loadbalance:   "random",
					},
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser1",
						Retries:       "2",
						Loadbalance:   "random",
					},
				},
			},
		},
	}

	c.SetFatherConfig(father)
	c.fresh()
	assert.Equal(t, "mock100", father.Registries["shanghai_reg1"].Protocol)
	assert.Equal(t, "10", father.References["MockService"].Retries)

	assert.Equal(t, "10", father.References["MockService"].Methods[0].Retries)
	assert.Equal(t, &[]bool{false}[0], father.Check)
	assert.Equal(t, "dubbo", father.ApplicationConfig.Name)
}

func Test_appExternal_refresh(t *testing.T) {
	c := &BaseConfig{}
	mockMap := map[string]string{}
	mockMap["dubbo.registries.shanghai_reg1.protocol"] = "mock100"
	mockMap["dubbo.reference.com.MockService.MockService.retries"] = "10"
	mockMap["dubbo.reference.com.MockService.retries"] = "5"
	mockMap["dubbo.com.MockService.MockService.GetUser.retries"] = "10"
	mockMap["dubbo.consumer.check"] = "false"
	mockMap["dubbo.application.name"] = "dubbo"

	config.GetEnvInstance().UpdateAppExternalConfigMap(mockMap)
	mockMap["dubbo.consumer.check"] = "true"
	config.GetEnvInstance().UpdateExternalConfigMap(mockMap)
	father := &ConsumerConfig{
		Check: &[]bool{true}[0],
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
		Registries: map[string]*RegistryConfig{
			//"shanghai_reg1": {
			//	id:         "shanghai_reg1",
			//	Protocol:   "mock",
			//	TimeoutStr: "2s",
			//	Group:      "shanghai_idc",
			//	Address:    "127.0.0.1:2181",
			//	Username:   "user1",
			//	Password:   "pwd1",
			//},
			"shanghai_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.2:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg1": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.3:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.4:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
		},
		References: map[string]*ReferenceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*MethodConfig{
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser",
						Retries:       "2",
						Loadbalance:   "random",
					},
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser1",
						Retries:       "2",
						Loadbalance:   "random",
					},
				},
			},
		},
	}

	c.SetFatherConfig(father)
	c.fresh()
	assert.Equal(t, "mock100", father.Registries["shanghai_reg1"].Protocol)
	assert.Equal(t, "10", father.References["MockService"].Retries)

	assert.Equal(t, "10", father.References["MockService"].Methods[0].Retries)
	assert.Equal(t, &[]bool{true}[0], father.Check)
	assert.Equal(t, "dubbo", father.ApplicationConfig.Name)
}

func Test_appExternalWithoutId_refresh(t *testing.T) {
	c := &BaseConfig{}
	mockMap := map[string]string{}
	mockMap["dubbo.registries.shanghai_reg1.protocol"] = "mock100"
	mockMap["dubbo.reference.com.MockService.retries"] = "10"
	mockMap["dubbo.com.MockService.MockService.GetUser.retries"] = "10"
	mockMap["dubbo.consumer.check"] = "false"
	mockMap["dubbo.application.name"] = "dubbo"

	config.GetEnvInstance().UpdateAppExternalConfigMap(mockMap)
	mockMap["dubbo.consumer.check"] = "true"
	config.GetEnvInstance().UpdateExternalConfigMap(mockMap)
	father := &ConsumerConfig{
		Check: &[]bool{true}[0],
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
		Registries: map[string]*RegistryConfig{
			//"shanghai_reg1": {
			//	id:         "shanghai_reg1",
			//	Protocol:   "mock",
			//	TimeoutStr: "2s",
			//	Group:      "shanghai_idc",
			//	Address:    "127.0.0.1:2181",
			//	Username:   "user1",
			//	Password:   "pwd1",
			//},
			"shanghai_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.2:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg1": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.3:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.4:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
		},
		References: map[string]*ReferenceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*MethodConfig{
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser",
						Retries:       "3",
						Loadbalance:   "random",
					},
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser1",
						Retries:       "2",
						Loadbalance:   "random",
					},
				},
			},
		},
	}

	c.SetFatherConfig(father)
	c.fresh()
	assert.Equal(t, "mock100", father.Registries["shanghai_reg1"].Protocol)
	assert.Equal(t, "10", father.References["MockService"].Retries)

	assert.Equal(t, "10", father.References["MockService"].Methods[0].Retries)
	assert.Equal(t, &[]bool{true}[0], father.Check)
	assert.Equal(t, "dubbo", father.ApplicationConfig.Name)
}

func Test_refresh_singleRegistry(t *testing.T) {
	c := &BaseConfig{}
	mockMap := map[string]string{}
	mockMap["dubbo.registry.address"] = "mock100://127.0.0.1:2181"
	mockMap["dubbo.reference.com.MockService.MockService.retries"] = "10"
	mockMap["dubbo.com.MockService.MockService.GetUser.retries"] = "10"
	mockMap["dubbo.consumer.check"] = "false"
	mockMap["dubbo.application.name"] = "dubbo"

	config.GetEnvInstance().UpdateExternalConfigMap(mockMap)

	father := &ConsumerConfig{
		Check: &[]bool{true}[0],
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
		Registries: map[string]*RegistryConfig{},
		Registry:   &RegistryConfig{},
		References: map[string]*ReferenceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*MethodConfig{
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser",
						Retries:       "2",
						Loadbalance:   "random",
					},
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser1",
						Retries:       "2",
						Loadbalance:   "random",
					},
				},
			},
		},
	}

	c.SetFatherConfig(father)
	c.fresh()
	assert.Equal(t, "mock100://127.0.0.1:2181", father.Registry.Address)
	assert.Equal(t, "10", father.References["MockService"].Retries)

	assert.Equal(t, "10", father.References["MockService"].Methods[0].Retries)
	assert.Equal(t, &[]bool{false}[0], father.Check)
	assert.Equal(t, "dubbo", father.ApplicationConfig.Name)
}

func Test_refreshProvider(t *testing.T) {
	c := &BaseConfig{}
	mockMap := map[string]string{}
	mockMap["dubbo.registries.shanghai_reg1.protocol"] = "mock100"
	mockMap["dubbo.service.com.MockService.MockService.retries"] = "10"
	mockMap["dubbo.com.MockService.MockService.GetUser.retries"] = "10"
	mockMap["dubbo.consumer.check"] = "false"
	mockMap["dubbo.application.name"] = "dubbo"
	mockMap["dubbo.protocols.jsonrpc1.name"] = "jsonrpc"
	mockMap["dubbo.protocols.jsonrpc1.ip"] = "127.0.0.1"
	mockMap["dubbo.protocols.jsonrpc1.port"] = "20001"

	config.GetEnvInstance().UpdateExternalConfigMap(mockMap)

	father := &ProviderConfig{
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
		Registries: map[string]*RegistryConfig{
			//"shanghai_reg1": {
			//	id:         "shanghai_reg1",
			//	Protocol:   "mock",
			//	TimeoutStr: "2s",
			//	Group:      "shanghai_idc",
			//	Address:    "127.0.0.1:2181",
			//	Username:   "user1",
			//	Password:   "pwd1",
			//},
			"shanghai_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.2:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg1": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.3:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.4:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
		},
		Services: map[string]*ServiceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*MethodConfig{
					{
						InterfaceId:   "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser",
						Retries:       "2",
						Loadbalance:   "random",
					},
					{InterfaceId: "MockService",
						InterfaceName: "com.MockService",
						Name:          "GetUser1",
						Retries:       "2",
						Loadbalance:   "random",
					},
				},
			},
		},
	}

	c.SetFatherConfig(father)
	c.fresh()
	assert.Equal(t, "mock100", father.Registries["shanghai_reg1"].Protocol)
	assert.Equal(t, "10", father.Services["MockService"].Retries)

	assert.Equal(t, "10", father.Services["MockService"].Methods[0].Retries)
	assert.Equal(t, "dubbo", father.ApplicationConfig.Name)
	assert.Equal(t, "20001", father.Protocols["jsonrpc1"].Port)
}

func Test_startConfigCenter(t *testing.T) {

	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{}
	})
	c := &BaseConfig{ConfigCenterConfig: &ConfigCenterConfig{
		Protocol:   "mock",
		Address:    "172.0.0.1",
		Group:      "dubbo",
		ConfigFile: "mockDubbo.properties",
	}}
	err := c.startConfigCenter(context.Background())
	assert.NoError(t, err)
	b, v := config.GetEnvInstance().Configuration().Back().Value.(*config.InmemoryConfiguration).GetProperty("dubbo.application.organization")
	assert.True(t, b)
	assert.Equal(t, "ikurento.com", v)
}

func Test_initializeStruct(t *testing.T) {
	consumerConfig := &ConsumerConfig{}
	tp := reflect.TypeOf(ConsumerConfig{})
	v := reflect.New(tp)
	initializeStruct(tp, v.Elem())
	fmt.Println(reflect.ValueOf(consumerConfig).Elem().Type().String())
	fmt.Println(v.Elem().Type().String())
	reflect.ValueOf(consumerConfig).Elem().Set(v.Elem())

	assert.Condition(t, func() (success bool) {
		return consumerConfig.ApplicationConfig != nil
	})
	assert.Condition(t, func() (success bool) {
		return consumerConfig.Registries != nil
	})
	assert.Condition(t, func() (success bool) {
		return consumerConfig.References != nil
	})
}

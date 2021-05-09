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

package nacos

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestNacosRegistry_Register(t *testing.T) {
	t.Skip()
	if !checkNacosServerAlive() {
		return
	}
	regurlMap := url.Values{}
	regurlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	regurlMap.Set(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true")
	regurl, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))

	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, "guangzhou-idc")
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.INTERFACE_KEY, "com.ikurento.user.UserProvider")
	urlMap.Set(constant.VERSION_KEY, "1.0.0")
	urlMap.Set(constant.CLUSTER_KEY, "mock")
	testUrl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg, err := newNacosRegistry(regurl)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("new nacos registry error:%s \n", err.Error())
		return
	}
	err = reg.Register(testUrl)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("register error:%s \n", err.Error())
		return
	}
	nacosReg := reg.(*nacosRegistry)
	service, _ := nacosReg.namingClient.GetService(vo.GetServiceParam{ServiceName: "providers:com.ikurento.user.UserProvider:1.0.0:guangzhou-idc"})
	data, _ := json.Marshal(service)
	t.Logf(string(data))
	assert.Equal(t, 1, len(service.Hosts))
}

func TestNacosRegistry_Subscribe(t *testing.T) {
	if !checkNacosServerAlive() {
		return
	}
	regurlMap := url.Values{}
	regurlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	regurlMap.Set(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true")
	regurl, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))

	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, "guangzhou-idc")
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.INTERFACE_KEY, "com.ikurento.user.UserProvider")
	urlMap.Set(constant.VERSION_KEY, "1.0.0")
	urlMap.Set(constant.CLUSTER_KEY, "mock")
	urlMap.Set(constant.NACOS_PATH_KEY, "")
	testUrl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg, _ := newNacosRegistry(regurl)
	err := reg.Register(testUrl)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("new nacos registry error:%s \n", err.Error())
		return
	}

	regurl.SetParam(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER))
	reg2, _ := newNacosRegistry(regurl)
	listener, err := reg2.(*nacosRegistry).subscribe(testUrl)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("subscribe error:%s \n", err.Error())
		return
	}
	serviceEvent, _ := listener.Next()
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("listener error:%s \n", err.Error())
		return
	}
	t.Logf("serviceEvent:%+v \n", serviceEvent)
	assert.Regexp(t, ".*ServiceEvent{Action{add}.*", serviceEvent.String())
}

func TestNacosRegistry_Subscribe_del(t *testing.T) {
	if !checkNacosServerAlive() {
		return
	}
	regurlMap := url.Values{}
	regurlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	regurlMap.Set(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true")
	regurl, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))

	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, "guangzhou-idc")
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.INTERFACE_KEY, "com.ikurento.user.UserProvider")
	urlMap.Set(constant.VERSION_KEY, "2.0.0")
	urlMap.Set(constant.CLUSTER_KEY, "mock")
	urlMap.Set(constant.NACOS_PATH_KEY, "")
	url1, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))
	url2, _ := common.NewURL("dubbo://127.0.0.2:20000/com.ikurento.user.UserProvider", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg, _ := newNacosRegistry(regurl)
	err := reg.Register(url1)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("register1 error:%s \n", err.Error())
		return
	}
	err = reg.Register(url2)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("register2 error:%s \n", err.Error())
		return
	}

	regurl.SetParam(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER))
	reg2, _ := newNacosRegistry(regurl)
	listener, err := reg2.(*nacosRegistry).subscribe(url1)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("subscribe error:%s \n", err.Error())
		return
	}

	serviceEvent1, _ := listener.Next()
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("listener1 error:%s \n", err.Error())
		return
	}
	t.Logf("serviceEvent1:%+v \n", serviceEvent1)
	assert.Regexp(t, ".*ServiceEvent{Action{add}.*", serviceEvent1.String())

	serviceEvent2, _ := listener.Next()
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("listener2 error:%s \n", err.Error())
		return
	}
	t.Logf("serviceEvent2:%+v \n", serviceEvent2)
	assert.Regexp(t, ".*ServiceEvent{Action{add}.*", serviceEvent2.String())

	nacosReg := reg.(*nacosRegistry)
	// deregister instance to mock instance offline
	_, err = nacosReg.namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip: "127.0.0.2", Port: 20000,
		ServiceName: "providers:com.ikurento.user.UserProvider:2.0.0:guangzhou-idc",
	})
	assert.NoError(t, err)

	serviceEvent3, _ := listener.Next()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Logf("serviceEvent3:%+v \n", serviceEvent3)
	assert.Regexp(t, ".*ServiceEvent{Action{delete}.*", serviceEvent3.String())
}

func TestNacosListener_Close(t *testing.T) {
	regurlMap := url.Values{}
	regurlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	regurlMap.Set(constant.NACOS_NOT_LOAD_LOCAL_CACHE, "true")
	regurl, _ := common.NewURL("registry://console.nacos.io:80", common.WithParams(regurlMap))

	urlMap := url.Values{}
	urlMap.Set(constant.GROUP_KEY, "guangzhou-idc")
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	urlMap.Set(constant.INTERFACE_KEY, "com.ikurento.user.UserProvider2")
	urlMap.Set(constant.VERSION_KEY, "1.0.0")
	urlMap.Set(constant.CLUSTER_KEY, "mock")
	urlMap.Set(constant.NACOS_PATH_KEY, "")
	url1, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider2", common.WithParams(urlMap), common.WithMethods([]string{"GetUser", "AddUser"}))
	reg, _ := newNacosRegistry(regurl)
	listener, err := reg.(*nacosRegistry).subscribe(url1)
	assert.Nil(t, err)
	if err != nil {
		t.Errorf("subscribe error:%s \n", err.Error())
		return
	}
	listener.Close()
	_, err = listener.Next()
	assert.NotNil(t, err)
}

func checkNacosServerAlive() bool {
	c := http.Client{Timeout: time.Second}
	if _, err := c.Get("http://console.nacos.io/nacos/"); err != nil {
		return false
	}
	return true
}

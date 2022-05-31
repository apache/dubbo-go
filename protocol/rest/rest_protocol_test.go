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
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/common/proxy/proxy_factory"
	rest_config "dubbo.apache.org/dubbo-go/v3/config"
)

func TestRestProtocolRefer(t *testing.T) {
	// Refer
	proto := GetRestProtocol()
	url, err := common.NewURL(mockRestCommonUrl)
	assert.NoError(t, err)
	con := rest_config.ConsumerConfig{
		ConnectTimeout: "5s",
		RequestTimeout: "5s",
	}
	rest_config.SetConsumerConfig(con)
	configMap := make(map[string]*rest_config.RestServiceConfig)
	configMap["com.ikurento.user.UserProvider"] = &rest_config.RestServiceConfig{
		Client: "resty",
	}
	rest_config.SetRestConsumerServiceConfigMap(configMap)
	invoker := proto.Refer(url)

	// make sure url
	eq := invoker.GetURL().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*RestProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*RestProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}

func TestRestProtocolExport(t *testing.T) {
	// Export
	proto := GetRestProtocol()
	url, err := common.NewURL(mockRestCommonUrl)
	assert.NoError(t, err)
	_, err = common.ServiceMap.Register(url.Service(), url.Protocol, "", "", &UserProvider{})
	assert.NoError(t, err)
	con := rest_config.ProviderConfig{}
	rest_config.SetProviderService(con)
	configMap := make(map[string]*rest_config.RestServiceConfig)
	methodConfigMap := make(map[string]*rest_config.RestMethodConfig)
	queryParamsMap := make(map[int]string)
	queryParamsMap[1] = "age"
	queryParamsMap[2] = "name"
	pathParamsMap := make(map[int]string)
	pathParamsMap[0] = "userid"
	methodConfigMap["GetUser"] = &rest_config.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUser",
		Path:           "/GetUser/{userid}",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "GET",
		PathParams:     "",
		PathParamsMap:  pathParamsMap,
		QueryParams:    "",
		QueryParamsMap: queryParamsMap,
		Body:           -1,
	}
	configMap["com.ikurento.user.UserProvider"] = &rest_config.RestServiceConfig{
		Server:            "go-restful",
		RestMethodConfigs: methodConfigMap,
	}
	rest_config.SetRestProviderServiceConfigMap(configMap)
	proxyFactory := extension.GetProxyFactory("default")
	exporter := proto.Export(proxyFactory.GetInvoker(url))
	// make sure url
	eq := exporter.GetInvoker().GetURL().URLEqual(url)
	assert.True(t, eq)
	// make sure exporterMap after 'Unexport'
	fmt.Println(url.Path)
	_, ok := proto.(*RestProtocol).ExporterMap().Load(strings.TrimPrefix(url.Path, "/"))
	assert.True(t, ok)
	exporter.Unexport()
	_, ok = proto.(*RestProtocol).ExporterMap().Load(strings.TrimPrefix(url.Path, "/"))
	assert.False(t, ok)

	// make sure serverMap after 'Destroy'
	_, ok = proto.(*RestProtocol).serverMap[url.Location]
	assert.True(t, ok)
	proto.Destroy()
	_, ok = proto.(*RestProtocol).serverMap[url.Location]
	assert.False(t, ok)
}

type UserProvider struct{}

func (p *UserProvider) Reference() string {
	return "com.ikurento.user.UserProvider"
}

func (p *UserProvider) GetUser(ctx context.Context, id int, age int32, name string, contentType string) (*User, error) {
	return &User{
		ID:   id,
		Time: nil,
		Age:  age,
		Name: name,
	}, nil
}

func (p *UserProvider) GetUserOne(ctx context.Context, user *User) (*User, error) {
	return user, nil
}

func (p *UserProvider) GetUserTwo(ctx context.Context, req []interface{}, rsp *User) error {
	m := req[0].(map[string]interface{})
	rsp.Name = m["Name"].(string)
	return nil
}

func (p *UserProvider) GetUserThree(ctx context.Context, user interface{}) (*User, error) {
	m := user.(map[string]interface{})

	u := &User{}
	u.Name = m["Name"].(string)
	return u, nil
}

func (p *UserProvider) GetUserFour(ctx context.Context, user []interface{}, id string) (*User, error) {
	m := user[0].(map[string]interface{})

	u := &User{}
	u.Name = m["Name"].(string)
	return u, nil
}

func (p *UserProvider) GetUserFive(ctx context.Context, user []interface{}) (*User, error) {
	return nil, errors.New("test error")
}

type User struct {
	ID   int
	Time *time.Time
	Age  int32
	Name string
}

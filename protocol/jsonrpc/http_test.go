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

package jsonrpc

import (
	"context"
)

import (
	perrors "github.com/pkg/errors"
)

//
//import (
//	"context"
//	"strings"
//	"testing"
//	"time"
//)
//
//import (
//	"github.com/opentracing/opentracing-go"
//
//	perrors "github.com/pkg/errors"
//
//	"github.com/stretchr/testify/assert"
//)
//
//import (
//	"dubbo.apache.org/dubbo-go/v3/common"
//	"dubbo.apache.org/dubbo-go/v3/common/constant"
//	"dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
//	"dubbo.apache.org/dubbo-go/v3/protocol"
//)
//
type (
	User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	UserProvider struct { // user map[string]User
	}
)

//
//const (
//	mockJsonCommonUrl = "jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&" +
//		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
//		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
//		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
//		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider"
//)
//
//func TestHTTPClientCall(t *testing.T) {
//	methods, err := common.ServiceMap.Register("com.ikurento.user.UserProvider", "jsonrpc", "", "", &UserProvider{})
//	assert.NoError(t, err)
//	assert.Equal(t, "GetUser,GetUser0,GetUser1,GetUser2,GetUser3,GetUser4", methods)
//
//	// Export
//	proto := GetProtocol()
//	url, err := common.NewURL(mockJsonCommonUrl)
//	assert.NoError(t, err)
//	proto.Export(&proxy_factory.ProxyInvoker{
//		BaseInvoker: *protocol.NewBaseInvoker(url),
//	})
//	time.Sleep(time.Second * 2)
//
//	client := NewHTTPClient(&HTTPOptions{})
//
//	// call GetUser
//	ctx := context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
//		"X-Proxy-ID": "dubbogo",
//		"X-Services": url.Path,
//		"X-Method":   "GetUser",
//	})
//
//	req := client.NewRequest(url, "GetUser", []interface{}{"1", "username"})
//	reply := &User{}
//	err = client.Call(ctx, url, req, reply)
//	assert.NoError(t, err)
//	assert.Equal(t, "1", reply.ID)
//	assert.Equal(t, "username", reply.Name)
//
//	// call GetUser0
//	ctx = context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
//		"X-Proxy-ID": "dubbogo",
//		"X-Services": url.Path,
//		"X-Method":   "GetUser0",
//	})
//	req = client.NewRequest(url, "GetUser0", []interface{}{"1", nil, "username"})
//	reply = &User{}
//	err = client.Call(ctx, url, req, reply)
//	assert.NoError(t, err)
//	assert.Equal(t, "1", reply.ID)
//	assert.Equal(t, "username", reply.Name)
//
//	// call GetUser1
//	ctx = context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
//		"X-Proxy-ID": "dubbogo",
//		"X-Services": url.Path,
//		"X-Method":   "GetUser1",
//	})
//	req = client.NewRequest(url, "GetUser1", []interface{}{})
//	reply = &User{}
//	err = client.Call(ctx, url, req, reply)
//	assert.True(t, strings.Contains(err.Error(), "500 Internal Server Error"))
//	assert.True(t, strings.Contains(err.Error(), "\\\"result\\\":{},\\\"error\\\":{\\\"code\\\":-32000,\\\"message\\\":\\\"error\\\"}"))
//
//	// call GetUser2
//	ctx = context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
//		"X-Proxy-ID": "dubbogo",
//		"X-Services": url.Path,
//		"X-Method":   "GetUser2",
//	})
//	req = client.NewRequest(url, "GetUser2", []interface{}{"1", "username"})
//	reply1 := []User{}
//	err = client.Call(ctx, url, req, &reply1)
//	assert.NoError(t, err)
//	assert.Equal(t, User{ID: "1", Name: "username"}, reply1[0])
//
//	// call GetUser3
//	ctx = context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
//		"X-Proxy-ID": "dubbogo",
//		"X-Services": url.Path,
//		"X-Method":   "GetUser3",
//	})
//	req = client.NewRequest(url, "GetUser3", []interface{}{"1", "username"})
//	reply1 = []User{}
//	err = client.Call(ctx, url, req, &reply1)
//	assert.NoError(t, err)
//	assert.Equal(t, User{ID: "1", Name: "username"}, reply1[0])
//
//	// call GetUser4
//	ctx = context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
//		"X-Proxy-ID": "dubbogo",
//		"X-Services": url.Path,
//		"X-Method":   "GetUser4",
//	})
//	req = client.NewRequest(url, "GetUser4", []interface{}{0})
//	reply = &User{}
//	err = client.Call(ctx, url, req, reply)
//	assert.NoError(t, err)
//	assert.Equal(t, &User{ID: "", Name: ""}, reply)
//
//	ctx = context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
//		"X-Proxy-ID": "dubbogo",
//		"X-Services": url.Path,
//		"X-Method":   "GetUser4",
//	})
//
//	span := opentracing.StartSpan("Test-Inject-Tracing-ID")
//	ctx = opentracing.ContextWithSpan(ctx, span)
//
//	req = client.NewRequest(url, "GetUser4", []interface{}{1})
//	reply = &User{}
//	err = client.Call(ctx, url, req, reply)
//	assert.NoError(t, err)
//	assert.Equal(t, &User{ID: "1", Name: ""}, reply)
//
//	// destroy
//	proto.Destroy()
//}
//
func (u *UserProvider) GetUser(ctx context.Context, req []interface{}, rsp *User) error {
	rsp.ID = req[0].(string)
	rsp.Name = req[1].(string)
	return nil
}

func (u *UserProvider) GetUser0(id string, k *User, name string) (User, error) {
	return User{ID: id, Name: name}, nil
}

func (u *UserProvider) GetUser1() error {
	return perrors.New("error")
}

func (u *UserProvider) GetUser2(ctx context.Context, req []interface{}, rsp *[]User) error {
	*rsp = append(*rsp, User{ID: req[0].(string), Name: req[1].(string)})
	return nil
}

func (u *UserProvider) GetUser3(ctx context.Context, req []interface{}) ([]User, error) {
	return []User{{ID: req[0].(string), Name: req[1].(string)}}, nil
}

func (u *UserProvider) GetUser4(id float64) (*User, error) {
	if id == 0 {
		return nil, nil
	}
	return &User{ID: "1"}, nil
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

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

package getty

import (
	"bytes"
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	. "github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
)

func TestRunSuite(t *testing.T) {
	svr, url := InitTest(t)
	client := getClient(url)
	testRequestOneWay(t, svr, url, client)
	testClient_Call(t, svr, url, client)
	testClient_AsyncCall(t, svr, url, client)
	svr.Stop()
}

func testRequestOneWay(t *testing.T, svr *Server, url *common.URL, client *Client) {

	request := remoting.NewRequest("2.0.2")
	up := &UserProvider{}
	invocation := createInvocation("GetUser", nil, nil, []interface{}{[]interface{}{"1", "username"}, up},
		[]reflect.Value{reflect.ValueOf([]interface{}{"1", "username"}), reflect.ValueOf(up)})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = false
	//user := &User{}
	err := client.Request(request, 3*time.Second, nil)
	assert.NoError(t, err)
}

func createInvocation(methodName string, callback interface{}, reply interface{}, arguments []interface{},
	parameterValues []reflect.Value) *invocation.RPCInvocation {
	return invocation.NewRPCInvocationWithOptions(invocation.WithMethodName(methodName),
		invocation.WithArguments(arguments), invocation.WithReply(reply),
		invocation.WithCallBack(callback), invocation.WithParameterValues(parameterValues))
}

func setAttachment(invocation *invocation.RPCInvocation, attachments map[string]string) {
	for key, value := range attachments {
		invocation.SetAttachments(key, value)
	}
}

func getClient(url *common.URL) *Client {
	client := NewClient(Options{
		ConnectTimeout: config.GetConsumerConfig().ConnectTimeout,
	})

	exchangeClient := remoting.NewExchangeClient(url, client, 5*time.Second, false)
	client.SetExchangeClient(exchangeClient)
	client.Connect(url)
	return client
}

func testClient_Call(t *testing.T, svr *Server, url *common.URL, c *Client) {
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	testGetBigPkg(t, c)
	testGetUser(t, c)
	testGetUser0(t, c)
	testGetUser1(t, c)
	testGetUser2(t, c)
	testGetUser3(t, c)
	testGetUser4(t, c)
	testGetUser5(t, c)
	testGetUser6(t, c)
	testGetUser61(t, c)
}

func testGetBigPkg(t *testing.T, c *Client) {
	user := &User{}
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetBigPkg", nil, nil, []interface{}{[]interface{}{nil}, user},
		[]reflect.Value{reflect.ValueOf([]interface{}{nil}), reflect.ValueOf(user)})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	pendingResponse.Reply = user
	remoting.AddPendingResponse(pendingResponse)
	err := c.Request(request, 8*time.Second, pendingResponse)
	assert.NoError(t, err)
	assert.NotEqual(t, "", user.Id)
	assert.NotEqual(t, "", user.Name)
}

func testGetUser(t *testing.T, c *Client) {
	user := &User{}
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser", nil, nil, []interface{}{"1", "username"},
		[]reflect.Value{reflect.ValueOf("1"), reflect.ValueOf("username")})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	pendingResponse.Reply = user
	remoting.AddPendingResponse(pendingResponse)
	err := c.Request(request, 3*time.Second, pendingResponse)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "1", Name: "username"}, *user)
}

func testGetUser0(t *testing.T, c *Client) {
	var (
		user *User
		err  error
	)
	user = &User{}
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser0", nil, nil, []interface{}{"1", nil, "username"},
		[]reflect.Value{reflect.ValueOf("1"), reflect.ValueOf(nil), reflect.ValueOf("username")})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	rsp := remoting.NewPendingResponse(request.ID)
	rsp.SetResponse(remoting.NewResponse(request.ID, "2.0.2"))
	remoting.AddPendingResponse(rsp)
	rsp.Reply = user
	err = c.Request(request, 3*time.Second, rsp)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "1", Name: "username"}, *user)
}

func testGetUser1(t *testing.T, c *Client) {
	var (
		err error
	)
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser1", nil, nil, []interface{}{},
		[]reflect.Value{})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	user := &User{}
	pendingResponse.Reply = user
	remoting.AddPendingResponse(pendingResponse)
	err = c.Request(request, 3*time.Second, pendingResponse)
	assert.NoError(t, err)
}

func testGetUser2(t *testing.T, c *Client) {
	var (
		err error
	)
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser2", nil, nil, []interface{}{},
		[]reflect.Value{})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	remoting.AddPendingResponse(pendingResponse)
	err = c.Request(request, 3*time.Second, pendingResponse)
	assert.EqualError(t, err, "error")
}

func testGetUser3(t *testing.T, c *Client) {
	var (
		err error
	)
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser3", nil, nil, []interface{}{},
		[]reflect.Value{})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	user2 := []interface{}{}
	pendingResponse.Reply = &user2
	remoting.AddPendingResponse(pendingResponse)
	err = c.Request(request, 3*time.Second, pendingResponse)
	assert.NoError(t, err)
	assert.Equal(t, &User{Id: "1", Name: "username"}, user2[0])
}

func testGetUser4(t *testing.T, c *Client) {
	var (
		err error
	)
	request := remoting.NewRequest("2.0.2")
	invocation := invocation.NewRPCInvocation("GetUser4", []interface{}{[]interface{}{"1", "username"}}, nil)
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	user2 := []interface{}{}
	pendingResponse.Reply = &user2
	remoting.AddPendingResponse(pendingResponse)
	err = c.Request(request, 3*time.Second, pendingResponse)
	assert.NoError(t, err)
	assert.Equal(t, &User{Id: "1", Name: "username"}, user2[0])
}

func testGetUser5(t *testing.T, c *Client) {
	var (
		err error
	)
	request := remoting.NewRequest("2.0.2")
	invocation := invocation.NewRPCInvocation("GetUser5", []interface{}{map[interface{}]interface{}{"id": "1", "name": "username"}}, nil)
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	user3 := map[interface{}]interface{}{}
	pendingResponse.Reply = &user3
	remoting.AddPendingResponse(pendingResponse)
	err = c.Request(request, 3*time.Second, pendingResponse)
	assert.NoError(t, err)
	assert.NotNil(t, user3)
	assert.Equal(t, &User{Id: "1", Name: "username"}, user3["key"])
}

func testGetUser6(t *testing.T, c *Client) {
	var (
		user *User
		err  error
	)
	user = &User{}
	request := remoting.NewRequest("2.0.2")
	invocation := invocation.NewRPCInvocation("GetUser6", []interface{}{0}, nil)
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	pendingResponse.Reply = user
	remoting.AddPendingResponse(pendingResponse)
	err = c.Request(request, 3*time.Second, pendingResponse)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "", Name: ""}, *user)
}

func testGetUser61(t *testing.T, c *Client) {
	var (
		user *User
		err  error
	)
	user = &User{}
	request := remoting.NewRequest("2.0.2")
	invocation := invocation.NewRPCInvocation("GetUser6", []interface{}{1}, nil)
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	pendingResponse := remoting.NewPendingResponse(request.ID)
	pendingResponse.Reply = user
	remoting.AddPendingResponse(pendingResponse)
	err = c.Request(request, 3*time.Second, pendingResponse)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "1", Name: ""}, *user)
}

func testClient_AsyncCall(t *testing.T, svr *Server, url *common.URL, client *Client) {
	user := &User{}
	lock := sync.Mutex{}
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser0", nil, nil, []interface{}{"4", nil, "username"},
		[]reflect.Value{reflect.ValueOf("4"), reflect.ValueOf(nil), reflect.ValueOf("username")})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = true
	rsp := remoting.NewPendingResponse(request.ID)
	rsp.SetResponse(remoting.NewResponse(request.ID, "2.0.2"))
	remoting.AddPendingResponse(rsp)
	rsp.Reply = user
	rsp.Callback = func(response common.CallbackResponse) {
		r := response.(remoting.AsyncCallbackResponse)
		rst := *r.Reply.(*remoting.Response).Result.(*protocol.RPCResult)
		assert.Equal(t, User{Id: "4", Name: "username"}, *(rst.Rest.(*User)))
		lock.Unlock()
	}
	lock.Lock()
	err := client.Request(request, 3*time.Second, rsp)
	assert.NoError(t, err)
	assert.Equal(t, User{}, *user)
	time.Sleep(1 * time.Second)
}

func InitTest(t *testing.T) (*Server, *common.URL) {

	hessian.RegisterPOJO(&User{})
	remoting.RegistryCodec("dubbo", &DubboTestCodec{})

	methods, err := common.ServiceMap.Register("", "dubbo", &UserProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetBigPkg,GetUser,GetUser0,GetUser1,GetUser2,GetUser3,GetUser4,GetUser5,GetUser6", methods)

	// config
	SetClientConf(ClientConfig{
		ConnectionNum:   2,
		HeartbeatPeriod: "5s",
		SessionTimeout:  "20s",
		PoolTTL:         600,
		PoolSize:        64,
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgWQSize:        512,
			TcpReadTimeout:   "4s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "client",
		},
	})
	assert.NoError(t, clientConf.CheckValidity())
	SetServerConfig(ServerConfig{
		SessionNumber:  700,
		SessionTimeout: "20s",
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgWQSize:        512,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "server",
		}})
	assert.NoError(t, srvConf.CheckValidity())

	url, err := common.NewURL("dubbo://127.0.0.1:20060/UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=127.0.0.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	// init server
	userProvider := &UserProvider{}
	common.ServiceMap.Register("", url.Protocol, userProvider)
	invoker := &proxy_factory.ProxyInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
	}
	handler := func(invocation *invocation.RPCInvocation) protocol.RPCResult {
		//result := protocol.RPCResult{}
		r := invoker.Invoke(context.Background(), invocation)
		result := protocol.RPCResult{
			Err:   r.Error(),
			Rest:  r.Result(),
			Attrs: r.Attachments(),
		}
		return result
	}
	server := NewServer(url, handler)
	server.Start()

	time.Sleep(time.Second * 2)

	return server, url
}

//////////////////////////////////
// provider
//////////////////////////////////

type (
	User struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}

	UserProvider struct {
		user map[string]User
	}
)

// size:4801228
func (u *UserProvider) GetBigPkg(ctx context.Context, req []interface{}, rsp *User) error {
	argBuf := new(bytes.Buffer)
	for i := 0; i < 400; i++ {
		argBuf.WriteString("击鼓其镗，踊跃用兵。土国城漕，我独南行。从孙子仲，平陈与宋。不我以归，忧心有忡。爰居爰处？爰丧其马？于以求之？于林之下。死生契阔，与子成说。执子之手，与子偕老。于嗟阔兮，不我活兮。于嗟洵兮，不我信兮。")
		argBuf.WriteString("击鼓其镗，踊跃用兵。土国城漕，我独南行。从孙子仲，平陈与宋。不我以归，忧心有忡。爰居爰处？爰丧其马？于以求之？于林之下。死生契阔，与子成说。执子之手，与子偕老。于嗟阔兮，不我活兮。于嗟洵兮，不我信兮。")
	}
	rsp.Id = argBuf.String()
	rsp.Name = argBuf.String()
	return nil
}

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}, rsp *User) error {
	rsp.Id = req[0].(string)
	rsp.Name = req[1].(string)
	return nil
}

func (u *UserProvider) GetUser0(id string, k *User, name string) (User, error) {
	return User{Id: id, Name: name}, nil
}

func (u *UserProvider) GetUser1() error {
	return nil
}

func (u *UserProvider) GetUser2() error {
	return perrors.New("error")
}

func (u *UserProvider) GetUser3(rsp *[]interface{}) error {
	*rsp = append(*rsp, User{Id: "1", Name: "username"})
	return nil
}

func (u *UserProvider) GetUser4(ctx context.Context, req []interface{}) ([]interface{}, error) {

	return []interface{}{User{Id: req[0].([]interface{})[0].(string), Name: req[0].([]interface{})[1].(string)}}, nil
}

func (u *UserProvider) GetUser5(ctx context.Context, req []interface{}) (map[interface{}]interface{}, error) {
	return map[interface{}]interface{}{"key": User{Id: req[0].(map[interface{}]interface{})["id"].(string), Name: req[0].(map[interface{}]interface{})["name"].(string)}}, nil
}

func (u *UserProvider) GetUser6(id int64) (*User, error) {
	if id == 0 {
		return nil, nil
	}
	return &User{Id: "1"}, nil
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}

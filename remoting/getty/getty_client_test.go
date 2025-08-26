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
	"dubbo.apache.org/dubbo-go/v3/common"
	. "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestRunSuite(t *testing.T) {
	svr, url := InitTest(t)
	client := getClient(url)
	assert.NotNil(t, client)
	testRequestOneWay(t, client)
	testClient_AsyncCall(t, client)
	svr.Stop()
}

func testRequestOneWay(t *testing.T, client *Client) {
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser", nil, nil, []any{"1", "username"},
		[]reflect.Value{reflect.ValueOf("1"), reflect.ValueOf("username")})
	attachment := map[string]string{InterfaceKey: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = false
	err := client.Request(request, 3*time.Second, nil)
	assert.NoError(t, err)
}

func createInvocation(methodName string, callback any, reply any, arguments []any,
	parameterValues []reflect.Value) *invocation.RPCInvocation {
	return invocation.NewRPCInvocationWithOptions(invocation.WithMethodName(methodName),
		invocation.WithArguments(arguments), invocation.WithReply(reply),
		invocation.WithCallBack(callback), invocation.WithParameterValues(parameterValues))
}

func setAttachment(invocation *invocation.RPCInvocation, attachments map[string]string) {
	for key, value := range attachments {
		invocation.SetAttachment(key, value)
	}
}

func getClient(url *common.URL) *Client {
	client := NewClient(Options{
		// todo fix timeout
		ConnectTimeout: 3 * time.Second, // config.GetConsumerConfig().ConnectTimeout,
	})
	if err := client.Connect(url); err != nil {
		return nil
	}
	return client
}

func testClient_AsyncCall(t *testing.T, client *Client) {
	user := &User{}
	wg := sync.WaitGroup{}
	request := remoting.NewRequest("2.0.2")
	invocation := createInvocation("GetUser0", nil, nil, []any{"4", nil, "username"},
		[]reflect.Value{reflect.ValueOf("4"), reflect.ValueOf(nil), reflect.ValueOf("username")})
	attachment := map[string]string{InterfaceKey: "com.ikurento.user.UserProvider"}
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
		rst := *r.Reply.(*remoting.Response).Result.(*result.RPCResult)
		assert.Equal(t, User{ID: "4", Name: "username"}, *(rst.Rest.(*User)))
		wg.Done()
	}
	wg.Add(1)
	err := client.Request(request, 3*time.Second, rsp)
	assert.NoError(t, err)
	assert.Equal(t, User{}, *user)
	wg.Done()
}

func InitTest(t *testing.T) (*Server, *common.URL) {
	hessian.RegisterPOJO(&User{})
	remoting.RegistryCodec("dubbo", &DubboTestCodec{})

	methods, err := common.ServiceMap.Register("com.ikurento.user.UserProvider", "dubbo", "", "", &UserProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetBigPkg,getBigPkg,GetUser,getUser,GetUser0,getUser0,GetUser1,getUser1,GetUser2,getUser2,GetUser3,getUser3,GetUser4,getUser4,GetUser5,getUser5,GetUser6,getUser6", methods)

	// config
	SetClientConf(ClientConfig{
		ConnectionNum:   2,
		HeartbeatPeriod: "5s",
		SessionTimeout:  "20s",
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
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
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "server",
		},
	})
	assert.NoError(t, srvConf.CheckValidity())

	url, err := common.NewURL("dubbo://127.0.0.1:20060/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=127.0.0.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	assert.NoError(t, err)
	// init server
	userProvider := &UserProvider{}
	_, err = common.ServiceMap.Register("", url.Protocol, "", "0.0.1", userProvider)
	assert.NoError(t, err)
	invoker := &proxy_factory.ProxyInvoker{
		BaseInvoker: *base.NewBaseInvoker(url),
	}
	handler := func(invocation *invocation.RPCInvocation) result.RPCResult {
		// result := protocol.RPCResult{}
		r := invoker.Invoke(context.Background(), invocation)
		res := result.RPCResult{
			Err:   r.Error(),
			Rest:  r.Result(),
			Attrs: r.Attachments(),
		}
		return res
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
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	UserProvider struct { // user map[string]User
	}
)

// size:4801228
func (u *UserProvider) GetBigPkg(ctx context.Context, req []any, rsp *User) error {
	argBuf := new(bytes.Buffer)
	for i := 0; i < 400; i++ {
		argBuf.WriteString("击鼓其镗，踊跃用兵。土国城漕，我独南行。从孙子仲，平陈与宋。不我以归，忧心有忡。爰居爰处？爰丧其马？于以求之？于林之下。死生契阔，与子成说。执子之手，与子偕老。于嗟阔兮，不我活兮。于嗟洵兮，不我信兮。")
		argBuf.WriteString("击鼓其镗，踊跃用兵。土国城漕，我独南行。从孙子仲，平陈与宋。不我以归，忧心有忡。爰居爰处？爰丧其马？于以求之？于林之下。死生契阔，与子成说。执子之手，与子偕老。于嗟阔兮，不我活兮。于嗟洵兮，不我信兮。")
	}
	rsp.ID = argBuf.String()
	rsp.Name = argBuf.String()
	return nil
}

func (u *UserProvider) GetUser(ctx context.Context, req []any, rsp *User) error {
	rsp.ID = req[0].(string)
	rsp.Name = req[1].(string)
	return nil
}

func (u *UserProvider) GetUser0(id string, k *User, name string) (User, error) {
	// fix testClient_AsyncCall assertion bug(#1233)
	time.Sleep(1 * time.Second)
	return User{ID: id, Name: name}, nil
}

func (u *UserProvider) GetUser1() error {
	return nil
}

func (u *UserProvider) GetUser2() error {
	return perrors.New("error")
}

func (u *UserProvider) GetUser3(rsp *[]any) error {
	*rsp = append(*rsp, User{ID: "1", Name: "username"})
	return nil
}

func (u *UserProvider) GetUser4(ctx context.Context, req []any) ([]any, error) {
	return []any{User{ID: req[0].([]any)[0].(string), Name: req[0].([]any)[1].(string)}}, nil
}

func (u *UserProvider) GetUser5(ctx context.Context, req []any) (map[any]any, error) {
	return map[any]any{"key": User{ID: req[0].(map[any]any)["id"].(string), Name: req[0].(map[any]any)["name"].(string)}}, nil
}

func (u *UserProvider) GetUser6(id int64) (*User, error) {
	if id == 0 {
		return nil, nil
	}
	return &User{ID: "1"}, nil
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}

func TestInitClient(t *testing.T) {
	originRootConf := config.GetRootConfig()
	rootConf := config.RootConfig{
		Protocols: map[string]*config.ProtocolConfig{
			"dubbo": {
				Name: "dubbo",
				Ip:   "127.0.0.1",
				Port: "20003",
			},
		},
	}
	config.SetRootConfig(rootConf)
	url, err := common.NewURL("dubbo://127.0.0.1:20003/test")
	assert.Nil(t, err)
	initServer(url)
	config.SetRootConfig(*originRootConf)
	assert.NotNil(t, srvConf)
}

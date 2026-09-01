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
	"errors"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

import (
	dubboGetty "github.com/apache/dubbo-getty"

	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type closeTrackingGettyClient struct {
	dubboGetty.Client
	closeOnce sync.Once
	closed    chan struct{}
}

func (c *closeTrackingGettyClient) Close() {
	c.closeOnce.Do(func() { close(c.closed) })
}

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
	attachment := map[string]string{constant.InterfaceKey: "com.ikurento.user.UserProvider"}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = false
	err := client.Request(request, 3*time.Second, nil)
	require.NoError(t, err)
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
	attachment := map[string]string{constant.InterfaceKey: "com.ikurento.user.UserProvider"}
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
	require.NoError(t, err)
	assert.Equal(t, User{}, *user)
	wg.Done()
}

func InitTest(t *testing.T) (*Server, *common.URL) {
	hessian.RegisterPOJO(&User{})
	remoting.RegistryCodec("dubbo", &DubboTestCodec{})

	methods, err := common.ServiceMap.Register("com.ikurento.user.UserProvider", "dubbo", "", "", &UserProvider{})
	require.NoError(t, err)
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
	require.NoError(t, clientConf.CheckValidity())
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
	require.NoError(t, srvConf.CheckValidity())

	url, err := common.NewURL("dubbo://127.0.0.1:20060/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=127.0.0.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
	require.NoError(t, err)
	// init server
	userProvider := &UserProvider{}
	_, err = common.ServiceMap.Register("", url.Protocol, "", "0.0.1", userProvider)
	require.NoError(t, err)
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
	for range 400 {
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
	return errors.New("error")
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
	url, err := common.NewURL("dubbo://127.0.0.1:20003/test")
	require.NoError(t, err)
	url.SetAttribute(constant.ProtocolConfigKey, map[string]*global.ProtocolConfig{
		"dubbo": {
			Name: "dubbo",
			Ip:   "127.0.0.1",
			Port: "20003",
		},
	})
	url.SetAttribute(constant.ApplicationKey, global.ApplicationConfig{})
	initClient(url)
}

func TestInitClientTLS(t *testing.T) {
	newURL := func() *common.URL {
		url, err := common.NewURL("dubbo://127.0.0.1:20003/test")
		require.NoError(t, err)
		url.SetAttribute(constant.ProtocolConfigKey, map[string]*global.ProtocolConfig{
			"dubbo": {
				Name:   "dubbo",
				Ip:     "127.0.0.1",
				Port:   "20003",
				Params: map[string]any{},
			},
		})
		url.SetAttribute(constant.ApplicationKey, global.ApplicationConfig{})
		return url
	}

	t.Run("valid TLS config enables SSLEnabled and TLSBuilder", func(t *testing.T) {
		clientConf = GetDefaultClientConfig()
		url := newURL()
		url.SetAttribute(constant.TLSConfigKey, &global.TLSConfig{
			CACertFile: "/path/to/ca.crt",
		})
		initClient(url)
		assert.True(t, clientConf.SSLEnabled)
		assert.NotNil(t, clientConf.TLSBuilder)
	})

	t.Run("invalid TLS config keeps TLS disabled", func(t *testing.T) {
		clientConf = GetDefaultClientConfig()
		url := newURL()
		url.SetAttribute(constant.TLSConfigKey, &global.TLSConfig{
			CACertFile: "",
		})
		initClient(url)
		assert.False(t, clientConf.SSLEnabled)
	})

	t.Run("wrong TLSConfigKey type returns early without panic", func(t *testing.T) {
		clientConf = GetDefaultClientConfig()
		url := newURL()
		url.SetAttribute(constant.TLSConfigKey, "not a *global.TLSConfig")
		initClient(url)
		assert.False(t, clientConf.SSLEnabled)
	})
}

func TestGettyConnectWaitStopsWhenClosed(t *testing.T) {
	client := NewClient(Options{ConnectTimeout: 5 * time.Second})
	started := make(chan struct{})
	var startOnce sync.Once
	available := func() bool {
		startOnce.Do(func() { close(started) })
		return false
	}
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- waitForGettyClient("127.0.0.1:1", client.opts.ConnectTimeout, available, client.done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("connection wait did not start")
	}

	start := time.Now()
	client.Close()
	err := <-waitDone

	require.Error(t, err)
	require.ErrorIs(t, err, errClientClosed)
	require.Less(t, time.Since(start), time.Second)
}

func TestGettyCloseAfterConnectionReadyBeforePublish(t *testing.T) {
	client := NewClient(Options{ConnectTimeout: time.Second})
	fakeGettyClient := &closeTrackingGettyClient{closed: make(chan struct{})}
	fakeRPCClient := &gettyRPCClient{gettyClient: fakeGettyClient}
	factoryReady := make(chan struct{})
	releaseFactory := make(chan struct{})
	connectDone := make(chan error, 1)

	go func() {
		_, err := client.getOrCreateGettyClient("", func(_ *Client, _ string) (*gettyRPCClient, error) {
			close(factoryReady)
			<-releaseFactory
			return fakeRPCClient, nil
		})
		connectDone <- err
	}()

	select {
	case <-factoryReady:
	case <-time.After(time.Second):
		t.Fatal("connection factory did not become ready")
	}

	client.Close()
	close(releaseFactory)

	select {
	case err := <-connectDone:
		require.ErrorIs(t, err, errClientClosed)
	case <-time.After(time.Second):
		t.Fatal("connection creation did not finish after release")
	}

	select {
	case <-fakeGettyClient.closed:
	case <-time.After(time.Second):
		t.Fatal("unpublished connection was not closed")
	}

	client.gettyClientMux.RLock()
	require.Nil(t, client.gettyClient)
	client.gettyClientMux.RUnlock()
}

func TestGettyConnectWaitHonorsTimeout(t *testing.T) {
	start := time.Now()
	err := waitForGettyClient("127.0.0.1:1", 30*time.Millisecond,
		func() bool { return false },
		nil,
	)

	require.Error(t, err)
	require.NotErrorIs(t, err, errClientClosed)
	require.Less(t, time.Since(start), time.Second)
}

func TestGettyNewConnectionStopsWhenClientCloses(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	client := NewClient(Options{ConnectTimeout: 5 * time.Second})
	client.conf = *GetDefaultClientConfig()
	connectDone := make(chan error, 1)
	go func() {
		_, connectErr := newGettyRPCClientConn(client, addr)
		connectDone <- connectErr
	}()
	time.AfterFunc(20*time.Millisecond, client.Close)

	start := time.Now()
	select {
	case err := <-connectDone:
		require.Error(t, err)
		require.ErrorIs(t, err, errClientClosed)
	case <-time.After(time.Second):
		t.Fatal("newGettyRPCClientConn remained blocked after Close")
	}
	require.Less(t, time.Since(start), time.Second)
}

func TestClientCloseDoesNotWaitForConnectLock(t *testing.T) {
	client := NewClient(Options{ConnectTimeout: time.Second, RequestTimeout: time.Second})
	client.connectMu.Lock()
	closeDone := make(chan struct{})
	go func() {
		client.Close()
		close(closeDone)
	}()

	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("Client.Close waited for the connection lock")
	}
	client.connectMu.Unlock()

	require.True(t, client.closed.Load())
	_, _, err := client.selectSession("")
	require.Error(t, err)
	require.ErrorIs(t, err, errClientClosed)
}

// mockWriteOnlySession embeds the getty.Session interface and overrides only
// WritePkg. It never delivers a response, so a two-way request must eventually
// hit the read-timeout branch.
type mockWriteOnlySession struct {
	dubboGetty.Session
}

func (m *mockWriteOnlySession) WritePkg(pkg any, _ time.Duration) (int, int, error) {
	return 0, 0, nil
}

func (m *mockWriteOnlySession) IsClosed() bool { return false }

func (m *mockWriteOnlySession) Stat() string { return "mock-session" }

func (m *mockWriteOnlySession) Close() {}

// TestRequestContextReadTimeout is a regression test for the request timeout
// wait: when no response arrives within the timeout, RequestContext must return
// errClientReadTimeout.
func TestRequestContextReadTimeout(t *testing.T) {
	client := NewClient(Options{RequestTimeout: 30 * time.Millisecond})
	client.gettyClient = &gettyRPCClient{
		rpcClient: client,
		sessions:  []*rpcSession{{session: &mockWriteOnlySession{}}},
	}

	request := remoting.NewRequest("2.0.2")
	request.TwoWay = true
	rsp := remoting.NewPendingResponse(request.ID)

	start := time.Now()
	err := client.RequestContext(context.Background(), request, 30*time.Millisecond, rsp)
	require.Error(t, err)
	require.ErrorIs(t, err, errClientReadTimeout)
	require.Less(t, time.Since(start), time.Second)
}

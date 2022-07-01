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

package dubbo

//
//import (
//	"bytes"
//	"context"
//	"sync"
//	"testing"
//	"time"
//)
//
//import (
//	hessian "github.com/apache/dubbo-go-hessian2"
//
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
//	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
//	"dubbo.apache.org/dubbo-go/v3/remoting"
//	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
//)
//
//func TestDubboInvokerInvoke(t *testing.T) {
//	proto, url := InitTest(t)
//
//	c := getExchangeClient(url)
//
//	invoker := NewDubboInvoker(url, c)
//	user := &User{}
//
//	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"), invocation.WithArguments([]interface{}{"1", "username"}),
//		invocation.WithReply(user), invocation.WithAttachments(map[string]interface{}{"test_key": "test_value"}))
//
//	// Call
//	res := invoker.Invoke(context.Background(), inv)
//	assert.NoError(t, res.Error())
//	assert.Equal(t, User{ID: "1", Name: "username"}, *res.Result().(*User))
//
//	// CallOneway
//	inv.SetAttachment(constant.ASYNC_KEY, "true")
//	res = invoker.Invoke(context.Background(), inv)
//	assert.NoError(t, res.Error())
//
//	// AsyncCall
//	lock := sync.Mutex{}
//	lock.Lock()
//	inv.SetCallBack(func(response common.CallbackResponse) {
//		r := response.(remoting.AsyncCallbackResponse)
//		rst := *r.Reply.(*remoting.Response).Result.(*protocol.RPCResult)
//		assert.Equal(t, User{ID: "1", Name: "username"}, *(rst.Rest.(*User)))
//		// assert.Equal(t, User{ID: "1", Name: "username"}, *r.Reply.(*Response).reply.(*User))
//		lock.Unlock()
//	})
//	res = invoker.Invoke(context.Background(), inv)
//	assert.NoError(t, res.Error())
//
//	// Err_No_Reply
//	inv.SetAttachment(constant.ASYNC_KEY, "false")
//	inv.SetReply(nil)
//	res = invoker.Invoke(context.Background(), inv)
//	assert.EqualError(t, res.Error(), "request need @response")
//
//	// testing appendCtx
//	span, ctx := opentracing.StartSpanFromContext(context.Background(), "TestOperation")
//	invoker.Invoke(ctx, inv)
//	span.Finish()
//
//	// destroy
//	lock.Lock()
//	defer lock.Unlock()
//	proto.Destroy()
//}
//
//func InitTest(t *testing.T) (protocol.Protocol, *common.URL) {
//	hessian.RegisterPOJO(&User{})
//
//	methods, err := common.ServiceMap.Register("com.ikurento.user.UserProvider", "dubbo", "", "", &UserProvider{})
//	assert.NoError(t, err)
//	assert.Equal(t, "GetBigPkg,GetUser,GetUser0,GetUser1,GetUser2,GetUser3,GetUser4,GetUser5,GetUser6", methods)
//
//	// config
//	getty.SetClientConf(getty.ClientConfig{
//		ConnectionNum:   2,
//		HeartbeatPeriod: "5s",
//		SessionTimeout:  "20s",
//		GettySessionParam: getty.GettySessionParam{
//			CompressEncoding: false,
//			TcpNoDelay:       true,
//			TcpKeepAlive:     true,
//			KeepAlivePeriod:  "120s",
//			TcpRBufSize:      262144,
//			TcpWBufSize:      65536,
//			TcpReadTimeout:   "4s",
//			TcpWriteTimeout:  "5s",
//			WaitTimeout:      "1s",
//			MaxMsgLen:        10240000000,
//			SessionName:      "client",
//		},
//	})
//	getty.SetServerConfig(getty.ServerConfig{
//		SessionNumber:  700,
//		SessionTimeout: "20s",
//		GettySessionParam: getty.GettySessionParam{
//			CompressEncoding: false,
//			TcpNoDelay:       true,
//			TcpKeepAlive:     true,
//			KeepAlivePeriod:  "120s",
//			TcpRBufSize:      262144,
//			TcpWBufSize:      65536,
//			TcpReadTimeout:   "1s",
//			TcpWriteTimeout:  "5s",
//			WaitTimeout:      "1s",
//			MaxMsgLen:        10240000000,
//			SessionName:      "server",
//		},
//	})
//
//	// Export
//	proto := GetProtocol()
//	url, err := common.NewURL("dubbo://127.0.0.1:20702/UserProvider?anyhost=true&" +
//		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
//		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
//		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
//		"side=provider&timeout=3000&timestamp=1556509797245&bean.name=UserProvider")
//	assert.NoError(t, err)
//	proto.Export(&proxy_factory.ProxyInvoker{
//		BaseInvoker: *protocol.NewBaseInvoker(url),
//	})
//
//	time.Sleep(time.Second * 2)
//
//	return proto, url
//}
//
////////////////////////////////////
//// provider
////////////////////////////////////
//
//type (
//	User struct {
//		ID   string `json:"id"`
//		Name string `json:"name"`
//	}
//
//	UserProvider struct { // user map[string]User
//	}
//)
//
//// size:4801228
//func (u *UserProvider) GetBigPkg(ctx context.Context, req []interface{}, rsp *User) error {
//	argBuf := new(bytes.Buffer)
//	for i := 0; i < 800; i++ {
//		// use chinese for test
//		argBuf.WriteString("击鼓其镗，踊跃用兵。土国城漕，我独南行。从孙子仲，平陈与宋。不我以归，忧心有忡。爰居爰处？爰丧其马？于以求之？于林之下。死生契阔，与子成说。执子之手，与子偕老。于嗟阔兮，不我活兮。于嗟洵兮，不我信兮。")
//	}
//	rsp.ID = argBuf.String()
//	rsp.Name = argBuf.String()
//	return nil
//}
//
//func (u *UserProvider) GetUser(ctx context.Context, req []interface{}, rsp *User) error {
//	rsp.ID = req[0].(string)
//	rsp.Name = req[1].(string)
//	return nil
//}
//
//func (u *UserProvider) GetUser0(id string, k *User, name string) (User, error) {
//	return User{ID: id, Name: name}, nil
//}
//
//func (u *UserProvider) GetUser1() error {
//	return nil
//}
//
//func (u *UserProvider) GetUser2() error {
//	return perrors.New("error")
//}
//
//func (u *UserProvider) GetUser3(rsp *[]interface{}) error {
//	*rsp = append(*rsp, User{ID: "1", Name: "username"})
//	return nil
//}
//
//func (u *UserProvider) GetUser4(ctx context.Context, req []interface{}) ([]interface{}, error) {
//	return []interface{}{User{ID: req[0].([]interface{})[0].(string), Name: req[0].([]interface{})[1].(string)}}, nil
//}
//
//func (u *UserProvider) GetUser5(ctx context.Context, req []interface{}) (map[interface{}]interface{}, error) {
//	return map[interface{}]interface{}{"key": User{ID: req[0].(map[interface{}]interface{})["id"].(string), Name: req[0].(map[interface{}]interface{})["name"].(string)}}, nil
//}
//
//func (u *UserProvider) GetUser6(id int64) (*User, error) {
//	if id == 0 {
//		return nil, nil
//	}
//	return &User{ID: "1"}, nil
//}
//
//func (u *UserProvider) Reference() string {
//	return "UserProvider"
//}
//
//func (u User) JavaClassName() string {
//	return "com.ikurento.user.User"
//}

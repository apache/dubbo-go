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

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/dubbogo/hessian2"
	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

func TestClient_CallOneway(t *testing.T) {
	proto, url := InitTest(t)

	c := &Client{
		pendingResponses: new(sync.Map),
		conf:             *clientConf,
	}
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	//user := &User{}
	err := c.CallOneway("127.0.0.1:20000", url, "GetUser", []interface{}{"1", "username"})
	assert.NoError(t, err)

	// destroy
	proto.Destroy()
}

func TestClient_Call(t *testing.T) {
	proto, url := InitTest(t)

	c := &Client{
		pendingResponses: new(sync.Map),
		conf:             *clientConf,
	}
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	user := &User{}
	//err := c.Call("127.0.0.1:20000", url, "GetBigPkg", []interface{}{nil}, user)
	//assert.NoError(t, err)
	//assert.NotEqual(t, "", user.Id)
	//assert.NotEqual(t, "", user.Name)

	user = &User{}
	err := c.Call("127.0.0.1:20000", url, "GetUser", []interface{}{"1", "username"}, user)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "1", Name: "username"}, *user)

	user = &User{}
	err = c.Call("127.0.0.1:20000", url, "GetUser0", []interface{}{"1", nil, "username"}, user)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "1", Name: "username"}, *user)

	err = c.Call("127.0.0.1:20000", url, "GetUser1", []interface{}{}, user)
	assert.NoError(t, err)

	err = c.Call("127.0.0.1:20000", url, "GetUser2", []interface{}{}, user)
	assert.EqualError(t, err, "error")

	user2 := []interface{}{}
	err = c.Call("127.0.0.1:20000", url, "GetUser3", []interface{}{}, &user2)
	assert.NoError(t, err)
	assert.Equal(t, &User{Id: "1", Name: "username"}, user2[0])

	user2 = []interface{}{}
	err = c.Call("127.0.0.1:20000", url, "GetUser4", []interface{}{[]interface{}{"1", "username"}}, &user2)
	assert.NoError(t, err)
	assert.Equal(t, &User{Id: "1", Name: "username"}, user2[0])

	user3 := map[interface{}]interface{}{}
	err = c.Call("127.0.0.1:20000", url, "GetUser5", []interface{}{map[interface{}]interface{}{"id": "1", "name": "username"}}, &user3)
	assert.NoError(t, err)
	assert.NotNil(t, user3)
	assert.Equal(t, &User{Id: "1", Name: "username"}, user3["key"])

	user = &User{}
	err = c.Call("127.0.0.1:20000", url, "GetUser6", []interface{}{0}, user)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "", Name: ""}, *user)

	user = &User{}
	err = c.Call("127.0.0.1:20000", url, "GetUser6", []interface{}{1}, user)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "1", Name: ""}, *user)

	// destroy
	proto.Destroy()
}

func TestClient_AsyncCall(t *testing.T) {
	proto, url := InitTest(t)

	c := &Client{
		pendingResponses: new(sync.Map),
		conf:             *clientConf,
	}
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	user := &User{}
	lock := sync.Mutex{}
	lock.Lock()
	err := c.AsyncCall("127.0.0.1:20000", url, "GetUser", []interface{}{"1", "username"}, func(response CallResponse) {
		assert.Equal(t, User{Id: "1", Name: "username"}, *response.Reply.(*User))
		lock.Unlock()
	}, user)
	assert.NoError(t, err)
	assert.Equal(t, User{}, *user)

	// destroy
	lock.Lock()
	proto.Destroy()
	lock.Unlock()
}

func InitTest(t *testing.T) (protocol.Protocol, common.URL) {

	hessian.RegisterPOJO(&User{})

	methods, err := common.ServiceMap.Register("dubbo", &UserProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetBigPkg,GetUser,GetUser0,GetUser1,GetUser2,GetUser3,GetUser4,GetUser5,GetUser6", methods)

	// config
	SetClientConf(ClientConfig{
		ConnectionNum:   2,
		HeartbeatPeriod: "5s",
		SessionTimeout:  "20s",
		FailFastTimeout: "5s",
		PoolTTL:         600,
		PoolSize:        64,
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgRQSize:        1024,
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
		SessionNumber:   700,
		SessionTimeout:  "20s",
		FailFastTimeout: "5s",
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgRQSize:        1024,
			PkgWQSize:        512,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "server",
		}})
	assert.NoError(t, srvConf.CheckValidity())

	// Export
	proto := GetProtocol()
	url, err := common.NewURL(context.Background(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	proto.Export(protocol.NewBaseInvoker(url))

	time.Sleep(time.Second * 2)

	return proto, url
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
	for i := 0; i < 4000; i++ {
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

func (u *UserProvider) Service() string {
	return "com.ikurento.user.UserProvider"
}

func (u *UserProvider) Version() string {
	return ""
}

func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}

package dubbo

import (
	"context"
	"errors"
	"github.com/dubbogo/hessian2"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type (
	User struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}

	UserProvider struct {
		user map[string]User
	}
)

func TestClient_CallOneway(t *testing.T) {
	proto, url := InitTest(t)

	c := &Client{
		pendingResponses: make(map[SequenceType]*PendingResponse),
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
		pendingResponses: make(map[SequenceType]*PendingResponse),
		conf:             *clientConf,
	}
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	user := &User{}
	err := c.Call("127.0.0.1:20000", url, "GetUser", []interface{}{"1", "username"}, user)
	assert.NoError(t, err)
	assert.Equal(t, User{Id: "1", Name: "username"}, *user)

	// destroy
	proto.Destroy()
}

func TestClient_AsyncCall(t *testing.T) {
	proto, url := InitTest(t)

	c := &Client{
		pendingResponses: make(map[SequenceType]*PendingResponse),
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
	assert.Equal(t, "GetUser,GetUser1", methods)

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
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        1024,
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
			MaxMsgLen:        1024,
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

	return proto, url
}

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}, rsp *User) error {
	rsp.Id = req[0].(string)
	rsp.Name = req[1].(string)
	return nil
}

func (u *UserProvider) GetUser1(ctx context.Context, req []interface{}, rsp *User) error {
	return errors.New("error")
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

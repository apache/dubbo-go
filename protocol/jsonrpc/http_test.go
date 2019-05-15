package jsonrpc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
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

func TestHTTPClient_Call(t *testing.T) {

	methods, err := common.ServiceMap.Register("jsonrpc", &UserProvider{})
	assert.NoError(t, err)
	assert.Equal(t, "GetUser,GetUser1", methods)

	// Export
	proto := GetProtocol()
	url, err := common.NewURL(context.Background(), "jsonrpc://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	proto.Export(protocol.NewBaseInvoker(url))

	client := NewHTTPClient(&HTTPOptions{
		HandshakeTimeout: time.Second,
		HTTPTimeout:      time.Second,
	})

	// call GetUser
	ctx := context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": url.Path,
		"X-Method":   "GetUser",
	})
	req := client.NewRequest(url, "GetUser", []interface{}{"1", "username"})
	reply := &User{}
	err = client.Call(ctx, url, req, reply)
	assert.NoError(t, err)
	assert.Equal(t, "1", reply.Id)
	assert.Equal(t, "username", reply.Name)

	// call GetUser1
	ctx = context.WithValue(context.Background(), constant.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": url.Path,
		"X-Method":   "GetUser1",
	})
	req = client.NewRequest(url, "GetUser1", []interface{}{""})
	reply = &User{}
	err = client.Call(ctx, url, req, reply)
	assert.True(t, strings.Contains(err.Error(), "500 Internal Server Error"))
	assert.True(t, strings.Contains(err.Error(), "\\\"result\\\":{},\\\"error\\\":{\\\"code\\\":-32000,\\\"message\\\":\\\"error\\\"}"))

	// destroy
	proto.Destroy()

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

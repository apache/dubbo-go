package rest

import (
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRestProtocol_Refer(t *testing.T) {
	// Refer
	proto := GetRestProtocol()
	url, err := common.NewURL(context.Background(), "rest://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	con := config.ConsumerConfig{
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}
	config.SetConsumerConfig(con)
	configMap := make(map[string]*rest_interface.RestConfig)
	configMap["com.ikurento.user.UserProvider"] = &rest_interface.RestConfig{
		Client: "resty",
	}
	SetRestConsumerServiceConfigMap(configMap)
	invoker := proto.Refer(url)

	// make sure url
	eq := invoker.GetUrl().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*RestProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*RestProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}

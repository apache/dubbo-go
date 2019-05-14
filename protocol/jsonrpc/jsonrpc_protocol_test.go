package jsonrpc

import (
	"context"
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/config"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

func TestJsonrpcProtocol_Export(t *testing.T) {
	// Export
	proto := GetProtocol()
	url, err := common.NewURL(context.Background(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	exporter := proto.Export(protocol.NewBaseInvoker(url))

	// make sure url
	eq := exporter.GetInvoker().GetUrl().URLEqual(url)
	assert.True(t, eq)

	// make sure exporterMap after 'Unexport'
	_, ok := proto.(*JsonrpcProtocol).ExporterMap().Load(url.Key())
	assert.True(t, ok)
	exporter.Unexport()
	_, ok = proto.(*JsonrpcProtocol).ExporterMap().Load(url.Key())
	assert.False(t, ok)

	// make sure serverMap after 'Destroy'
	_, ok = proto.(*JsonrpcProtocol).serverMap[url.Location]
	assert.True(t, ok)
	proto.Destroy()
	_, ok = proto.(*JsonrpcProtocol).serverMap[url.Location]
	assert.False(t, ok)
}

func TestJsonrpcProtocol_Refer(t *testing.T) {
	// Refer
	proto := GetProtocol()
	url, err := common.NewURL(context.Background(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
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
	invoker := proto.Refer(url)

	// make sure url
	eq := invoker.GetUrl().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*JsonrpcProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*JsonrpcProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}

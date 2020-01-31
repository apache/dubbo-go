package rest

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"github.com/stretchr/testify/assert"
	"strings"
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

func TestJsonrpcProtocol_Export(t *testing.T) {
	// Export
	proto := GetRestProtocol()
	url, err := common.NewURL(context.Background(), "rest://127.0.0.1:8888/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245")
	assert.NoError(t, err)
	_, err = common.ServiceMap.Register(url.Protocol, &UserProvider{})
	assert.NoError(t, err)
	con := config.ProviderConfig{}
	config.SetProviderConfig(con)
	configMap := make(map[string]*rest_interface.RestConfig)
	methodConfigMap := make(map[string]*rest_interface.RestMethodConfig)
	queryParamsMap := make(map[int]string)
	queryParamsMap[1] = "age"
	queryParamsMap[2] = "name"
	pathParamsMap := make(map[int]string)
	pathParamsMap[0] = "userid"
	methodConfigMap["GetUser"] = &rest_interface.RestMethodConfig{
		InterfaceName:  "",
		MethodName:     "GetUser",
		Path:           "/GetUser/{userid}",
		Produces:       "application/json",
		Consumes:       "application/json",
		MethodType:     "GET",
		PathParams:     "",
		PathParamsMap:  pathParamsMap,
		QueryParams:    "",
		QueryParamsMap: queryParamsMap,
		Body:           -1,
	}
	configMap["com.ikurento.user.UserProvider"] = &rest_interface.RestConfig{
		Server:               "go-restful",
		RestMethodConfigsMap: methodConfigMap,
	}
	SetRestProviderServiceConfigMap(configMap)
	proxyFactory := extension.GetProxyFactory("default")
	exporter := proto.Export(proxyFactory.GetInvoker(url))
	// make sure url
	eq := exporter.GetInvoker().GetUrl().URLEqual(url)
	assert.True(t, eq)
	// make sure exporterMap after 'Unexport'
	fmt.Println(url.Path)
	_, ok := proto.(*RestProtocol).ExporterMap().Load(strings.TrimPrefix(url.Path, "/"))
	assert.True(t, ok)
	_, ok = proto.(*RestProtocol).ExporterMap().Load(strings.TrimPrefix(url.Path, "/"))
	assert.False(t, ok)

	// make sure serverMap after 'Destroy'
	_, ok = proto.(*RestProtocol).serverMap[url.Location]
	assert.True(t, ok)
	proto.Destroy()
	_, ok = proto.(*RestProtocol).serverMap[url.Location]
	assert.False(t, ok)
}

type UserProvider struct {
}

func (p *UserProvider) Reference() string {
	return "com.ikurento.user.UserProvider"
}

func (p *UserProvider) GetUser(ctx context.Context, id int, age int32, name string) (*User, error) {
	return &User{
		Id:   id,
		Time: time.Now(),
		Age:  age,
		Name: name,
	}, nil
}

type User struct {
	Id   int
	Time time.Time
	Age  int32
	Name string
}

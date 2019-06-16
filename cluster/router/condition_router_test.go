package router

import (
	"context"
	perrors "errors"
	"fmt"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

type MockInvoker struct {
	url       common.URL
	available bool
	destroyed bool

	successCount int
}

func NewMockInvoker(url common.URL, successCount int) *MockInvoker {
	return &MockInvoker{
		url:          url,
		available:    true,
		destroyed:    false,
		successCount: successCount,
	}
}

func (bi *MockInvoker) GetUrl() common.URL {
	return bi.url
}

func (bi *MockInvoker) IsAvailable() bool {
	return bi.available
}

func (bi *MockInvoker) IsDestroyed() bool {
	return bi.destroyed
}

type rest struct {
	tried   int
	success bool
}

var count int

func (bi *MockInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	count++
	var success bool
	var err error = nil
	if count >= bi.successCount {
		success = true
	} else {
		err = perrors.New("error")
	}
	result := &protocol.RPCResult{Err: err, Rest: rest{tried: count, success: success}}

	return result
}

func (bi *MockInvoker) Destroy() {
	logger.Infof("Destroy invoker: %v", bi.GetUrl().String())
	bi.destroyed = true
	bi.available = false
}
func Test_parseRule(t *testing.T) {
	parseRule("host = 10.20.153.10 => host = 10.20.153.11")

}
func LocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
	}
	var ip string = "localhost"
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}
	return ip
}
func TestRoute_matchWhen(t *testing.T) {

}
func TestRoute_matchFilter(t *testing.T) {
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService?default.serialization=fastjson")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", LocalIp()))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", LocalIp()))
	invokers := []protocol.Invoker{NewMockInvoker(url1, 1), NewMockInvoker(url2, 2), NewMockInvoker(url3, 3)}
	option := common.WithParamsValue("rule", "host = "+LocalIp()+" => "+" host = 10.20.3.3")
	option1 := common.WithParamsValue("force", "true")
	sUrl, _ := common.NewURL(context.TODO(), "condition://0.0.0.0/com.foo.BarService", option, option1)

	router1, _ := NewConditionRouterFactory().GetRouter(sUrl)
	cUrl, _ := common.NewURL(context.TODO(), "consumer://"+LocalIp()+"/com.foo.BarService")

	routers, _ := router1.Route(invokers, cUrl, &invocation.RPCInvocation{})
	assert.Equal(t, 1, len(routers))

}

package router

import (
	"context"
	"encoding/base64"
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
	rpcInvacation := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("=> host = 1.2.3.4"))
	router, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule))
	cUrl, _ := common.NewURL(context.TODO(), "consumer://1.1.1.1/com.foo.BarService")

	matchWhen := router.(*ConditionRouter).MatchWhen(cUrl, rpcInvacation)
	assert.Equal(t, true, matchWhen)

	rule1 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router1, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule1))
	matchWhen1 := router1.(*ConditionRouter).MatchWhen(cUrl, rpcInvacation)
	assert.Equal(t, true, matchWhen1)

	rule2 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4"))
	router2, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule2))
	matchWhen2 := router2.(*ConditionRouter).MatchWhen(cUrl, rpcInvacation)
	assert.Equal(t, false, matchWhen2)

	rule3 := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.4 & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router3, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule3))
	matchWhen3 := router3.(*ConditionRouter).MatchWhen(cUrl, rpcInvacation)
	assert.Equal(t, true, matchWhen3)

	rule4 := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router4, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule4))
	matchWhen4 := router4.(*ConditionRouter).MatchWhen(cUrl, rpcInvacation)
	assert.Equal(t, true, matchWhen4)

	rule5 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.1 => host = 1.2.3.4"))
	router5, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule5))
	matchWhen5 := router5.(*ConditionRouter).MatchWhen(cUrl, rpcInvacation)
	assert.Equal(t, false, matchWhen5)

	rule6 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4"))
	router6, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule6))
	matchWhen6 := router6.(*ConditionRouter).MatchWhen(cUrl, rpcInvacation)
	assert.Equal(t, true, matchWhen6)
}
func TestRoute_matchFilter(t *testing.T) {
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService?default.serialization=fastjson")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", LocalIp()))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", LocalIp()))
	invokers := []protocol.Invoker{NewMockInvoker(url1, 1), NewMockInvoker(url2, 2), NewMockInvoker(url3, 3)}
	rule1 := base64.URLEncoding.EncodeToString([]byte("host = " + LocalIp() + " => " + " host = 10.20.3.3"))
	rule2 := base64.URLEncoding.EncodeToString([]byte("host = " + LocalIp() + " => " + " host = 10.20.3.* & host != 10.20.3.3"))
	rule3 := base64.URLEncoding.EncodeToString([]byte("host = " + LocalIp() + " => " + " host = 10.20.3.3  & host != 10.20.3.3"))
	rule4 := base64.URLEncoding.EncodeToString([]byte("host = " + LocalIp() + " => " + " host = 10.20.3.2,10.20.3.3,10.20.3.4"))
	rule5 := base64.URLEncoding.EncodeToString([]byte("host = " + LocalIp() + " => " + " host != 10.20.3.3"))
	rule6 := base64.URLEncoding.EncodeToString([]byte("host = " + LocalIp() + " => " + " serialization = fastjson"))
	router1, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule1))
	router2, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule2))
	router3, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule3))
	router4, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule4))
	router5, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule5))
	router6, _ := NewConditionRouterFactory().GetRouter(getRouteUrl(rule6))
	cUrl, _ := common.NewURL(context.TODO(), "consumer://"+LocalIp()+"/com.foo.BarService")

	fileredInvokers1, _ := router1.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers2, _ := router2.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers3, _ := router3.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers4, _ := router4.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers5, _ := router5.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers6, _ := router6.Route(invokers, cUrl, &invocation.RPCInvocation{})
	assert.Equal(t, 1, len(fileredInvokers1))
	assert.Equal(t, 0, len(fileredInvokers2))
	assert.Equal(t, 0, len(fileredInvokers3))
	assert.Equal(t, 1, len(fileredInvokers4))
	assert.Equal(t, 2, len(fileredInvokers5))
	assert.Equal(t, 1, len(fileredInvokers6))

}

func getRouteUrl(rule string) common.URL {
	url, _ := common.NewURL(context.TODO(), "condition://0.0.0.0/com.foo.BarService")
	url.AddParam("rule", rule)
	url.AddParam("force", "true")
	return url
}

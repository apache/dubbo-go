package loadbalance

import (
	"context"
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
)

func TestRoundRobinSelect(t *testing.T) {
	loadBalance := NewRoundRobinLoadBalance()

	var invokers []protocol.Invoker

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.1.0:20000/org.apache.demo.HelloService")
	invokers = append(invokers, protocol.NewBaseInvoker(url))
	i := loadBalance.Select(invokers, &invocation.RPCInvocation{})
	assert.True(t, i.GetUrl().URLEqual(url))

	for i := 1; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/org.apache.demo.HelloService", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}
	loadBalance.Select(invokers, &invocation.RPCInvocation{})

}

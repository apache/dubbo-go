package loadbalance

import (
	"context"
	"fmt"
	"strconv"
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

func TestRoundRobinByWeight(t *testing.T) {
	loadBalance := NewRoundRobinLoadBalance()

	var invokers []protocol.Invoker
	loop := 10
	for i := 1; i <= loop; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/org.apache.demo.HelloService?weight=%v", i, i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	loop = (1 + loop) * loop / 2
	selected := make(map[protocol.Invoker]int)

	for i := 1; i <= loop; i++ {
		invoker := loadBalance.Select(invokers, &invocation.RPCInvocation{})
		selected[invoker]++
	}

	for _, i := range invokers {
		w, _ := strconv.Atoi(i.GetUrl().GetParam("weight", "-1"))
		assert.True(t, selected[i] == w)
	}
}

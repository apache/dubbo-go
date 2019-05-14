package loadbalance

import (
	"context"
	"fmt"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
	"github.com/stretchr/testify/assert"
	"net/url"
	"strconv"
	"testing"
	"time"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

func Test_RandomlbSelect(t *testing.T) {
	randomlb := NewRandomLoadBalance()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}
	randomlb.Select(invokers, &invocation.RPCInvocation{})
}

func Test_RandomlbSelectWeight(t *testing.T) {
	randomlb := NewRandomLoadBalance()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	urlParams := url.Values{}
	urlParams.Set("methods.test."+constant.WEIGHT_KEY, "10000000000000")
	urll, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.100:20000/com.ikurento.user.UserProvider"), common.WithParams(urlParams))
	invokers = append(invokers, protocol.NewBaseInvoker(urll))
	ivc := &invocation.RPCInvocation{}
	ivc.SetMethod("test")

	var selectedInvoker []protocol.Invoker
	var selected float64
	for i := 0; i < 10000; i++ {
		s := randomlb.Select(invokers, ivc)
		if s.GetUrl().Ip == "192.168.1.100" {
			selected++
		}
		selectedInvoker = append(selectedInvoker, s)
	}

	assert.Condition(t, func() bool {
		//really is 0.9999999999999
		return selected/10000 > 0.9
	})
}

func Test_RandomlbSelectWarmup(t *testing.T) {
	randomlb := NewRandomLoadBalance()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	urlParams := url.Values{}
	urlParams.Set(constant.REMOTE_TIMESTAMP_KEY, strconv.FormatInt(time.Now().Add(time.Minute*(-9)).Unix(), 10))
	urll, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.100:20000/com.ikurento.user.UserProvider"), common.WithParams(urlParams))
	invokers = append(invokers, protocol.NewBaseInvoker(urll))
	ivc := &invocation.RPCInvocation{}
	ivc.SetMethod("test")

	var selectedInvoker []protocol.Invoker
	var selected float64
	for i := 0; i < 10000; i++ {
		s := randomlb.Select(invokers, ivc)
		if s.GetUrl().Ip == "192.168.1.100" {
			selected++
		}
		selectedInvoker = append(selectedInvoker, s)
	}
	assert.Condition(t, func() bool {
		return selected/10000 < 0.1
	})
}

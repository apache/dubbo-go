package impl

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/dubbo"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInitHystrixConfig(t *testing.T) {
	//Use the example config file
	config.Load()
	err := initHystrixConfig()
	assert.Nil(t, err, err)
	assert.NotNil(t, conf, "Conf should not be nil")
	assert.Equal(t, "Default", conf.Default)
	configsIn := conf.Configs["Default"]
	assert.NotNil(t, configsIn, "Configs should not be nil")
	assert.Equal(t, 20, configsIn.RequestVolumeThreshold)
	assert.Equal(t, configsIn.ErrorPercentThreshold, 50)
	assert.Equal(t, 5000, configsIn.SleepWindow)
	assert.Equal(t, 1000, configsIn.Timeout)
	serviceConfigIn := conf.Services["com.ikurento.user.UserProvider"]
	assert.NotNil(t, serviceConfigIn, "Service configs should not be nil")
	assert.Equal(t, "userp", serviceConfigIn.ServiceConfig)
	assert.NotNil(t, serviceConfigIn.Methods, "Method configs should not be nil")
	assert.Equal(t, "userp_m", serviceConfigIn.Methods["GetUser"])

}

func TestGetHystrixFilter(t *testing.T) {
	filterGot := GetHystrixFilter()
	assert.NotNil(t, filterGot)
}

type MockFallback struct {
}

func (m *MockFallback) FallbackFunc(err error, invoker protocol.Invoker, invocation protocol.Invocation, cb hystrix.CircuitBreaker) protocol.Result {
	return &protocol.RPCResult{Rest: "MOCK"}
}

func TestRefreshHystrix(t *testing.T) {
	err := RefreshHystrix()
	assert.NoError(t, err)
	assert.NotNil(t, conf, "Conf should not be nil")
	assert.Equal(t, "Default", conf.Default)
}

func TestGetConfig_1(t *testing.T) {
	_ = initHystrixConfig()
	configGot := getConfig("com.ikurento.user.UserProvider", "GetUser")
	assert.NotNil(t, configGot)
	assert.Equal(t, 1200, configGot.Timeout)
	assert.Equal(t, 12, configGot.MaxConcurrentRequests)
	assert.Equal(t, 6000, configGot.SleepWindow)
	assert.Equal(t, 60, configGot.ErrorPercentThreshold)
	assert.Equal(t, 5, configGot.RequestVolumeThreshold)
	assert.Equal(t, "exampleFallback", configGot.Fallback)
}

func TestGetConfig_2(t *testing.T) {
	_ = initHystrixConfig()
	configGot := getConfig("com.ikurento.user.UserProvider", "GetUser0")
	assert.NotNil(t, configGot)
	assert.Equal(t, 800, configGot.Timeout)
	assert.Equal(t, 8, configGot.MaxConcurrentRequests)
	assert.Equal(t, 4, configGot.SleepWindow)
	assert.Equal(t, 45, configGot.ErrorPercentThreshold)
	assert.Equal(t, 15, configGot.RequestVolumeThreshold)
	assert.Equal(t, "", configGot.Fallback)
}

func TestGetConfig_3(t *testing.T) {
	_ = initHystrixConfig()
	//This should use default
	configGot := getConfig("Mock.Service", "GetMock")
	assert.NotNil(t, configGot)
	assert.Equal(t, 1000, configGot.Timeout)
	assert.Equal(t, 10, configGot.MaxConcurrentRequests)
	assert.Equal(t, 5000, configGot.SleepWindow)
	assert.Equal(t, 50, configGot.ErrorPercentThreshold)
	assert.Equal(t, 20, configGot.RequestVolumeThreshold)
	assert.Equal(t, "", configGot.Fallback)
}

func TestGetHystrixFallback(t *testing.T) {
	fallback["mock"] = &MockFallback{}
	fallbackGot := getHystrixFallback("mock")
	assert.NotNil(t, fallbackGot)
	fallbackGot = getHystrixFallback("notExist")
	assert.Nil(t, fallbackGot)
}

func TestDefaultHystrixFallback_FallbackFunc(t *testing.T) {
	cb, _, _ := hystrix.GetCircuit("newCB")
	defaultFallback := &DefaultHystrixFallback{}
	result := defaultFallback.FallbackFunc(errors.Errorf("error"), &dubbo.DubboInvoker{}, nil, *cb)
	assert.NotNil(t, result)
	assert.Error(t, result.Error())

}

type (
	User struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
)

type testMockSuccessInvoker struct {
	protocol.BaseInvoker
}

func (iv *testMockSuccessInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	return &protocol.RPCResult{
		Rest: "Sucess",
		Err:  nil,
	}
}

type testMockFailInvoker struct {
	protocol.BaseInvoker
}

func (iv *testMockFailInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	return &protocol.RPCResult{
		Err: errors.Errorf("Error"),
	}
}

func TestHystrixFilter_Invoke_Success(t *testing.T) {
	hf := &HystrixFilter{&DefaultHystrixFallback{}}
	result := hf.Invoke(&testMockSuccessInvoker{}, &invocation.RPCInvocation{})
	assert.NotNil(t, result)
	assert.NoError(t, result.Error())
	assert.NotNil(t, result.Result())
}

func TestHystrixFilter_Invoke_Fail(t *testing.T) {
	hf := &HystrixFilter{&DefaultHystrixFallback{}}
	result := hf.Invoke(&testMockFailInvoker{}, &invocation.RPCInvocation{})
	assert.NotNil(t, result)
	assert.Error(t, result.Error())
}

type testHystrixFallback struct {
}

func (d *testHystrixFallback) FallbackFunc(err error, invoker protocol.Invoker, invocation protocol.Invocation, cb hystrix.CircuitBreaker) protocol.Result {
	if cb.IsOpen() {
		return &protocol.RPCResult{
			//For the request is blocked due to the circuit breaker is open
			Rest: true,
		}
	} else {
		return &protocol.RPCResult{
			//Circuit breaker not open
			Rest: false,
		}
	}
}

func TestHystricFilter_Invoke_CircuitBreak(t *testing.T) {
	hf := &HystrixFilter{&testHystrixFallback{}}
	resChan := make(chan protocol.Result, 50)
	for i := 0; i < 50; i++ {
		go func() {
			result := hf.Invoke(&testMockFailInvoker{}, &invocation.RPCInvocation{})
			resChan <- result
		}()
	}
	time.Sleep(time.Second * 6)
	var lastRest bool
	for i := 0; i < 50; i++ {
		lastRest = (<-resChan).Result().(bool)
	}
	//Normally the last result should be true, which means the circuit has been opened
	assert.True(t, lastRest)

}

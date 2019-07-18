package impl
import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)




func TestInitHystrixConfig(t *testing.T) {
	//Use the example config file
	config.Load()
	err := initHystrixConfig()
	assert.Nil(t,err,err)
	assert.NotNil(t,conf,"Conf should not be nil")
	assert.Equal(t,"Default",conf.Default)
	configsIn:=conf.Configs["Default"]
	assert.NotNil(t,configsIn,"Configs should not be nil")
	assert.Equal(t,20,configsIn.RequestVolumeThreshold)
	assert.Equal(t,configsIn.ErrorPercentThreshold,50)
	assert.Equal(t,5000,configsIn.SleepWindow)
	assert.Equal(t,1000,configsIn.Timeout)
	serviceConfigIn:=conf.Services["com.ikurento.user.UserProvider"]
	assert.NotNil(t,serviceConfigIn,"Service configs should not be nil")
	assert.Equal(t,"userp",serviceConfigIn.ServiceConfig)
	assert.NotNil(t,serviceConfigIn.Methods,"Method configs should not be nil")
	assert.Equal(t,"userp_m",serviceConfigIn.Methods["GetUser"])

}



type MockFallback struct {

}
func (m *MockFallback) FallbackFunc(err error, invoker protocol.Invoker, invocation protocol.Invocation, cb hystrix.CircuitBreaker) protocol.Result{
	return &protocol.RPCResult{Rest:"MOCK"}
}

func TestRefreshHystrix(t *testing.T) {

}

func TestGetHystrixFilter(t *testing.T) {
	filterGot:=GetHystrixFilter()
	assert.NotNil(t,filterGot)
}

func TestGetConfig_1(t *testing.T) {
	configGot:=getConfig("com.ikurento.user.UserProvider","GetUser")
	assert.NotNil(t,configGot)
	assert.Equal(t,1200,configGot.Timeout)
	assert.Equal(t,12,configGot.MaxConcurrentRequests)
	assert.Equal(t,6000,configGot.SleepWindow)
	assert.Equal(t,60,configGot.ErrorPercentThreshold)
	assert.Equal(t,5,configGot.RequestVolumeThreshold)
	assert.Equal(t,"exampleFallback",configGot.Fallback)
}

func TestGetConfig_2(t *testing.T){
	configGot:=getConfig("com.ikurento.user.UserProvider","GetUser0")
	assert.NotNil(t,configGot)
	assert.Equal(t,800,configGot.Timeout)
	assert.Equal(t,8,configGot.MaxConcurrentRequests)
	assert.Equal(t,4,configGot.SleepWindow)
	assert.Equal(t,45,configGot.ErrorPercentThreshold)
	assert.Equal(t,15,configGot.RequestVolumeThreshold)
	assert.Equal(t,"",configGot.Fallback)
}

func TestGetConfig_3(t *testing.T){
	configGot:= getConfig("Mock.Service","GetMock")
	assert.NotNil(t,configGot)
	assert.Equal(t,1000,configGot.Timeout)
	assert.Equal(t,10,configGot.MaxConcurrentRequests)
	assert.Equal(t,5000,configGot.SleepWindow)
	assert.Equal(t,50,configGot.ErrorPercentThreshold)
	assert.Equal(t,20,configGot.RequestVolumeThreshold)
	assert.Equal(t,"",configGot.Fallback)
}

func TestGetHystrixFallback(t *testing.T) {
	fallback["mock"]=&MockFallback{}
	fallbackGot:=getHystrixFallback("mock")
	assert.NotNil(t,fallbackGot)
}


func TestDefaultHystrixFallback_FallbackFunc(t *testing.T) {

}

func TestHystrixFilter_Invoke(t *testing.T) {



}



package impl

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
)


const (
	HYSTRIX="hystrix"
	TIMEOUT_KEY="timeout"
	MAXCONCURRENTREQUESTS_KEY="maxconcurrentrequests"
	SLEEPWINDOW_KEY="sleepwindow"
	ERRORPERCENTTHRESHOLD_KEY="errorpercentthreshold"
	CONF_HYSTRIXFILTER_FILE_PATH="CONF_HYSTRIXFILTER_FILE_PATH"
)


var (
	//Timeout
	//MaxConcurrentRequests
	//RequestVolumeThreshold
	//SleepWindow
	//ErrorPercentThreshold
	isConfigLoaded = false

	//
	methodLevelConfigMap = make(map[string]hystrix.CommandConfig)
	serviceLevelConfigMap = make(map[string]hystrix.CommandConfig)
	defaultConfig hystrix.CommandConfig


)

func init(){
	extension.SetFilter(HYSTRIX,GetHystrixFilter)
}
type HystrixFilter struct {

}

func (hf *HystrixFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result{

	cmdName := fmt.Sprintf("%s&method=%s",invoker.GetUrl().Key(),invocation.MethodName())

	_,ifNew,err := hystrix.GetCircuit(cmdName)
	if err != nil{
		logger.Errorf("[Hystrix Filter]Errors occurred getting circuit for %s , will invoke without hystrix, error is: ",cmdName,err)
		return invoker.Invoke(invocation)
	}

	// Do the configuration if the circuit breaker is created for the first time
	if ifNew {
		hystrix.ConfigureCommand(cmdName,hystrix.CommandConfig{

		})
	}

	logger.Infof("[Hystrix Filter]Using hystrix filter: %s",cmdName)
	var result protocol.Result
	_ = hystrix.Do(cmdName, func() error {
		result = invoker.Invoke(invocation)
		return nil
	}, func(err error) error {

		//failure logic
		result = &protocol.RPCResult{}
		result.SetError(err)
		return nil
		//failure logic

	})
	return result
}

func (hf *HystrixFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result{
	return result
}
func GetHystrixFilter() filter.Filter{
	//When first called, load the config in
	if !isConfigLoaded{
		if err:=initHystrixConfig();err!=nil{
			logger.Warnf("[Hystrix Filter]Config load failed, error is: %v , will use default",err)
		}

		isConfigLoaded=true
	}


	return &HystrixFilter{}
}





type HystrixFilterConfig struct {
	Configs map[string] hystrix.CommandConfig
	Default string
	Services map[string] ServiceHystrixConfig
}
type ServiceHystrixConfig struct{
	ServiceConfig string	`yaml:"service_config,omitempty"`
	Methods map[string]string
}
func initHystrixConfig() error{
	confHystrixFile := os.Getenv(CONF_HYSTRIXFILTER_FILE_PATH)
	if confHystrixFile==""{
		return perrors.Errorf("hystrix filter config file is nil")
	}
	if path.Ext(confHystrixFile) != ".yml"{
		return perrors.Errorf("hystrix filter config file suffix must be .yml")
	}
	confStream, err := ioutil.ReadFile(confHystrixFile)
	if err != nil {
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", confHystrixFile, perrors.WithStack(err))
	}
	hystrixConfig:=&HystrixFilterConfig{}
	if err = yaml.Unmarshal(confStream,hystrixConfig);err!=nil{
		return perrors.Errorf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
	}
	return nil
}


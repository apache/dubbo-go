package impl

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"strings"
)

const (
	GENERIC_SERVICE               = "generic-service"
	GENERIC_SERIALIZATION_DEFAULT = "true"
)

func init() {
	extension.SetFilter(GENERIC_SERVICE, GetGenericServiceFilter)
}

type GenericServiceFilter struct{}

func (ef *GenericServiceFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking generic service filter.")
	logger.Debugf("generic service filter methodName:%v,args:%v", invocation.MethodName(), len(invocation.Arguments()))
	if invocation.MethodName() != constant.GENERIC || len(invocation.Arguments()) != 3 {
		return invoker.Invoke(invocation)
	}
	var (
		err        error
		methodName string
		newParams  []interface{}
		genericKey string
		argsType   []reflect.Type
		oldParams  []hessian.Object
	)
	url := invoker.GetUrl()
	methodName = invocation.Arguments()[0].(string)
	// get service
	svc := common.ServiceMap.GetService(url.Protocol, strings.TrimPrefix(url.Path, "/"))
	// get method
	method := svc.Method()[methodName]
	if method == nil {
		logger.Errorf("[Generic Service Filter] Don't have this method: %v", method)
		return &protocol.RPCResult{}
	}
	argsType = method.ArgsType()
	genericKey = invocation.AttachmentsByKey(constant.GENERIC_KEY, GENERIC_SERIALIZATION_DEFAULT)
	if genericKey == GENERIC_SERIALIZATION_DEFAULT {
		oldParams = invocation.Arguments()[2].([]hessian.Object)
	} else {
		logger.Errorf("[Generic Service Filter] Don't support this generic: %v", genericKey)
		return &protocol.RPCResult{}
	}
	if len(oldParams) != len(argsType) {
		logger.Errorf("[Generic Service Filter] method:%s invocation arguments number was wrong", methodName)
		return &protocol.RPCResult{}
	}
	// oldParams convert to newParams
	for i := range argsType {
		var newParam interface{}
		if argsType[i].Kind() == reflect.Ptr {
			newParam = reflect.New(argsType[i].Elem()).Interface()
			err = mapstructure.Decode(oldParams[i], newParam)
		} else if argsType[i].Kind() == reflect.Struct || argsType[i].Kind() == reflect.Slice {
			newParam = reflect.New(argsType[i]).Interface()
			err = mapstructure.Decode(oldParams[i], newParam)
			newParam = reflect.ValueOf(newParam).Elem().Interface()
		} else {
			newParam = oldParams[i]
		}
		if err != nil {
			logger.Errorf("[Generic Service Filter] decode arguments map to struct wrong")
		}
		newParams = append(newParams, newParam)
	}
	newInvocation := invocation2.NewRPCInvocation(methodName, newParams, invocation.Attachments())
	newInvocation.SetReply(invocation.Reply())
	return invoker.Invoke(newInvocation)
}

func (ef *GenericServiceFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() == constant.GENERIC && len(invocation.Arguments()) == 3 && result.Result() != nil {
		s := reflect.ValueOf(result.Result()).Elem().Interface()
		r := struct2MapAll(s)
		result.SetResult(r)
	}
	return result
}

func GetGenericServiceFilter() filter.Filter {
	return &GenericServiceFilter{}
}

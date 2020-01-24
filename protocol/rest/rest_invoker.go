package rest

import (
	"context"
	"fmt"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
)

type RestInvoker struct {
	protocol.BaseInvoker
	client              rest_interface.RestClient
	restMethodConfigMap map[string]*rest_interface.RestMethodConfig
}

func NewRestInvoker(url common.URL, client *rest_interface.RestClient, restMethodConfig map[string]*rest_interface.RestMethodConfig) *RestInvoker {
	return &RestInvoker{
		BaseInvoker:         *protocol.NewBaseInvoker(url),
		client:              *client,
		restMethodConfigMap: restMethodConfig,
	}
}

func (ri *RestInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	inv := invocation.(*invocation_impl.RPCInvocation)
	methodConfig := ri.restMethodConfigMap[inv.MethodName()]
	var result protocol.RPCResult
	if methodConfig == nil {
		logger.Errorf("[RestInvoker]Rest methodConfig:%s is nil", inv.MethodName())
		return nil
	}
	pathParams := restStringMapTransform(methodConfig.PathParamsMap, inv.Arguments())
	queryParams := restStringMapTransform(methodConfig.QueryParamsMap, inv.Arguments())
	headers := restStringMapTransform(methodConfig.HeadersMap, inv.Arguments())
	bodyParams := restInterfaceMapTransform(methodConfig.BodyMap, inv.Arguments())
	req := &rest_interface.RestRequest{
		Location:    ri.GetUrl().Location,
		Produces:    methodConfig.Produces,
		Consumes:    methodConfig.Consumes,
		Method:      methodConfig.MethodType,
		Path:        methodConfig.Path,
		PathParams:  pathParams,
		QueryParams: queryParams,
		Body:        bodyParams,
		Headers:     headers,
	}
	result.Err = ri.client.Do(req, inv.Reply())
	if result.Err == nil {
		result.Rest = inv.Reply()
	}
	return &result
}

func restStringMapTransform(paramsMap map[int]string, args []interface{}) map[string]string {
	resMap := make(map[string]string, len(paramsMap))
	for key, value := range paramsMap {
		resMap[value] = fmt.Sprintf("%v", args[key])
	}
	return resMap
}

func restInterfaceMapTransform(paramsMap map[int]string, args []interface{}) map[string]interface{} {
	resMap := make(map[string]interface{}, len(paramsMap))
	for key, value := range paramsMap {
		resMap[value] = args[key]
	}
	return resMap
}

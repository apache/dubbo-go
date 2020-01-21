package rest

import (
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

func NewRestInvoker(url common.URL, client rest_interface.RestClient, restMethodConfig map[string]*rest_interface.RestMethodConfig) *RestInvoker {
	return &RestInvoker{
		BaseInvoker:         *protocol.NewBaseInvoker(url),
		client:              client,
		restMethodConfigMap: restMethodConfig,
	}
}

func (ri *RestInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	inv := invocation.(*invocation_impl.RPCInvocation)
	methodConfig := ri.restMethodConfigMap[inv.MethodName()]
	var result protocol.RPCResult
	if methodConfig == nil {
		logger.Errorf("[RestInvoker]Rest methodConfig:%s is nil", inv.MethodName())
		return nil
	}
	pathParams := make(map[string]string)
	queryParams := make(map[string]string)
	bodyParams := make(map[string]interface{})
	for key, value := range methodConfig.PathParamsMap {
		pathParams[value] = fmt.Sprintf("%v", inv.Arguments()[key])
	}
	for key, value := range methodConfig.QueryParamsMap {
		queryParams[value] = fmt.Sprintf("%v", inv.Arguments()[key])
	}
	for key, value := range methodConfig.BodyMap {
		bodyParams[value] = inv.Arguments()[key]
	}
	req := &rest_interface.RestRequest{
		Location:    ri.GetUrl().Location,
		Produces:    methodConfig.Produces,
		Consumes:    methodConfig.Consumes,
		Method:      methodConfig.MethodType,
		Path:        methodConfig.Path,
		PathParams:  pathParams,
		QueryParams: queryParams,
		Body:        bodyParams,
	}
	result.Err = ri.client.Do(req, inv.Reply())
	if result.Err == nil {
		result.Rest = inv.Reply()
	}
	return &result

}

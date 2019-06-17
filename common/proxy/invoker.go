package proxy

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

type CallInvoker struct {
	protocol.BaseInvoker
}

func NewCallInvoker(url common.URL) *CallInvoker {
	return &CallInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
	}
}

func (bi *CallInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	rpcResult := &protocol.RPCResult{Attrs: invocation.Attachments()}
	//todo:call service

	// get method
	//proto := bi.GetUrl().Protocol
	//if bi.GetUrl().SubURL != nil {
	//	proto = bi.GetUrl().SubURL.Protocol
	//}
	//svc := common.ServiceMap.GetService(proto, serviceName)
	//if svc == nil {
	//	return perrors.New("cannot find svc " + serviceName)
	//}
	//method := svc.Method()[methodName]
	//if method == nil {
	//	logger.Errorf("method not found!")
	//	rpcResult.Err = perrors.New("cannot find method " + methodName + " of svc " + serviceName)
	//	return rpcResult
	//}
	//
	//returnValues := method.Method().Func.Call(in)
	return rpcResult
}

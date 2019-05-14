package invocation

import (
	"reflect"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

/////////////////////////////
// Invocation Impletment of RPC
/////////////////////////////
// todo: is it necessary to separate fields of consumer(provider) from RPCInvocation
type RPCInvocation struct {
	methodName     string
	parameterTypes []reflect.Type
	arguments      []interface{}
	reply          interface{}
	callBack       interface{}
	attachments    map[string]string
	invoker        protocol.Invoker
}

func NewRPCInvocationForConsumer(methodName string, parameterTypes []reflect.Type, arguments []interface{},
	reply interface{}, callBack interface{}, url common.URL, invoker protocol.Invoker) *RPCInvocation {

	attachments := map[string]string{}
	attachments[constant.PATH_KEY] = url.Path
	attachments[constant.GROUP_KEY] = url.GetParam(constant.GROUP_KEY, "")
	attachments[constant.INTERFACE_KEY] = url.GetParam(constant.INTERFACE_KEY, "")
	attachments[constant.VERSION_KEY] = url.GetParam(constant.VERSION_KEY, constant.DEFAULT_VERSION)

	return &RPCInvocation{
		methodName:     methodName,
		parameterTypes: parameterTypes,
		arguments:      arguments,
		reply:          reply,
		callBack:       callBack,
		attachments:    attachments,
		invoker:        invoker,
	}
}

func NewRPCInvocationForProvider(methodName string, arguments []interface{}, attachments map[string]string) *RPCInvocation {
	return &RPCInvocation{
		methodName:  methodName,
		arguments:   arguments,
		attachments: attachments,
	}
}

func (r *RPCInvocation) MethodName() string {
	return r.methodName
}

func (r *RPCInvocation) ParameterTypes() []reflect.Type {
	return r.parameterTypes
}

func (r *RPCInvocation) Arguments() []interface{} {
	return r.arguments
}

func (r *RPCInvocation) Reply() interface{} {
	return r.reply
}

func (r *RPCInvocation) Attachments() map[string]string {
	return r.attachments
}

func (r *RPCInvocation) AttachmentsByKey(key string, defaultValue string) string {
	if r.attachments == nil {
		return defaultValue
	}
	value, ok := r.attachments[key]
	if ok {
		return value
	}
	return defaultValue
}

func (r *RPCInvocation) SetAttachments(key string, value string) {
	if r.attachments == nil {
		r.attachments = make(map[string]string)
	}
	r.attachments[key] = value
}

func (r *RPCInvocation) Invoker() protocol.Invoker {
	return r.invoker
}

func (r *RPCInvocation) SetInvoker() protocol.Invoker {
	return r.invoker
}

func (r *RPCInvocation) CallBack() interface{} {
	return r.callBack
}

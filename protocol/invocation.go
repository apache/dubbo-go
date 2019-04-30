package protocol

import (
	"reflect"
)

type Invocation interface {
	MethodName() string
	ParameterTypes() []reflect.Type
	Arguments() []interface{}
	Reply() interface{}
	Attachments() map[string]string
	AttachmentsByKey(string, string) string
	Invoker() Invoker
}

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
	invoker        Invoker
}

func NewRPCInvocationForConsumer(methodName string, parameterTypes []reflect.Type, arguments []interface{},
	reply interface{}, callBack interface{}, attachments map[string]string, invoker Invoker) *RPCInvocation {
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

func NewRPCInvocationForProvider(attachments map[string]string) *RPCInvocation {
	return &RPCInvocation{
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

func (r *RPCInvocation) Invoker() Invoker {
	return r.invoker
}

// SetInvoker is called while getting url, maybe clusterInvoker?
func (r *RPCInvocation) SetInvoker() Invoker {
	return r.invoker
}

func (r *RPCInvocation) CallBack() interface{} {
	return r.callBack
}

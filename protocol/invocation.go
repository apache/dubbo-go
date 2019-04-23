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

type RPCInvocation struct {
	methodName     string
	parameterTypes []reflect.Type
	arguments      []interface{}
	reply          interface{}
	attachments    map[string]string
	invoker        Invoker
	params         map[string]interface{} // Store some parameters that are not easy to refine
}

func NewRPCInvocation(methodName string, parameterTypes []reflect.Type, arguments []interface{},
	reply interface{}, attachments map[string]string, invoker Invoker, params map[string]interface{}) *RPCInvocation {
	return &RPCInvocation{
		methodName:     methodName,
		parameterTypes: parameterTypes,
		arguments:      arguments,
		reply:          reply,
		attachments:    attachments,
		invoker:        invoker,
		params:         params,
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
	value, ok := r.attachments[key]
	if ok {
		return value
	}
	return defaultValue
}

func (r *RPCInvocation) Invoker() Invoker {
	return r.invoker
}
func (r *RPCInvocation) SetInvoker() Invoker {
	return r.invoker
}

func (r *RPCInvocation) Params() map[string]interface{} {
	return r.params
}

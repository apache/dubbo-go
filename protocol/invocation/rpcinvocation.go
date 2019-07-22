/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package invocation

import (
	"reflect"
)

import (
	"github.com/apache/dubbo-go/protocol"
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

func NewRPCInvocation(methodName string, arguments []interface{}, attachments map[string]string) *RPCInvocation {
	return &RPCInvocation{
		methodName:  methodName,
		arguments:   arguments,
		attachments: attachments,
	}
}

func NewRPCInvocationWithOptions(opts ...option) *RPCInvocation {
	invo := &RPCInvocation{}
	for _, opt := range opts {
		opt(invo)
	}
	return invo
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

func (r *RPCInvocation) SetReply(reply interface{}) {
	r.reply = reply
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

func (r *RPCInvocation) SetCallBack(c interface{}) {
	r.callBack = c
}

///////////////////////////
// option
///////////////////////////

type option func(invo *RPCInvocation)

func WithMethodName(methodName string) option {
	return func(invo *RPCInvocation) {
		invo.methodName = methodName
	}
}

func WithParameterTypes(parameterTypes []reflect.Type) option {
	return func(invo *RPCInvocation) {
		invo.parameterTypes = parameterTypes
	}
}

func WithArguments(arguments []interface{}) option {
	return func(invo *RPCInvocation) {
		invo.arguments = arguments
	}
}

func WithReply(reply interface{}) option {
	return func(invo *RPCInvocation) {
		invo.reply = reply
	}
}

func WithCallBack(callBack interface{}) option {
	return func(invo *RPCInvocation) {
		invo.callBack = callBack
	}
}

func WithAttachments(attachments map[string]string) option {
	return func(invo *RPCInvocation) {
		invo.attachments = attachments
	}
}

func WithInvoker(invoker protocol.Invoker) option {
	return func(invo *RPCInvocation) {
		invo.invoker = invoker
	}
}

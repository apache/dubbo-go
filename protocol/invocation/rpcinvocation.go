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
	"sync"
)

import (
	"github.com/apache/dubbo-go/protocol"
)

// ///////////////////////////
// Invocation Implement of RPC
// ///////////////////////////

// todo: is it necessary to separate fields of consumer(provider) from RPCInvocation
// nolint
type RPCInvocation struct {
	methodName      string
	parameterTypes  []reflect.Type
	parameterValues []reflect.Value
	arguments       []interface{}
	reply           interface{}
	callBack        interface{}
	attachments     map[string]string
	// Refer to dubbo 2.7.6.  It is different from attachment. It is used in internal process.
	attributes map[string]interface{}
	invoker    protocol.Invoker
	lock       sync.RWMutex
}

// NewRPCInvocation creates a RPC invocation.
func NewRPCInvocation(methodName string, arguments []interface{}, attachments map[string]string) *RPCInvocation {
	return &RPCInvocation{
		methodName:  methodName,
		arguments:   arguments,
		attachments: attachments,
		attributes:  make(map[string]interface{}, 8),
	}
}

// NewRPCInvocationWithOptions creates a RPC invocation with @opts.
func NewRPCInvocationWithOptions(opts ...option) *RPCInvocation {
	invo := &RPCInvocation{}
	for _, opt := range opts {
		opt(invo)
	}
	if invo.attributes == nil {
		invo.attributes = make(map[string]interface{})
	}
	return invo
}

// MethodName gets RPC invocation method name.
func (r *RPCInvocation) MethodName() string {
	return r.methodName
}

// ParameterTypes gets RPC invocation parameter types.
func (r *RPCInvocation) ParameterTypes() []reflect.Type {
	return r.parameterTypes
}

// ParameterValues gets RPC invocation parameter values.
func (r *RPCInvocation) ParameterValues() []reflect.Value {
	return r.parameterValues
}

// Arguments gets RPC arguments.
func (r *RPCInvocation) Arguments() []interface{} {
	return r.arguments
}

// Reply gets response of RPC request.
func (r *RPCInvocation) Reply() interface{} {
	return r.reply
}

// SetReply sets response of RPC request.
func (r *RPCInvocation) SetReply(reply interface{}) {
	r.reply = reply
}

// Attachments gets all attachments of RPC.
func (r *RPCInvocation) Attachments() map[string]string {
	return r.attachments
}

// AttachmentsByKey gets RPC attachment by key, if nil then return default value.
func (r *RPCInvocation) AttachmentsByKey(key string, defaultValue string) string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.attachments == nil {
		return defaultValue
	}
	value, ok := r.attachments[key]
	if ok {
		return value
	}
	return defaultValue
}

// Attributes gets all attributes of RPC.
func (r *RPCInvocation) Attributes() map[string]interface{} {
	return r.attributes
}

// AttributeByKey gets attribute by @key. If it is not exist, it will return default value.
func (r *RPCInvocation) AttributeByKey(key string, defaultValue interface{}) interface{} {
	r.lock.RLock()
	defer r.lock.RUnlock()
	value, ok := r.attributes[key]
	if ok {
		return value
	}
	return defaultValue
}

// SetAttachments sets attribute by @key and @value.
func (r *RPCInvocation) SetAttachments(key string, value string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.attachments == nil {
		r.attachments = make(map[string]string)
	}
	r.attachments[key] = value
}

// SetAttribute sets attribute by @key and @value.
func (r *RPCInvocation) SetAttribute(key string, value interface{}) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.attributes[key] = value
}

// Invoker gets the invoker in current context.
func (r *RPCInvocation) Invoker() protocol.Invoker {
	return r.invoker
}

// nolint
func (r *RPCInvocation) SetInvoker(invoker protocol.Invoker) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.invoker = invoker
}

// CallBack sets RPC callback method.
func (r *RPCInvocation) CallBack() interface{} {
	return r.callBack
}

// SetCallBack sets RPC callback method.
func (r *RPCInvocation) SetCallBack(c interface{}) {
	r.callBack = c
}

// /////////////////////////
// option
// /////////////////////////

type option func(invo *RPCInvocation)

// WithMethodName creates option with @methodName.
func WithMethodName(methodName string) option {
	return func(invo *RPCInvocation) {
		invo.methodName = methodName
	}
}

// WithParameterTypes creates option with @parameterTypes.
func WithParameterTypes(parameterTypes []reflect.Type) option {
	return func(invo *RPCInvocation) {
		invo.parameterTypes = parameterTypes
	}
}

// WithParameterValues creates option with @parameterValues
func WithParameterValues(parameterValues []reflect.Value) option {
	return func(invo *RPCInvocation) {
		invo.parameterValues = parameterValues
	}
}

// WithArguments creates option with @arguments function.
func WithArguments(arguments []interface{}) option {
	return func(invo *RPCInvocation) {
		invo.arguments = arguments
	}
}

// WithReply creates option with @reply function.
func WithReply(reply interface{}) option {
	return func(invo *RPCInvocation) {
		invo.reply = reply
	}
}

// WithCallBack creates option with @callback function.
func WithCallBack(callBack interface{}) option {
	return func(invo *RPCInvocation) {
		invo.callBack = callBack
	}
}

// WithAttachments creates option with @attachments.
func WithAttachments(attachments map[string]string) option {
	return func(invo *RPCInvocation) {
		invo.attachments = attachments
	}
}

// WithInvoker creates option with @invoker.
func WithInvoker(invoker protocol.Invoker) option {
	return func(invo *RPCInvocation) {
		invo.invoker = invoker
	}
}

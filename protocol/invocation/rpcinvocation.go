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
	"context"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

var _ protocol.Invocation = (*RPCInvocation)(nil)

// todo: is it necessary to separate fields of consumer(provider) from RPCInvocation
// nolint
type RPCInvocation struct {
	methodName string
	// Parameter Type Names. It is used to specify the parameterType
	parameterTypeNames []string
	parameterTypes     []reflect.Type

	parameterValues    []reflect.Value
	parameterRawValues []any
	arguments          []any
	reply              any
	callBack           any
	attachments        map[string]any
	// Refer to dubbo 2.7.6.  It is different from attachment. It is used in internal process.
	attributes map[string]any
	invoker    protocol.Invoker
	lock       sync.RWMutex
}

// NewRPCInvocation creates a RPC invocation.
func NewRPCInvocation(methodName string, arguments []any, attachments map[string]any) *RPCInvocation {
	return &RPCInvocation{
		methodName:  methodName,
		arguments:   arguments,
		attachments: attachments,
		attributes:  make(map[string]any, 8),
	}
}

// NewRPCInvocationWithOptions creates a RPC invocation with @opts.
func NewRPCInvocationWithOptions(opts ...option) *RPCInvocation {
	invo := &RPCInvocation{}
	for _, opt := range opts {
		opt(invo)
	}
	if invo.attributes == nil {
		invo.attributes = make(map[string]any)
	}
	return invo
}

// MethodName gets RPC invocation method name.
func (r *RPCInvocation) MethodName() string {
	return r.methodName
}

// ActualMethodName gets actual invocation method name. It returns the method name been called if it's a generic call
func (r *RPCInvocation) ActualMethodName() string {
	if r.IsGenericInvocation() {
		return r.Arguments()[0].(string)
	} else {
		return r.MethodName()
	}
}

// IsGenericInvocation gets if this is a generic invocation
func (r *RPCInvocation) IsGenericInvocation() bool {
	return (r.MethodName() == constant.Generic ||
		r.MethodName() == constant.GenericAsync) &&
		r.Arguments() != nil &&
		len(r.Arguments()) == 3
}

// ParameterTypes gets RPC invocation parameter types.
func (r *RPCInvocation) ParameterTypes() []reflect.Type {
	return r.parameterTypes
}

// ParameterTypeNames gets RPC invocation parameter types of string expression.
func (r *RPCInvocation) ParameterTypeNames() []string {
	return r.parameterTypeNames
}

// ParameterValues gets RPC invocation parameter values.
func (r *RPCInvocation) ParameterValues() []reflect.Value {
	return r.parameterValues
}

func (r *RPCInvocation) ParameterRawValues() []any {
	return r.parameterRawValues
}

// Arguments gets RPC arguments.
func (r *RPCInvocation) Arguments() []any {
	return r.arguments
}

// Reply gets response of RPC request.
func (r *RPCInvocation) Reply() any {
	return r.reply
}

// SetReply sets response of RPC request.
func (r *RPCInvocation) SetReply(reply any) {
	r.reply = reply
}

// Attachments gets all attachments of RPC.
func (r *RPCInvocation) Attachments() map[string]any {
	return r.attachments
}

// GetAttachmentInterface returns the corresponding value from dubbo's attachment with the given key.
func (r *RPCInvocation) GetAttachmentInterface(key string) any {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.attachments == nil {
		return nil
	}
	return r.attachments[key]
}

// Attributes gets all attributes of RPC.
func (r *RPCInvocation) Attributes() map[string]any {
	return r.attributes
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
func (r *RPCInvocation) CallBack() any {
	return r.callBack
}

// SetCallBack sets RPC callback method.
func (r *RPCInvocation) SetCallBack(c any) {
	r.callBack = c
}

func (r *RPCInvocation) ServiceKey() string {
	return common.ServiceKey(strings.TrimPrefix(r.GetAttachmentWithDefaultValue(constant.PathKey, r.GetAttachmentWithDefaultValue(constant.InterfaceKey, "")), "/"),
		r.GetAttachmentWithDefaultValue(constant.GroupKey, ""), r.GetAttachmentWithDefaultValue(constant.VersionKey, ""))
}

func (r *RPCInvocation) SetAttachment(key string, value any) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.attachments == nil {
		r.attachments = make(map[string]any)
	}
	r.attachments[key] = value
}

func (r *RPCInvocation) GetAttachment(key string) (string, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.attachments == nil {
		return "", false
	}
	if value, existed := r.attachments[key]; existed {
		if str, strOK := value.(string); strOK {
			return str, true
		} else if strArr, strArrOK := value.([]string); strArrOK && len(strArr) > 0 {
			// For triple protocol, the attachment value is wrapped by an array.
			return strArr[0], true
		}
	}
	return "", false
}

func (r *RPCInvocation) GetAttachmentWithDefaultValue(key string, defaultValue string) string {
	if value, ok := r.GetAttachment(key); ok {
		return value
	}
	return defaultValue
}

func (r *RPCInvocation) SetAttribute(key string, value any) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.attributes == nil {
		r.attributes = make(map[string]any)
	}
	r.attributes[key] = value
}

func (r *RPCInvocation) GetAttribute(key string) (any, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.attributes == nil {
		return nil, false
	}
	value, ok := r.attributes[key]
	return value, ok
}

func (r *RPCInvocation) GetAttributeWithDefaultValue(key string, defaultValue any) any {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.attributes == nil {
		return defaultValue
	}
	if value, ok := r.attributes[key]; ok {
		return value
	}
	return defaultValue
}

func (r *RPCInvocation) GetAttachmentAsContext() context.Context {
	ctx := context.Background()
	var header = http.Header{}
	for k, v := range r.Attachments() {
		if str, ok := v.(string); ok {
			header.Set(k, str)
			continue
		}
		if str, ok := v.([]string); ok {
			for _, s := range str {
				header.Add(k, s)
			}
			continue
		}
	}
	return triple_protocol.NewOutgoingContext(ctx, header)
}

func (r *RPCInvocation) MergeAttachmentFromContext(ctx context.Context) {
	header := triple_protocol.ExtractFromOutgoingContext(ctx)
	if header == nil {
		return
	}
	for k, v := range header {
		if len(v) == 1 {
			r.SetAttachment(k, v[0])
		} else {
			r.SetAttachment(k, v)
		}
	}
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

// WithParameterTypeNames creates option with @parameterTypeNames.
func WithParameterTypeNames(parameterTypeNames []string) option {
	return func(invo *RPCInvocation) {
		if len(parameterTypeNames) == 0 {
			return
		}
		parameterTypeNamesTmp := make([]string, len(parameterTypeNames))
		copy(parameterTypeNamesTmp, parameterTypeNames)
		invo.parameterTypeNames = parameterTypeNamesTmp
	}
}

// WithParameterValues creates option with @parameterValues
func WithParameterValues(parameterValues []reflect.Value) option {
	return func(invo *RPCInvocation) {
		invo.parameterValues = parameterValues
	}
}

// WithParameterRawValues creates option with @parameterRawValues
func WithParameterRawValues(parameterRawValues []any) option {
	return func(invo *RPCInvocation) {
		invo.parameterRawValues = parameterRawValues
	}
}

// WithArguments creates option with @arguments function.
func WithArguments(arguments []any) option {
	return func(invo *RPCInvocation) {
		invo.arguments = arguments
	}
}

// WithReply creates option with @reply function.
func WithReply(reply any) option {
	return func(invo *RPCInvocation) {
		invo.reply = reply
	}
}

// WithCallBack creates option with @callback function.
func WithCallBack(callBack any) option {
	return func(invo *RPCInvocation) {
		invo.callBack = callBack
	}
}

// WithAttachments creates option with @attachments.
func WithAttachments(attachments map[string]any) option {
	return func(invo *RPCInvocation) {
		invo.attachments = attachments
	}
}

// WithAttachment put a key-value pair into attachments.
func WithAttachment(k string, v any) option {
	return func(invo *RPCInvocation) {
		if invo.attachments == nil {
			invo.attachments = make(map[string]any)
		}
		invo.attachments[k] = v
	}
}

// WithInvoker creates option with @invoker.
func WithInvoker(invoker protocol.Invoker) option {
	return func(invo *RPCInvocation) {
		invo.invoker = invoker
	}
}

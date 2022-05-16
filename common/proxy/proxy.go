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

package proxy

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

import (
	"github.com/apache/dubbo-go-hessian2/java_exception"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// nolint
type Proxy struct {
	rpc         common.RPCService
	invoke      protocol.Invoker
	callback    interface{}
	attachments map[string]string
	implement   ImplementFunc
	once        sync.Once
}

type (
	// ProxyOption a function to init Proxy with options
	ProxyOption func(p *Proxy)
	// ImplementFunc function for proxy impl of RPCService functions
	ImplementFunc func(p *Proxy, v common.RPCService)
)

var typError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()

// NewProxy create service proxy.
func NewProxy(invoke protocol.Invoker, callback interface{}, attachments map[string]string) *Proxy {
	return NewProxyWithOptions(invoke, callback, attachments,
		WithProxyImplementFunc(DefaultProxyImplementFunc))
}

// NewProxyWithOptions create service proxy with options.
func NewProxyWithOptions(invoke protocol.Invoker, callback interface{}, attachments map[string]string, opts ...ProxyOption) *Proxy {
	p := &Proxy{
		invoke:      invoke,
		callback:    callback,
		attachments: attachments,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithProxyImplementFunc an option function to setup proxy.ImplementFunc
func WithProxyImplementFunc(f ImplementFunc) ProxyOption {
	return func(p *Proxy) {
		p.implement = f
	}
}

// Implement
// proxy implement
// In consumer, RPCService like:
// 		type XxxProvider struct {
//  		Yyy func(ctx context.Context, args []interface{}, rsp *Zzz) error
// 		}
func (p *Proxy) Implement(v common.RPCService) {
	p.once.Do(func() {
		p.implement(p, v)
		p.rpc = v
	})
}

// Get gets rpc service instance.
func (p *Proxy) Get() common.RPCService {
	return p.rpc
}

// GetCallback gets callback.
func (p *Proxy) GetCallback() interface{} {
	return p.callback
}

// GetInvoker gets Invoker.
func (p *Proxy) GetInvoker() protocol.Invoker {
	return p.invoke
}

// DefaultProxyImplementFunc the default function for proxy impl
func DefaultProxyImplementFunc(p *Proxy, v common.RPCService) {
	// check parameters, incoming interface must be a elem's pointer.
	valueOf := reflect.ValueOf(v)

	valueOfElem := valueOf.Elem()

	makeDubboCallProxy := func(methodName string, outs []reflect.Type) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			var (
				err            error
				inv            *invocation_impl.RPCInvocation
				inIArr         []interface{}
				inVArr         []reflect.Value
				reply          reflect.Value
				replyEmptyFlag bool
			)
			if methodName == "Echo" {
				methodName = "$echo"
			}

			if len(outs) == 2 { // return (reply, error)
				if outs[0].Kind() == reflect.Ptr {
					reply = reflect.New(outs[0].Elem())
				} else {
					reply = reflect.New(outs[0])
				}
			} else { // only return error
				replyEmptyFlag = true
			}

			start := 0
			end := len(in)
			invCtx := context.Background()
			// retrieve the context from the first argument if existed
			if end > 0 {
				if in[0].Type().String() == "context.Context" {
					if !in[0].IsNil() {
						// the user declared context as method's parameter
						invCtx = in[0].Interface().(context.Context)
					}
					start += 1
				}
			}

			if end-start <= 0 {
				inIArr = []interface{}{}
				inVArr = []reflect.Value{}
			} else if v, ok := in[start].Interface().([]interface{}); ok && end-start == 1 {
				inIArr = v
				inVArr = []reflect.Value{in[start]}
			} else {
				inIArr = make([]interface{}, end-start)
				inVArr = make([]reflect.Value, end-start)
				index := 0
				for i := start; i < end; i++ {
					inIArr[index] = in[i].Interface()
					inVArr[index] = in[i]
					index++
				}
			}

			inv = invocation_impl.NewRPCInvocationWithOptions(invocation_impl.WithMethodName(methodName),
				invocation_impl.WithArguments(inIArr),
				//invocation_impl.WithReply(reply.Interface()),
				invocation_impl.WithCallBack(p.callback), invocation_impl.WithParameterValues(inVArr))
			if !replyEmptyFlag {
				inv.SetReply(reply.Interface())
			}

			for k, value := range p.attachments {
				inv.SetAttachment(k, value)
			}

			// add user setAttachment. It is compatibility with previous versions.
			atm := invCtx.Value(constant.AttachmentKey)
			if m, ok := atm.(map[string]string); ok {
				for k, value := range m {
					inv.SetAttachment(k, value)
				}
			} else if m2, ok2 := atm.(map[string]interface{}); ok2 {
				// it is support to transfer map[string]interface{}. It refers to dubbo-java 2.7.
				for k, value := range m2 {
					inv.SetAttachment(k, value)
				}
			}

			result := p.invoke.Invoke(invCtx, inv)
			err = result.Error()
			// cause is raw user level error
			cause := perrors.Cause(err)
			if err != nil {
				// if some error happened, it should be log some info in the separate file.
				if throwabler, ok := cause.(java_exception.Throwabler); ok {
					logger.Warnf("[CallProxy] invoke service throw exception: %v , stackTraceElements: %v", cause.Error(), throwabler.GetStackTrace())
				} else {
					// entire error is only for printing, do not return, because user would not want to deal with massive framework-level error message
					logger.Warnf("[CallProxy] received rpc err: %v", err)
				}
			} else {
				logger.Debugf("[CallProxy] received rpc result successfully: %s", result)
			}
			if len(outs) == 1 {
				return []reflect.Value{reflect.ValueOf(&cause).Elem()}
			}
			if len(outs) == 2 && outs[0].Kind() != reflect.Ptr {
				return []reflect.Value{reply.Elem(), reflect.ValueOf(&cause).Elem()}
			}
			return []reflect.Value{reply, reflect.ValueOf(&cause).Elem()}
		}
	}

	if err := refectAndMakeObjectFunc(valueOfElem, makeDubboCallProxy); err != nil {
		logger.Errorf("The type or combination type of RPCService %T must be a pointer of a struct. error is %s", v, err)
		return
	}
}

func refectAndMakeObjectFunc(valueOfElem reflect.Value, makeDubboCallProxy func(methodName string, outs []reflect.Type) func(in []reflect.Value) []reflect.Value) error {
	typeOf := valueOfElem.Type()
	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		return errors.New("invalid type kind")
	}
	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		methodName := t.Tag.Get("dubbo")
		if methodName == "" {
			methodName = t.Name
		}
		f := valueOfElem.Field(i)
		if f.Kind() == reflect.Func && f.IsValid() && f.CanSet() {
			outNum := t.Type.NumOut()

			if outNum != 1 && outNum != 2 {
				logger.Warnf("method %s of mtype %v has wrong number of in out parameters %d; needs exactly 1/2",
					t.Name, t.Type.String(), outNum)
				continue
			}

			// The latest return type of the method must be error.
			if returnType := t.Type.Out(outNum - 1); returnType != typError {
				logger.Warnf("the latest return type %s of method %q is not error", returnType, t.Name)
				continue
			}

			funcOuts := make([]reflect.Type, outNum)
			for i := 0; i < outNum; i++ {
				funcOuts[i] = t.Type.Out(i)
			}

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeDubboCallProxy(methodName, funcOuts)))
			logger.Debugf("set method [%s]", methodName)
		} else if f.IsValid() && f.CanSet() {
			// for struct combination
			valueOfSub := reflect.New(t.Type)
			valueOfElemInterface := valueOfSub.Elem()
			if valueOfElemInterface.Type().Kind() == reflect.Struct {
				if err := refectAndMakeObjectFunc(valueOfElemInterface, makeDubboCallProxy); err != nil {
					return err
				}
				f.Set(valueOfElemInterface)
			}
		}
	}
	return nil
}

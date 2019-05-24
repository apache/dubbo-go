// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"reflect"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	invocation_impl "github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
)

// Proxy struct
type Proxy struct {
	rpc         common.RPCService
	invoke      protocol.Invoker
	callBack    interface{}
	attachments map[string]string
}

var typError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()

func NewProxy(invoke protocol.Invoker, callBack interface{}, attachments map[string]string) *Proxy {
	return &Proxy{
		invoke:      invoke,
		callBack:    callBack,
		attachments: attachments,
	}
}

// proxy implement
// In consumer, RPCService like:
// 		type XxxProvider struct {
//  		Yyy func(ctx context.Context, args []interface{}, rsp *Zzz) error
// 		}
func (p *Proxy) Implement(v common.RPCService) {

	// check parameters, incoming interface must be a elem's pointer.
	valueOf := reflect.ValueOf(v)
	logger.Debugf("[Implement] reflect.TypeOf: %s", valueOf.String())

	valueOfElem := valueOf.Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		logger.Errorf("%s must be a struct ptr", valueOf.String())
		return
	}

	makeDubboCallProxy := func(methodName string, outs []reflect.Type) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			var (
				err   error
				inv   *invocation_impl.RPCInvocation
				inArr []interface{}
				reply reflect.Value
			)
			if methodName == "Echo" {
				methodName = "$echo"
			}

			start := 0
			end := len(in)
			if in[0].Type().String() == "context.Context" {
				start += 1
			}
			if len(outs) == 1 {
				end -= 1
				reply = in[len(in)-1]
			} else {
				if outs[0].Kind() == reflect.Ptr {
					reply = reflect.New(outs[0].Elem())
				} else {
					reply = reflect.New(outs[0])
				}
			}

			if v, ok := in[start].Interface().([]interface{}); ok && end-start == 1 {
				inArr = v
			} else {
				inArr = make([]interface{}, end-start)
				index := 0
				for i := start; i < end; i++ {
					inArr[index] = in[i].Interface()
					index++
				}
			}

			inv = invocation_impl.NewRPCInvocationForConsumer(methodName, nil, inArr, reply.Interface(), p.callBack, common.URL{}, nil)

			for k, value := range p.attachments {
				inv.SetAttachments(k, value)
			}

			result := p.invoke.Invoke(inv)

			err = result.Error()
			logger.Infof("[makeDubboCallProxy] result: %v, err: %v", result.Result(), err)
			if len(outs) == 1 {
				return []reflect.Value{reflect.ValueOf(&err).Elem()}
			}
			if len(outs) == 2 && outs[0].Kind() != reflect.Ptr {
				return []reflect.Value{reply.Elem(), reflect.ValueOf(&err).Elem()}
			}
			return []reflect.Value{reply, reflect.ValueOf(&err).Elem()}
		}
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
			inNum := t.Type.NumIn()
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

			// reply must be Ptr when outNum == 1
			if outNum == 1 && t.Type.In(inNum-1).Kind() != reflect.Ptr {
				logger.Warnf("reply type of method %q is not a pointer", t.Name)
				continue
			}

			var funcOuts = make([]reflect.Type, outNum)
			for i := 0; i < outNum; i++ {
				funcOuts[i] = t.Type.Out(i)
			}

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeDubboCallProxy(methodName, funcOuts)))
			logger.Debugf("set method [%s]", methodName)
		}
	}

	p.rpc = v

}

func (p *Proxy) Get() common.RPCService {
	return p.rpc
}

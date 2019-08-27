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

package common

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

// rpc service interface
type RPCService interface {
	Reference() string // rpc service id or reference id
}

// for lowercase func
// func MethodMapper() map[string][string] {
//     return map[string][string]{}
// }
const METHOD_MAPPER = "MethodMapper"

var (
	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()

	// todo: lowerecas?
	ServiceMap = &serviceMap{
		serviceMap: make(map[string]map[string]*Service),
	}
)

//////////////////////////
// info of method
//////////////////////////

type MethodType struct {
	method    reflect.Method
	ctxType   reflect.Type   // request context
	argsType  []reflect.Type // args except ctx, include replyType if existing
	replyType reflect.Type   // return value, otherwise it is nil
}

func (m *MethodType) Method() reflect.Method {
	return m.method
}
func (m *MethodType) CtxType() reflect.Type {
	return m.ctxType
}
func (m *MethodType) ArgsType() []reflect.Type {
	return m.argsType
}
func (m *MethodType) ReplyType() reflect.Type {
	return m.replyType
}
func (m *MethodType) SuiteContext(ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ctxType)
}

//////////////////////////
// info of service interface
//////////////////////////

type Service struct {
	name     string
	rcvr     reflect.Value
	rcvrType reflect.Type
	methods  map[string]*MethodType
}

func (s *Service) Method() map[string]*MethodType {
	return s.methods
}
func (s *Service) RcvrType() reflect.Type {
	return s.rcvrType
}
func (s *Service) Rcvr() reflect.Value {
	return s.rcvr
}

//////////////////////////
// serviceMap
//////////////////////////

type serviceMap struct {
	mutex      sync.RWMutex                   // protects the serviceMap
	serviceMap map[string]map[string]*Service // protocol -> service name -> service
}

func (sm *serviceMap) GetService(protocol, name string) *Service {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	if s, ok := sm.serviceMap[protocol]; ok {
		if srv, ok := s[name]; ok {
			return srv
		}
		return nil
	}
	return nil
}

func (sm *serviceMap) Register(protocol string, rcvr RPCService) (string, error) {
	if sm.serviceMap[protocol] == nil {
		sm.serviceMap[protocol] = make(map[string]*Service)
	}

	s := new(Service)
	s.rcvrType = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		s := "no service name for type " + s.rcvrType.String()
		logger.Errorf(s)
		return "", perrors.New(s)
	}
	if !isExported(sname) {
		s := "type " + sname + " is not exported"
		logger.Errorf(s)
		return "", perrors.New(s)
	}

	sname = rcvr.Reference()
	if server := sm.GetService(protocol, sname); server != nil {
		return "", perrors.New("service already defined: " + sname)
	}
	s.name = sname
	s.methods = make(map[string]*MethodType)

	// Install the methods
	methods := ""
	methods, s.methods = suitableMethods(s.rcvrType)

	if len(s.methods) == 0 {
		s := "type " + sname + " has no exported methods of suitable type"
		logger.Errorf(s)
		return "", perrors.New(s)
	}
	sm.mutex.Lock()
	sm.serviceMap[protocol][s.name] = s
	sm.mutex.Unlock()

	return strings.TrimSuffix(methods, ","), nil
}

func (sm *serviceMap) UnRegister(protocol, serviceId string) error {
	if protocol == "" || serviceId == "" {
		return perrors.New("protocol or serviceName is nil")
	}
	sm.mutex.RLock()
	svcs, ok := sm.serviceMap[protocol]
	if !ok {
		sm.mutex.RUnlock()
		return perrors.New("no services for " + protocol)
	}
	_, ok = svcs[serviceId]
	if !ok {
		sm.mutex.RUnlock()
		return perrors.New("no service for " + serviceId)
	}
	sm.mutex.RUnlock()

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(svcs, serviceId)
	delete(sm.serviceMap, protocol)

	return nil
}

// Is this an exported - upper case - name
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// suitableMethods returns suitable Rpc methods of typ
func suitableMethods(typ reflect.Type) (string, map[string]*MethodType) {
	methods := make(map[string]*MethodType)
	var mts []string
	logger.Debugf("[%s] NumMethod is %d", typ.String(), typ.NumMethod())
	method, ok := typ.MethodByName(METHOD_MAPPER)
	var methodMapper map[string]string
	if ok && method.Type.NumIn() == 1 && method.Type.NumOut() == 1 && method.Type.Out(0).String() == "map[string]string" {
		methodMapper = method.Func.Call([]reflect.Value{reflect.New(typ.Elem())})[0].Interface().(map[string]string)
	}

	for m := 0; m < typ.NumMethod(); m++ {
		method = typ.Method(m)
		if mt := suiteMethod(method); mt != nil {
			methodName, ok := methodMapper[method.Name]
			if !ok {
				methodName = method.Name
			}
			methods[methodName] = mt
			mts = append(mts, methodName)
		}
	}
	return strings.Join(mts, ","), methods
}

// suiteMethod returns a suitable Rpc methodType
func suiteMethod(method reflect.Method) *MethodType {
	mtype := method.Type
	mname := method.Name
	inNum := mtype.NumIn()
	outNum := mtype.NumOut()

	// Method must be exported.
	if method.PkgPath != "" {
		return nil
	}

	var (
		replyType, ctxType reflect.Type
		argsType           []reflect.Type
	)

	if outNum != 1 && outNum != 2 {
		logger.Warnf("method %s of mtype %v has wrong number of in out parameters %d; needs exactly 1/2",
			mname, mtype.String(), outNum)
		return nil
	}

	// The latest return type of the method must be error.
	if returnType := mtype.Out(outNum - 1); returnType != typeOfError {
		logger.Warnf("the latest return type %s of method %q is not error", returnType, mname)
		return nil
	}

	// replyType
	if outNum == 2 {
		replyType = mtype.Out(0)
		if !isExportedOrBuiltinType(replyType) {
			logger.Errorf("reply type of method %s not exported{%v}", mname, replyType)
			return nil
		}
	}

	index := 1

	// ctxType
	if inNum > 1 && mtype.In(1).String() == "context.Context" {
		ctxType = mtype.In(1)
		index = 2
	}

	for ; index < inNum; index++ {
		argsType = append(argsType, mtype.In(index))
		// need not be a pointer.
		if !isExportedOrBuiltinType(mtype.In(index)) {
			logger.Errorf("argument type of method %q is not exported %v", mname, mtype.In(index))
			return nil
		}
	}

	return &MethodType{method: method, argsType: argsType, replyType: replyType, ctxType: ctxType}
}

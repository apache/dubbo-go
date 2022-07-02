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
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

// RPCService the type alias of interface{}
type RPCService = interface{}

// ReferencedRPCService is the rpc service interface which wraps base Reference method.
//
// Reference method refers rpc service id or reference id.
type ReferencedRPCService interface {
	Reference() string
}

// TriplePBService is  the type alias of interface{}
type TriplePBService interface {
	XXX_InterfaceName() string
}

// GetReference return the reference id of the service.
// If the service implemented the ReferencedRPCService interface,
// it will call the Reference method. If not, it will
// return the struct name as the reference id.
func GetReference(service RPCService) string {
	if s, ok := service.(ReferencedRPCService); ok {
		return s.Reference()
	}

	ref := ""
	sType := reflect.TypeOf(service)
	kind := sType.Kind()
	switch kind {
	case reflect.Struct:
		ref = sType.Name()
	case reflect.Ptr:
		sName := sType.Elem().Name()
		if sName != "" {
			ref = sName
		} else {
			ref = sType.Elem().Field(0).Name
		}
	}
	return ref
}

// AsyncCallbackService callback interface for async
type AsyncCallbackService interface {
	CallBack(response CallbackResponse)
}

// CallbackResponse for different protocol
type CallbackResponse interface{}

// AsyncCallback async callback method
type AsyncCallback func(response CallbackResponse)

const (
	METHOD_MAPPER = "MethodMapper"
)

var (
	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()

	// ServiceMap store description of service.
	ServiceMap = &serviceMap{
		serviceMap:   make(map[string]map[string]*Service),
		interfaceMap: make(map[string][]*Service),
	}
)

// MethodType is description of service method.
type MethodType struct {
	method    reflect.Method
	ctxType   reflect.Type   // request context
	argsType  []reflect.Type // args except ctx, include replyType if existing
	replyType reflect.Type   // return value, otherwise it is nil
}

// Method gets @m.method.
func (m *MethodType) Method() reflect.Method {
	return m.method
}

// CtxType gets @m.ctxType.
func (m *MethodType) CtxType() reflect.Type {
	return m.ctxType
}

// ArgsType gets @m.argsType.
func (m *MethodType) ArgsType() []reflect.Type {
	return m.argsType
}

// ReplyType gets @m.replyType.
func (m *MethodType) ReplyType() reflect.Type {
	return m.replyType
}

// SuiteContext transfers @ctx to reflect.Value type or get it from @m.ctxType.
func (m *MethodType) SuiteContext(ctx context.Context) reflect.Value {
	if ctxV := reflect.ValueOf(ctx); ctxV.IsValid() {
		return ctxV
	}
	return reflect.Zero(m.ctxType)
}

// Service is description of service
type Service struct {
	name     string
	rcvr     reflect.Value
	rcvrType reflect.Type
	methods  map[string]*MethodType
}

// Method gets @s.methods.
func (s *Service) Method() map[string]*MethodType {
	return s.methods
}

// Name will return service name
func (s *Service) Name() string {
	return s.name
}

// RcvrType gets @s.rcvrType.
func (s *Service) RcvrType() reflect.Type {
	return s.rcvrType
}

// Rcvr gets @s.rcvr.
func (s *Service) Rcvr() reflect.Value {
	return s.rcvr
}

type serviceMap struct {
	mutex        sync.RWMutex                   // protects the serviceMap
	serviceMap   map[string]map[string]*Service // protocol -> service name -> service
	interfaceMap map[string][]*Service          // interface -> service
}

// GetService gets a service definition by protocol and name
func (sm *serviceMap) GetService(protocol, interfaceName, group, version string) *Service {
	serviceKey := ServiceKey(interfaceName, group, version)
	return sm.GetServiceByServiceKey(protocol, serviceKey)
}

// GetServiceByServiceKey gets a service definition by protocol and service key
func (sm *serviceMap) GetServiceByServiceKey(protocol, serviceKey string) *Service {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	if s, ok := sm.serviceMap[protocol]; ok {
		if srv, ok := s[serviceKey]; ok {
			return srv
		}
		return nil
	}
	return nil
}

// GetInterface gets an interface definition by interface name
func (sm *serviceMap) GetInterface(interfaceName string) []*Service {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	if s, ok := sm.interfaceMap[interfaceName]; ok {
		return s
	}
	return nil
}

// Register registers a service by @interfaceName and @protocol
func (sm *serviceMap) Register(interfaceName, protocol, group, version string, rcvr RPCService) (string, error) {
	if sm.serviceMap[protocol] == nil {
		sm.serviceMap[protocol] = make(map[string]*Service)
	}
	if sm.interfaceMap[interfaceName] == nil {
		sm.interfaceMap[interfaceName] = make([]*Service, 0, 16)
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

	sname = ServiceKey(interfaceName, group, version)
	if server := sm.GetService(protocol, interfaceName, group, version); server != nil {
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
	sm.interfaceMap[interfaceName] = append(sm.interfaceMap[interfaceName], s)
	sm.mutex.Unlock()

	return strings.TrimSuffix(methods, ","), nil
}

// UnRegister cancels a service by @interfaceName, @protocol and @serviceId
func (sm *serviceMap) UnRegister(interfaceName, protocol, serviceKey string) error {
	if protocol == "" || serviceKey == "" {
		return perrors.New("protocol or ServiceKey is nil")
	}

	var (
		err   error
		index = -1
		svcs  map[string]*Service
		svrs  []*Service
		ok    bool
	)

	f := func() error {
		sm.mutex.RLock()
		defer sm.mutex.RUnlock()
		svcs, ok = sm.serviceMap[protocol]
		if !ok {
			return perrors.New("no services for " + protocol)
		}
		s, ok := svcs[serviceKey]
		if !ok {
			return perrors.New("no service for " + serviceKey)
		}
		svrs, ok = sm.interfaceMap[interfaceName]
		if !ok {
			return perrors.New("no service for " + interfaceName)
		}
		for i, svr := range svrs {
			if svr == s {
				index = i
			}
		}
		return nil
	}

	if err = f(); err != nil {
		return err
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.interfaceMap[interfaceName] = make([]*Service, 0, len(svrs))
	for i := range svrs {
		if i != index {
			sm.interfaceMap[interfaceName] = append(sm.interfaceMap[interfaceName], svrs[i])
		}
	}
	delete(svcs, serviceKey)
	if len(sm.serviceMap[protocol]) == 0 {
		delete(sm.serviceMap, protocol)
	}

	return nil
}

// Is this an exported - upper case - name
func isExported(name string) bool {
	s, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(s)
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

	// Reference is used to define service reference, and method with prefix 'XXX' is generated by triple pb tool.
	// SetGRPCServer is used for pb reflection.
	// They should not to be checked.
	if mname == "Reference" || mname == "SetGRPCServer" || strings.HasPrefix(mname, "XXX") {
		return nil
	}

	if outNum != 1 && outNum != 2 {
		logger.Warnf("method %s of mtype %v has wrong number of in out parameters %d; needs exactly 1/2",
			mname, mtype.String(), outNum)
		return nil
	}

	// The latest return type of the method must be error.
	if returnType := mtype.Out(outNum - 1); returnType != typeOfError {
		logger.Debugf(`"%s" method will not be exported because its last return type %v doesn't have error`, mname, returnType)
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

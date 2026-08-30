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
	"errors"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/dubboutil"
)

// RPCService the type alias of any
type RPCService = any

// ReferencedRPCService is the rpc service interface which wraps base Reference method.
//
// Reference method refers rpc service id or reference id.
type ReferencedRPCService interface {
	Reference() string
}

// TriplePBService is  the type alias of any
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
	case reflect.Pointer:
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
type CallbackResponse any

// AsyncCallback async callback method
type AsyncCallback func(response CallbackResponse)

const (
	METHOD_MAPPER = "MethodMapper"
)

var (
	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeFor[error]()

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

// IsVariadic returns true if the method has a variadic (...T) final parameter.
func (m *MethodType) IsVariadic() bool {
	return m.method.Type.IsVariadic()
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
	name    string
	svc     reflect.Value
	svcType reflect.Type
	methods map[string]*MethodType
}

// Method gets @s.methods.
func (s *Service) Method() map[string]*MethodType {
	return s.methods
}

// Name will return service name
func (s *Service) Name() string {
	return s.name
}

// ServiceType gets @s.SvcType.
func (s *Service) ServiceType() reflect.Type {
	return s.svcType
}

// Service gets @s.Svc.
func (s *Service) Service() reflect.Value {
	return s.svc
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
func (sm *serviceMap) Register(interfaceName, protocol, group, version string, svc RPCService) (string, error) {
	if sm.serviceMap[protocol] == nil {
		sm.serviceMap[protocol] = make(map[string]*Service)
	}
	if sm.interfaceMap[interfaceName] == nil {
		sm.interfaceMap[interfaceName] = make([]*Service, 0, 16)
	}

	s := new(Service)
	s.svcType = reflect.TypeOf(svc)
	s.svc = reflect.ValueOf(svc)
	sname := reflect.Indirect(s.svc).Type().Name()
	if sname == "" {
		s := "no service name for type " + s.svcType.String()
		logger.Errorf("[RPCService] %s", s)
		return "", errors.New(s)
	}
	if !isExported(sname) {
		s := "type " + sname + " is not exported"
		logger.Errorf("[RPCService] %s", s)
		return "", errors.New(s)
	}

	sname = ServiceKey(interfaceName, group, version)
	if server := sm.GetService(protocol, interfaceName, group, version); server != nil {
		return "", errors.New("service already defined: " + sname)
	}
	s.name = sname
	s.methods = make(map[string]*MethodType)

	// Install the methods
	methods := ""
	methods, s.methods = suitableMethods(s.svcType)

	if len(s.methods) == 0 {
		s := "type " + sname + " has no exported methods of suitable type"
		logger.Errorf("[RPCService] %s", s)
		return "", errors.New(s)
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
		return errors.New("protocol or ServiceKey is nil")
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
			return errors.New("no services for " + protocol)
		}
		s, ok := svcs[serviceKey]
		if !ok {
			return errors.New("no service for " + serviceKey)
		}
		svrs, ok = sm.interfaceMap[interfaceName]
		if !ok {
			return errors.New("no service for " + interfaceName)
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
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// VariadicRPCMethodNames returns exported RPC method names whose final
// parameter uses Go variadic syntax (...T). The detection reuses suiteMethod so
// only methods Dubbo-go would export as RPC methods are included.
func VariadicRPCMethodNames(svc RPCService) []string {
	return variadicRPCMethodNames(reflect.TypeOf(svc))
}

// WarnVariadicRPCMethods emits guidance for exported variadic RPC methods while
// keeping existing services compatible.
func WarnVariadicRPCMethods(serviceName string, svc RPCService) {
	methodNames := VariadicRPCMethodNames(svc)
	if len(methodNames) == 0 {
		return
	}

	logger.Warnf(
		"[RPCService] service %s exports variadic RPC method(s): %s. Existing services remain supported, but new cross-language or generic contracts should avoid variadic (...T); prefer []T, request structs, or Triple + Protobuf IDL.",
		serviceName,
		strings.Join(methodNames, ", "),
	)
}

func variadicRPCMethodNames(typ reflect.Type) []string {
	if typ == nil {
		return nil
	}

	methodNames := make([]string, 0)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if suiteMethod(method) != nil && method.Type.IsVariadic() {
			methodNames = append(methodNames, method.Name)
		}
	}

	return methodNames
}

// WarnDiscardedReplyRPCMethods flags exported methods shaped like net/rpc's
// func(args, reply *T) error, whose reply pointer dubbo-go does not send back.
//
// The shape is a leftover from net/rpc, which dubbo-go's early API borrowed —
// suiteMethod still accepts a lone error return, and MethodType.argsType's
// comment still says the reply is among the args. Neither proxy honors it any
// more: the provider builds its response from the method's return values only
// (proxy_factory.ProxyInvoker), and the consumer only allocates a reply when the
// method declares two return values (proxy.Proxy). Whatever the method writes
// into that pointer stays in the provider's memory.
//
// So the author's intent silently does not happen, and nothing else in the stack
// says so. Warning at registration is the only place it can be caught before a
// caller sees an empty result.
//
// Advisory, not a rejection: func(ctx, from *Account, to *Account) error is the
// same shape and is perfectly correct — it just does not return anything, which
// is what dubbo-go will do.
func WarnDiscardedReplyRPCMethods(serviceName string, svc RPCService) {
	methodNames := discardedReplyRPCMethodNames(reflect.TypeOf(svc))
	if len(methodNames) == 0 {
		return
	}

	logger.Warnf(
		"[RPCService] service %s exports method(s) shaped like a net/rpc reply-pointer call: %s. "+
			"dubbo-go returns only a method's declared return values, so anything written into the "+
			"trailing pointer is discarded and the caller receives an empty result. Return the value "+
			"instead, as (T, error).",
		serviceName,
		strings.Join(methodNames, ", "),
	)
}

// discardedReplyRPCMethodNames returns exported RPC methods that return only an
// error and take at least two parameters, the last of them a pointer.
//
// Two parameters, not one: net/rpc's shape is (args, reply), so requiring a
// preceding argument keeps the very common func(ctx, req *Req) error — a
// genuinely void call such as a delete — out of the warning.
func discardedReplyRPCMethodNames(typ reflect.Type) []string {
	if typ == nil {
		return nil
	}

	methodNames := make([]string, 0)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		mt := suiteMethod(method)
		if mt == nil || mt.ReplyType() != nil {
			continue
		}
		args := mt.ArgsType()
		if len(args) >= 2 && args[len(args)-1].Kind() == reflect.Pointer {
			methodNames = append(methodNames, method.Name)
		}
	}

	return methodNames
}

// CanonicalMethod pairs an exported RPC method with the canonical wire name
// dubbo-go advertises for it.
type CanonicalMethod struct {
	// Name is the MethodMapper mapping when one exists, otherwise the Go method
	// name. It never holds the first-rune-swapped alias.
	Name string
	// GoName is the Go method name. Two entries with the same GoName are the
	// same method; this is what distinguishes "one method, two spellings" from
	// "two methods fighting over one name".
	GoName string
	Method *MethodType
}

// MethodNameConflict records two distinct Go methods that resolve to the same
// runtime wire name.
type MethodNameConflict struct {
	First    string
	Second   string
	WireName string
}

func (c MethodNameConflict) String() string {
	return c.First + " and " + c.Second + " are both routable as " + c.WireName
}

// MethodWireNames returns every name a canonical method is routable under at
// runtime: the name itself, plus its first-rune-swapped alias when that differs.
func MethodWireNames(canonical string) []string {
	alias := dubboutil.SwapCaseFirstRune(canonical)
	if alias == canonical {
		return []string{canonical}
	}
	return []string{canonical, alias}
}

// CanonicalMethods returns typ's exported RPC methods in reflect's stable method
// order, each resolved through MethodMapper.
//
// suitableMethods additionally registers a first-rune-swapped alias for every
// name, so Java callers can invoke sayHello on a Go SayHello. Callers that want
// the advertised contract rather than the runtime lookup table need this
// alias-free view — iterating the map suitableMethods returns would publish each
// method twice under two spellings.
func CanonicalMethods(typ reflect.Type) []CanonicalMethod {
	methodMapper := methodMapperOf(typ)

	methods := make([]CanonicalMethod, 0, typ.NumMethod())
	for m := range typ.NumMethod() {
		method := typ.Method(m)
		mt := suiteMethod(method)
		if mt == nil {
			continue
		}
		name, mapped := methodMapper[method.Name]
		if !mapped {
			name = method.Name
		}
		methods = append(methods, CanonicalMethod{Name: name, GoName: method.Name, Method: mt})
	}
	return methods
}

// MethodNameConflicts reports canonical names whose runtime wire-name sets
// intersect.
//
// Both MethodMapper and the alias mechanism can produce these: two Go methods
// mapped onto one name, or onto names that are first-rune-case variants of each
// other. Either way the later registration silently overwrites the earlier one
// in the lookup map, so one method becomes unreachable while calls to its name
// land on the other.
func MethodNameConflicts(methods []CanonicalMethod) []MethodNameConflict {
	owner := make(map[string]CanonicalMethod, len(methods)*2)
	reported := make(map[string]bool)
	var conflicts []MethodNameConflict
	for _, m := range methods {
		for _, wire := range MethodWireNames(m.Name) {
			prev, taken := owner[wire]
			if taken && prev.GoName != m.GoName {
				// One entry per pair of methods, not per contested name. Two
				// methods whose canonical names are first-rune variants of each
				// other contest both spellings, and reporting each would log the
				// same problem twice. Iteration follows the caller's stable
				// slice order, so prev is always the earlier method and the pair
				// key is consistently ordered.
				pair := prev.GoName + "\x00" + m.GoName
				if !reported[pair] {
					reported[pair] = true
					conflicts = append(conflicts, MethodNameConflict{
						First:    prev.GoName,
						Second:   m.GoName,
						WireName: wire,
					})
				}
				continue
			}
			owner[wire] = m
		}
	}
	return conflicts
}

// methodMapperOf invokes the service's MethodMapper hook, if it declares one.
func methodMapperOf(typ reflect.Type) map[string]string {
	method, ok := typ.MethodByName(METHOD_MAPPER)
	if !ok || method.Type.NumIn() != 1 || method.Type.NumOut() != 1 ||
		method.Type.Out(0).String() != "map[string]string" {
		return nil
	}
	return method.Func.Call([]reflect.Value{reflect.New(typ.Elem())})[0].Interface().(map[string]string)
}

// suitableMethods returns suitable Rpc methods of typ
func suitableMethods(typ reflect.Type) (string, map[string]*MethodType) {
	logger.Debugf("[RPCService] NumMethod is %d, type=%s", typ.NumMethod(), typ.String())

	canonical := CanonicalMethods(typ)

	// Conflicts are reported but not fatal: services that already run with an
	// overwriting name collision must keep starting after an upgrade. The
	// contract builder refuses to publish the affected methods, so the warning
	// is the only signal a maintainer gets here — worth keeping loud.
	for _, conflict := range MethodNameConflicts(canonical) {
		logger.Warnf("[RPCService] method name conflict on type %s: %s. "+
			"Only one of them is reachable at runtime, and neither is published "+
			"in the service definition.", typ.String(), conflict)
	}

	methods := make(map[string]*MethodType, len(canonical)*2)
	mts := make([]string, 0, len(canonical)*2)
	for _, m := range canonical {
		methods[m.Name] = m.Method
		mts = append(mts, m.Name)
		// For better interoperability with java class, we convert the first
		// letter in methodName between upper and lower case.
		alias := dubboutil.SwapCaseFirstRune(m.Name)
		methods[alias] = m.Method
		mts = append(mts, alias)
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
	// Health helper methods are not RPCs and should be ignored.
	// They should not to be checked.
	if mname == "Reference" || mname == "SetGRPCServer" || strings.HasPrefix(mname, "XXX") ||
		(method.Type.In(0).String() == "*health.HealthTripleServer" &&
			(mname == "Resume" || mname == "SetServingStatus" || mname == "Shutdown")) {
		return nil
	}

	if outNum != 1 && outNum != 2 {
		logger.Warnf("[RPCService] method %s of mtype %v has wrong number of in out parameters %d; needs exactly 1/2",
			mname, mtype.String(), outNum)
		return nil
	}

	// The latest return type of the method must be error.
	if returnType := mtype.Out(outNum - 1); returnType != typeOfError {
		logger.Debugf(`[RPCService] "%s" method will not be exported because its last return type %v doesn't have error`, mname, returnType)
		return nil
	}

	// replyType
	if outNum == 2 {
		replyType = mtype.Out(0)
		if !isExportedOrBuiltinType(replyType) {
			logger.Errorf("[RPCService] reply type of method %s not exported, type=%v", mname, replyType)
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
			logger.Errorf("[RPCService] argument type of method %q is not exported, type=%v", mname, mtype.In(index))
			return nil
		}
	}

	return &MethodType{method: method, argsType: argsType, replyType: replyType, ctxType: ctxType}
}

// ServiceInfo is meta info of a service
type ServiceInfo struct {
	InterfaceName string
	ServiceType   any
	Methods       []MethodInfo
	Meta          map[string]any
}

type MethodInfo struct {
	Name           string
	Type           string
	ReqInitFunc    func() any
	StreamInitFunc func(baseStream any) any
	MethodFunc     func(ctx context.Context, args []any, handler any) (any, error)
	Meta           map[string]any
}

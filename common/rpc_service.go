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
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

// rpc service interface
type RPCService interface {
	Service() string // Path InterfaceName
	Version() string
}

var (
	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()

	ServiceMap = &serviceMap{
		serviceMap: make(map[string]map[string]*Service),
	}
)

//////////////////////////
// info of method
//////////////////////////

type MethodType struct {
	method    reflect.Method
	ctxType   reflect.Type // type of the request context
	argType   reflect.Type
	replyType reflect.Type
}

func (m *MethodType) Method() reflect.Method {
	return m.method
}
func (m *MethodType) CtxType() reflect.Type {
	return m.ctxType
}
func (m *MethodType) ArgType() reflect.Type {
	return m.argType
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
		log.Error(s)
		return "", jerrors.New(s)
	}
	if !isExported(sname) {
		s := "type " + sname + " is not exported"
		log.Error(s)
		return "", jerrors.New(s)
	}

	sname = rcvr.Service()
	if server := sm.GetService(protocol, sname); server != nil {
		return "", jerrors.New("service already defined: " + sname)
	}
	s.name = sname
	s.methods = make(map[string]*MethodType)

	// Install the methods
	methods := ""
	methods, s.methods = suitableMethods(s.rcvrType)

	if len(s.methods) == 0 {
		s := "type " + sname + " has no exported methods of suitable type"
		log.Error(s)
		return "", jerrors.New(s)
	}
	sm.mutex.Lock()
	sm.serviceMap[protocol][s.name] = s
	sm.mutex.Unlock()

	return strings.TrimSuffix(methods, ","), nil
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
	mts := ""
	log.Debug("[%s] NumMethod is %d", typ.String(), typ.NumMethod())
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if mt := suiteMethod(method); mt != nil {
			methods[method.Name] = mt
			if m == 0 {
				mts += method.Name
			} else {
				mts += "," + method.Name
			}
		}
	}
	return mts, methods
}

// suiteMethod returns a suitable Rpc methodType
func suiteMethod(method reflect.Method) *MethodType {
	mtype := method.Type
	mname := method.Name

	// Method must be exported.
	if method.PkgPath != "" {
		return nil
	}

	var replyType, argType, ctxType reflect.Type
	switch mtype.NumIn() {
	case 3:
		argType = mtype.In(1)
		replyType = mtype.In(2)
	case 4:
		ctxType = mtype.In(1)
		argType = mtype.In(2)
		replyType = mtype.In(3)
	default:
		log.Error("method %s of mtype %v has wrong number of in parameters %d; needs exactly 3/4",
			mname, mtype, mtype.NumIn())
		return nil
	}
	// First arg need not be a pointer.
	if !isExportedOrBuiltinType(argType) {
		log.Error("argument type of method %q is not exported %v", mname, argType)
		return nil
	}
	// Second arg must be a pointer.
	if replyType.Kind() != reflect.Ptr {
		log.Error("reply type of method %q is not a pointer %v", mname, replyType)
		return nil
	}
	// Reply type must be exported.
	if !isExportedOrBuiltinType(replyType) {
		log.Error("reply type of method %s not exported{%v}", mname, replyType)
		return nil
	}
	// Method needs one out.
	if mtype.NumOut() != 1 {
		log.Error("method %q has %d out parameters; needs exactly 1", mname, mtype.NumOut())
		return nil
	}
	// The return type of the method must be error.
	if returnType := mtype.Out(0); returnType != typeOfError {
		log.Error("return type %s of method %q is not error", returnType, mname)
		return nil
	}

	return &MethodType{method: method, argType: argType, replyType: replyType, ctxType: ctxType}
}

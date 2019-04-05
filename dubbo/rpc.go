package dubbo

import (
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
)

import (
	log "github.com/AlexStocks/log4go"
)

var (
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()
)

type GettyRPCService interface {
	Service() string // Service Interface
	Version() string
}

type methodType struct {
	sync.Mutex
	method    reflect.Method
	CtxType   reflect.Type // type of the request context
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name     string
	rcvr     reflect.Value
	rcvrType reflect.Type
	method   map[string]*methodType
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
func suitableMethods(typ reflect.Type) (string, map[string]*methodType) {
	methods := make(map[string]*methodType)
	mts := ""
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if mt := suiteMethod(method); mt != nil {
			methods[method.Name] = mt
			mts += method.Name + ","
		}
	}
	return mts, methods
}

// suiteMethod returns a suitable Rpc methodType
func suiteMethod(method reflect.Method) *methodType {
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

	return &methodType{method: method, ArgType: argType, ReplyType: replyType, CtxType: ctxType}
}

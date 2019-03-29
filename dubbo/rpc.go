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
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name   string
	rcvr   reflect.Value
	typ    reflect.Type
	method map[string]*methodType
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
func suitableMethods(typ reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// service Method needs three ins: receiver, *args, *reply.
		// notify Method needs two ins: receiver, *args.
		mInNum := mtype.NumIn()
		if mInNum != 2 && mInNum != 3 {
			log.Warn("method %s has wrong number of ins %d which should be "+
				"2(notify method) or 3(serive method)", mname, mtype.NumIn())
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			log.Error("method{%s} argument type not exported{%v}", mname, argType)
			continue
		}

		var replyType reflect.Type
		if mInNum == 3 {
			// Second arg must be a pointer.
			replyType = mtype.In(2)
			if replyType.Kind() != reflect.Ptr {
				log.Error("method{%s} reply type not a pointer{%v}", mname, replyType)
				continue
			}
			// Reply type must be exported.
			if !isExportedOrBuiltinType(replyType) {
				log.Error("method{%s} reply type not exported{%v}", mname, replyType)
				continue
			}
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			log.Error("method{%s} has wrong number of out parameters{%d}", mname, mtype.NumOut())
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			log.Error("method{%s}'s return type{%s} is not error", mname, returnType.String())
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

package proxy

import (
	"reflect"
)
import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
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
	log.Debug("[Implement] reflect.TypeOf: %s", valueOf.String())

	valueOfElem := valueOf.Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		log.Error("%s must be a struct ptr", valueOf.String())
		return
	}

	makeDubboCallProxy := func(methodName string, outs []reflect.Type) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {

			if methodName == "Echo" {
				methodName = "$echo"
			}
			inv := invocation_impl.NewRPCInvocationForConsumer(methodName, nil, in[1].Interface().([]interface{}), in[2].Interface(), p.callBack, common.URL{}, nil)

			for k, v := range p.attachments {
				inv.SetAttachments(k, v)
			}

			result := p.invoke.Invoke(inv)
			var err error
			err = result.Error()
			log.Info("[makeDubboCallProxy] err: %v", err)
			return []reflect.Value{reflect.ValueOf(&err).Elem()}
		}
	}

	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if f.Kind() == reflect.Func && f.IsValid() && f.CanSet() {

			if t.Type.NumIn() != 3 && t.Type.NumIn() != 4 {
				log.Warn("method %s of mtype %v has wrong number of in parameters %d; needs exactly 3/4",
					t.Name, t.Type.String(), t.Type.NumIn())
				continue
			}

			if t.Type.NumIn() == 3 && t.Type.In(2).Kind() != reflect.Ptr {
				log.Warn("reply type of method %q is not a pointer %v", t.Name, t.Type.In(2))
				continue
			}

			if t.Type.NumIn() == 4 && t.Type.In(3).Kind() != reflect.Ptr {
				log.Warn("reply type of method %q is not a pointer %v", t.Name, t.Type.In(3))
				continue
			}

			// Method needs one out.
			if t.Type.NumOut() != 1 {
				log.Warn("method %q has %d out parameters; needs exactly 1", t.Name, t.Type.NumOut())
				continue
			}
			// The return type of the method must be error.
			if returnType := t.Type.Out(0); returnType != typError {
				log.Warn("return type %s of method %q is not error", returnType, t.Name)
				continue
			}

			var funcOuts = make([]reflect.Type, t.Type.NumOut())
			funcOuts[0] = t.Type.Out(0)

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeDubboCallProxy(t.Name, funcOuts)))
			log.Debug("set method [%s]", t.Name)
		}
	}

	p.rpc = v

}

func (p *Proxy) Get() common.RPCService {
	return p.rpc
}

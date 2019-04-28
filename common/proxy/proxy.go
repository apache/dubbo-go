package proxy

import (
	"fmt"
	"reflect"
)
import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/protocol"
)

// Proxy struct
type Proxy struct {
	v           interface{}
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
func (p *Proxy) Implement(v interface{}) error {

	// check parameters, incoming interface must be a elem's pointer.
	valueOf := reflect.ValueOf(v)
	log.Debug("[Implement] reflect.TypeOf: %s", valueOf.String())
	if valueOf.Kind() != reflect.Ptr {
		return fmt.Errorf("%s must be a pointer", valueOf)
	}

	valueOfElem := valueOf.Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		return fmt.Errorf("%s must be a struct ptr", valueOf.String())
	}

	makeDubboCallProxy := func(methodName string, outs []reflect.Type) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			// Convert input parameters to interface.
			var argsInterface = make([]interface{}, len(in))
			for k, v := range in {
				argsInterface[k] = v.Interface()
			}

			//todo:call
			inv := protocol.NewRPCInvocationForConsumer(methodName, nil, argsInterface, in[2].Interface(), p.callBack, p.attachments, nil)
			result := p.invoke.Invoke(inv)
			var err error
			err = result.Error()
			return []reflect.Value{reflect.ValueOf(&err).Elem()}
		}
	}

	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if f.Kind() == reflect.Func && f.IsValid() && f.CanSet() {

			if t.Type.NumIn() != 3 && t.Type.NumIn() != 4 {
				log.Error("method %s of mtype %v has wrong number of in parameters %d; needs exactly 3/4",
					t.Name, t.Type.String(), t.Type.NumIn())
				return fmt.Errorf("method %s of mtype %v has wrong number of in parameters %d; needs exactly 3/4",
					t.Name, t.Type.String(), t.Type.NumIn())
			}

			// Method needs one out.
			if t.Type.NumOut() != 1 {
				log.Error("method %q has %d out parameters; needs exactly 1", t.Name, t.Type.NumOut())
				return fmt.Errorf("method %q has %d out parameters; needs exactly 1", t.Name, t.Type.NumOut())
			}
			// The return type of the method must be error.
			if returnType := t.Type.Out(0); returnType != typError {
				log.Error("return type %s of method %q is not error", returnType, t.Name)
				return fmt.Errorf("return type %s of method %q is not error", returnType, t.Name)
			}

			var funcOuts = make([]reflect.Type, t.Type.NumOut())
			funcOuts[i] = t.Type.Out(0)

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeDubboCallProxy(t.Name, funcOuts)))
			log.Debug("set method [%s]", t.Name)
		}
	}

	p.v = v

	return nil
}

func (p *Proxy) Get() interface{} {
	return p.v
}

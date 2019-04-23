package protocol

import "reflect"

type Invocation interface {
	MethodName() string
	Parameters() []reflect.Value
}

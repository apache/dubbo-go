package protocol

import (
	"reflect"
)

type Invocation interface {
	MethodName() string
	ParameterTypes() []reflect.Type
	Arguments() []interface{}
	Reply() interface{}
	Attachments() map[string]string
	AttachmentsByKey(string, string) string
	Invoker() Invoker
}

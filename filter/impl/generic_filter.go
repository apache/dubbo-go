package impl

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
	"reflect"
	"strings"
)

const (
	GENERIC = "generic"
)

func init() {
	extension.SetFilter(GENERIC, GetGenericFilter)
}

//  when do a generic invoke, struct need to be map

type GenericFilter struct{}

func (ef *GenericFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() == constant.GENERIC {
		var newArguments = invocation.Arguments()
		for i := range newArguments {
			newArguments[i] = Struct2MapAll(newArguments[i])
		}
		newInvocation := invocation2.NewRPCInvocation(invocation.MethodName(), newArguments, invocation.Attachments())
		return invoker.Invoke(newInvocation)
	}
	return invoker.Invoke(invocation)

}

func (ef *GenericFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func GetGenericFilter() filter.Filter {
	return &GenericFilter{}
}
func Struct2MapAll(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	result := make(map[string]interface{})
	if reflect.TypeOf(obj).Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			if v.Field(i).Kind() == reflect.Struct {
				if v.Field(i).CanInterface() {
					result[headerAtoa(t.Field(i).Name)] = Struct2MapAll(v.Field(i).Interface())
				} else {
					println("not in to map,field:" + t.Field(i).Name)
				}
			} else {
				if v.Field(i).CanInterface() {
					if tagName := t.Field(i).Tag.Get("m"); tagName == "" {
						result[headerAtoa(t.Field(i).Name)] = v.Field(i).Interface()
					} else {
						result[tagName] = v.Field(i).Interface()
					}
				} else {
					println("not in to map,field:" + t.Field(i).Name)
				}
			}
		}
	}
	return result
}
func headerAtoa(a string) (b string) {
	b = strings.ToLower(a[:1]) + a[1:]
	return
}

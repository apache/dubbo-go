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

package impl

import (
	"reflect"
	"strings"
)
import (
	hessian "github.com/apache/dubbo-go-hessian2"
)
import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
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
	if invocation.MethodName() == constant.GENERIC && len(invocation.Arguments()) == 3 {
		oldArguments := invocation.Arguments()
		var newParams []hessian.Object
		if oldParams, ok := oldArguments[2].([]interface{}); ok {
			for i := range oldParams {
				newParams = append(newParams, hessian.Object(struct2MapAll(oldParams[i])))
			}
		} else {
			return invoker.Invoke(invocation)
		}
		newArguments := []interface{}{
			oldArguments[0],
			oldArguments[1],
			newParams,
		}
		newInvocation := invocation2.NewRPCInvocation(invocation.MethodName(), newArguments, invocation.Attachments())
		newInvocation.SetReply(invocation.Reply())
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
func struct2MapAll(obj interface{}) interface{} {
	if obj == nil {
		return obj
	}
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if t.Kind() == reflect.Struct {
		result := make(map[string]interface{}, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			if v.Field(i).Kind() == reflect.Struct || v.Field(i).Kind() == reflect.Slice || v.Field(i).Kind() == reflect.Map {
				if v.Field(i).CanInterface() {
					setInMap(result, t.Field(i), struct2MapAll(v.Field(i).Interface()))
				}
			} else {
				if v.Field(i).CanInterface() {
					setInMap(result, t.Field(i), v.Field(i).Interface())
				}
			}
		}
		return result
	} else if t.Kind() == reflect.Slice {
		value := reflect.ValueOf(obj)
		var newTemps = make([]interface{}, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			newTemp := struct2MapAll(value.Index(i).Interface())
			newTemps = append(newTemps, newTemp)
		}
		return newTemps
	} else if t.Kind() == reflect.Map {
		var newTempMap = make(map[string]interface{}, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			mapK := iter.Key().String()
			if !iter.Value().CanInterface() {
				continue
			}
			mapV := iter.Value().Interface()
			newTempMap[mapK] = struct2MapAll(mapV)
		}
		return newTempMap
	} else {
		return obj
	}
}
func setInMap(m map[string]interface{}, structField reflect.StructField, value interface{}) (result map[string]interface{}) {
	result = m
	if tagName := structField.Tag.Get("m"); tagName == "" {
		result[headerAtoa(structField.Name)] = value
	} else {
		result[tagName] = value
	}
	return
}
func headerAtoa(a string) (b string) {
	b = strings.ToLower(a[:1]) + a[1:]
	return
}

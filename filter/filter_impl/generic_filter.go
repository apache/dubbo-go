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

package filter_impl

import (
	"context"
	"reflect"
	"strings"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	// GENERIC
	// generic module name
	GENERIC = "generic"
)

func init() {
	extension.SetFilter(GENERIC, GetGenericFilter)
}

//  when do a generic invoke, struct need to be map

// nolint
type GenericFilter struct{}

// Invoke turns the parameters to map for generic method
func (ef *GenericFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() == constant.GENERIC && len(invocation.Arguments()) == 3 {
		oldArguments := invocation.Arguments()

		if oldParams, ok := oldArguments[2].([]interface{}); ok {
			newParams := make([]hessian.Object, 0, len(oldParams))
			for i := range oldParams {
				newParams = append(newParams, hessian.Object(struct2MapAll(oldParams[i])))
			}
			newArguments := []interface{}{
				oldArguments[0],
				oldArguments[1],
				newParams,
			}
			newInvocation := invocation2.NewRPCInvocation(invocation.MethodName(), newArguments, invocation.Attachments())
			newInvocation.SetReply(invocation.Reply())
			return invoker.Invoke(ctx, newInvocation)
		}
	}
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (ef *GenericFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {
	return result
}

// GetGenericFilter returns GenericFilter instance
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
			field := t.Field(i)
			value := v.Field(i)
			kind := value.Kind()
			if kind == reflect.Struct || kind == reflect.Slice || kind == reflect.Map {
				if value.CanInterface() {
					tmp := value.Interface()
					if _, ok := tmp.(time.Time); ok {
						setInMap(result, field, tmp)
						continue
					}
					setInMap(result, field, struct2MapAll(tmp))
				}
			} else {
				if value.CanInterface() {
					setInMap(result, field, value.Interface())
				}
			}
		}
		return result
	} else if t.Kind() == reflect.Slice {
		value := reflect.ValueOf(obj)
		newTemps := make([]interface{}, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			newTemp := struct2MapAll(value.Index(i).Interface())
			newTemps = append(newTemps, newTemp)
		}
		return newTemps
	} else if t.Kind() == reflect.Map {
		newTempMap := make(map[interface{}]interface{}, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			if !iter.Value().CanInterface() {
				continue
			}
			key := iter.Key()
			mapV := iter.Value().Interface()
			newTempMap[convertMapKey(key)] = struct2MapAll(mapV)
		}
		return newTempMap
	} else {
		return obj
	}
}

func convertMapKey(key reflect.Value) interface{} {
	switch key.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8,
		reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Float32,
		reflect.Float64, reflect.String:
		return key.Interface()
	default:
		return key.String()
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

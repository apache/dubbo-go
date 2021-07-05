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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	GENERIC = "generic"
)

func init() {
	extension.SetFilter(GENERIC, GetGenericFilter)
}

// GenericFilter ensures the structs are converted to maps
type GenericFilter struct{}

// Invoke turns the parameters to map for generic method
func (ef *GenericFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if isCallingToGenericService(invoker, invocation) { // call to a generic service
		mtdname := invocation.MethodName()
		oldargs := invocation.Arguments()

		// TODO: get types of args
		types := make([]interface{}, 0, len(oldargs))

		// convert args to map
		args := make([]interface{}, 0, len(oldargs))
		for _, arg := range oldargs {
			args = append(args, objToMap(arg))
		}

		newargs := []interface{} {
			mtdname,
			types,
			args,
		}

		newivc := invocation2.NewRPCInvocation(constant.GENERIC, newargs, invocation.Attachments())
		newivc.SetReply(invocation.Reply())
		return invoker.Invoke(ctx, newivc)
	} else if isMakingAGenericCall(invoker, invocation) { // making a generic call to normal service
		// TODO: type check
		invocation.Attachments()[constant.GENERIC_KEY] = invoker.GetURL().GetParam(constant.GENERIC_KEY, "")
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



// objToMap converts an object(interface{}) to a map
func objToMap(obj interface{}) interface{} {
	if obj == nil {
		return obj
	}
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if t.Kind() == reflect.Struct { // for struct
		result := make(map[string]interface{}, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			kind := value.Kind()
			if !value.CanInterface() {
				logger.Debugf("objToMap for %v is skipped because it couldn't be converted to interface", field)
				continue
			}
			valueIface := value.Interface()
			if kind == reflect.Struct || kind == reflect.Slice || kind == reflect.Map {
				if _, ok := valueIface.(time.Time); ok {
					setInMap(result, field, valueIface)
					continue
				}
				setInMap(result, field, objToMap(valueIface))
			} else {
				setInMap(result, field, valueIface)
			}
		}
		return result
	} else if t.Kind() == reflect.Slice { // for slice
		value := reflect.ValueOf(obj)
		newTemps := make([]interface{}, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			newTemp := objToMap(value.Index(i).Interface())
			newTemps = append(newTemps, newTemp)
		}
		return newTemps
	} else if t.Kind() == reflect.Map { // for map
		newTempMap := make(map[interface{}]interface{}, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			if !iter.Value().CanInterface() {
				continue
			}
			key := iter.Key()
			mapV := iter.Value().Interface()
			newTempMap[mapKeyToIface(key)] = objToMap(mapV)
		}
		return newTempMap
	} else {
		return obj
	}
}

// mapKeyToIface converts the map key to interface type
func mapKeyToIface(key reflect.Value) interface{} {
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

// setInMap sets the struct into the map using the tag or the name of the struct as the key
func setInMap(m map[string]interface{}, structField reflect.StructField, value interface{}) (result map[string]interface{}) {
	result = m
	if tagName := structField.Tag.Get("m"); tagName == "" {
		result[firstLetterToLower(structField.Name)] = value
	} else {
		result[tagName] = value
	}
	return
}

// firstLetterToLower is to lower the first letter
func firstLetterToLower(a string) (b string) {
	b = strings.ToLower(a[:1]) + a[1:]
	return
}

func isCallingToGenericService(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GENERIC_KEY, "")) &&
		invocation.MethodName() != constant.GENERIC
}

func isMakingAGenericCall(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GENERIC_KEY, "")) &&
		invocation.MethodName() == constant.GENERIC &&
		invocation.Arguments() != nil &&
		len(invocation.Arguments()) == 3
}

func isGeneric(generic string) bool {
	lowerGeneric := strings.ToLower(generic)
	return lowerGeneric != "" && (
		lowerGeneric == constant.GENERIC_SERIALIZATION_DEFAULT ||
			lowerGeneric == constant.GENERIC_SERIALIZATION_PROTOBUF)
}

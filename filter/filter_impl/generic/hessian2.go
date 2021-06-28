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

package generic

import (
	"reflect"
	"strings"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
)

const (
	// same with dubbo
	Hessian2 = "true"
)

func init() {
	extension.SetGenericProcessor(Hessian2, NewHessian2GenericProcessor())
}

type hessian2GenericProcessor struct{}

func NewHessian2GenericProcessor() filter.GenericProcessor {
	return &hessian2GenericProcessor{}
}

func (hessian2GenericProcessor) Serialize(args []interface{}) ([]interface{}, error) {
	var newArgs []interface{}
	if oldParams, ok := args[2].([]interface{}); ok {
		newParams := make([]hessian.Object, 0, len(oldParams))
		for i := range oldParams {
			newParams = append(newParams, hessian.Object(Struct2MapAll(oldParams[i])))
		}
		newArgs = []interface{}{
			args[0],
			args[1],
			newParams,
		}
	}
	return newArgs, nil
}

func (hessian2GenericProcessor) Deserialize(args interface{}) ([]interface{}, error) {
	oldArgs, ok := args.([]hessian.Object)
	if !ok {
		return nil, perrors.New("Deserialize generic invoke arguments error")
	}
	// convert []hessian.Object to []interface{}
	newArgs := make([]interface{}, len(oldArgs))
	for i := range oldArgs {
		newArgs[i] = oldArgs[i]
	}
	return newArgs, nil
}

func Struct2MapAll(obj interface{}) interface{} {
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
					setInMap(result, field, Struct2MapAll(tmp))
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
			newTemp := Struct2MapAll(value.Index(i).Interface())
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
			newTempMap[convertMapKey(key)] = Struct2MapAll(mapV)
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

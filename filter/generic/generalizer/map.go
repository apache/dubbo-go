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

package generalizer

import (
	"reflect"
	"strings"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	"github.com/mitchellh/mapstructure"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
)

var (
	mapGeneralizer     Generalizer
	mapGeneralizerOnce sync.Once
)

func GetMapGeneralizer() Generalizer {
	mapGeneralizerOnce.Do(func() {
		mapGeneralizer = &MapGeneralizer{}
	})
	return mapGeneralizer
}

type MapGeneralizer struct{}

func (g *MapGeneralizer) Generalize(obj interface{}) (gobj interface{}, err error) {
	gobj = objToMap(obj)
	return
}

func (g *MapGeneralizer) Realize(obj interface{}, typ reflect.Type) (interface{}, error) {
	newobj := reflect.New(typ).Interface()
	err := mapstructure.Decode(obj, newobj)
	if err != nil {
		return nil, perrors.Errorf("realizing map failed, %v", err)
	}

	return reflect.ValueOf(newobj).Elem().Interface(), nil
}

func (g *MapGeneralizer) GetType(obj interface{}) (typ string, err error) {
	typ, err = hessian2.GetJavaName(obj)
	// no error or error is not NilError
	if err == nil || err != hessian2.NilError {
		return
	}

	typ = "java.lang.Object"
	if err == hessian2.NilError {
		logger.Debugf("the type of nil object couldn't be inferred, use the default value(\"%s\")", typ)
		return
	}

	logger.Debugf("the type of object(=%T) couldn't be recognized as a POJO, use the default value(\"%s\")", obj, typ)
	return
}

// objToMap converts an object(interface{}) to a map
func objToMap(obj interface{}) interface{} {
	if obj == nil {
		return obj
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	// if obj is a POJO, get the struct from the pointer (if it is a pointer)
	pojo, isPojo := obj.(hessian.POJO)
	if isPojo {
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
			v = v.Elem()
		}
	}

	switch t.Kind() {
	case reflect.Struct:
		result := make(map[string]interface{}, t.NumField())
		if isPojo {
			result["class"] = pojo.JavaClassName()
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			kind := value.Kind()
			if !value.CanInterface() {
				logger.Debugf("objToMap for %v is skipped because it couldn't be converted to interface", field)
				continue
			}
			valueIface := value.Interface()
			switch kind {
			case reflect.Ptr:
				if value.IsNil() {
					setInMap(result, field, nil)
					continue
				}
				setInMap(result, field, objToMap(valueIface))
			case reflect.Struct, reflect.Slice, reflect.Map:
				if isPrimitive(valueIface) {
					logger.Warnf("\"%s\" is primitive. The application may crash if it's transferred between "+
						"systems implemented by different languages, e.g. dubbo-go <-> dubbo-java. We recommend "+
						"you represent the object by basic types, like string.", value.Type())
					setInMap(result, field, valueIface)
					continue
				}

				setInMap(result, field, objToMap(valueIface))
			default:
				setInMap(result, field, valueIface)
			}
		}
		return result
	case reflect.Array, reflect.Slice:
		value := reflect.ValueOf(obj)
		newTemps := make([]interface{}, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			newTemp := objToMap(value.Index(i).Interface())
			newTemps = append(newTemps, newTemp)
		}
		return newTemps
	case reflect.Map:
		newTempMap := make(map[interface{}]interface{}, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			if !iter.Value().CanInterface() {
				continue
			}
			key := iter.Key()
			mapV := iter.Value().Interface()
			newTempMap[mapKey(key)] = objToMap(mapV)
		}
		return newTempMap
	case reflect.Ptr:
		return objToMap(v.Elem().Interface())
	default:
		return obj
	}
}

// mapKey converts the map key to interface type
func mapKey(key reflect.Value) interface{} {
	switch key.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8,
		reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Float32,
		reflect.Float64, reflect.String:
		return key.Interface()
	default:
		name := key.String()
		if name == "class" {
			panic(`"class" is a reserved keyword`)
		}
		return name
	}
}

// setInMap sets the struct into the map using the tag or the name of the struct as the key
func setInMap(m map[string]interface{}, structField reflect.StructField, value interface{}) (result map[string]interface{}) {
	result = m
	if tagName := structField.Tag.Get("m"); tagName == "" {
		result[toUnexport(structField.Name)] = value
	} else {
		result[tagName] = value
	}
	return
}

// toUnexport is to lower the first letter
func toUnexport(a string) string {
	return strings.ToLower(a[:1]) + a[1:]
}

// isPrimitive determines if the object is primitive
func isPrimitive(obj interface{}) bool {
	if _, ok := obj.(time.Time); ok {
		return true
	}
	return false
}

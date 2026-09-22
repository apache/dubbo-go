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
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"strconv"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	"github.com/mitchellh/mapstructure"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/internal/genericfield"
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

func (g *MapGeneralizer) Generalize(obj any) (gobj any, err error) {
	gobj, err = objToMap(obj)
	if err != nil {
		return nil, fmt.Errorf("generalizing map failed, %v", err)
	}
	if !getGenericIncludeClass() {
		gobj = removeClass(gobj)
	}
	return
}

func (g *MapGeneralizer) Realize(obj any, typ reflect.Type) (any, error) {
	if !getGenericIncludeClass() {
		obj = removeClass(obj)
	}
	newobj := reflect.New(typ).Interface()
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: genericRealizeDecodeHook,
		Result:     newobj,
		TagName:    "m",
	})
	if err != nil {
		return nil, fmt.Errorf("creating map decoder failed, %v", err)
	}

	err = decoder.Decode(obj)
	if err != nil {
		return nil, fmt.Errorf("realizing map failed, %v", err)
	}

	return reflect.ValueOf(newobj).Elem().Interface(), nil
}

// genericRealizeDecodeHook closes two gaps in mapstructure's numeric
// conversion that matter to generic RPC contracts:
//   - SetUint truncates values above the target type's maximum;
//   - Java byte[] carries signed bytes while Go []byte carries the same bits as
//     uint8 values.
//
// The first must fail loudly rather than silently change an RPC argument. The
// second is a deliberate bit-preserving conversion and only applies inside a
// byte slice/array, never to a scalar uint8 (which definitions publish as Java
// short so 0..255 stays directly representable).
func genericRealizeDecodeHook(from, to reflect.Type, data any) (any, error) {
	if from == nil || to == nil || data == nil {
		return data, nil
	}

	if isByteSequence(to) && (from.Kind() == reflect.Slice || from.Kind() == reflect.Array) {
		return realizeJavaByteSequence(data, to)
	}

	if isUnsignedKind(to.Kind()) {
		if err := validateUnsignedValue(data, to); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func isByteSequence(t reflect.Type) bool {
	return (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) &&
		t.Elem().Kind() == reflect.Uint8
}

func realizeJavaByteSequence(data any, target reflect.Type) (any, error) {
	source := reflect.ValueOf(data)
	result := make([]byte, source.Len())
	byteType := reflect.TypeFor[byte]()
	for i := range source.Len() {
		value := source.Index(i)
		for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return nil, fmt.Errorf("nil byte at index %d cannot be realized as %s", i, target)
			}
			value = value.Elem()
		}
		if value.Kind() == reflect.Int8 {
			// Java byte and Go byte carry the same eight bits under different
			// signedness. Preserve those bits instead of treating -1 as an
			// out-of-range numeric value.
			result[i] = byte(int8(value.Int()))
			continue
		}
		converted, recognized, err := checkedUnsignedValue(value.Interface(), byteType)
		if err != nil {
			return nil, fmt.Errorf("byte at index %d: %w", i, err)
		}
		if !recognized {
			return nil, fmt.Errorf("byte at index %d has unsupported type %s", i, value.Type())
		}
		result[i] = byte(converted)
	}
	return result, nil
}

func isUnsignedKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}

func validateUnsignedValue(data any, target reflect.Type) error {
	_, _, err := checkedUnsignedValue(data, target)
	return err
}

func checkedUnsignedValue(data any, target reflect.Type) (uint64, bool, error) {
	if number, ok := data.(json.Number); ok {
		value, err := strconv.ParseUint(string(number), 10, target.Bits())
		if err != nil {
			return 0, true, fmt.Errorf("value %s overflows %s: %w", number, target, err)
		}
		if target.OverflowUint(value) {
			return 0, true, fmt.Errorf("value %s overflows %s", number, target)
		}
		return value, true, nil
	}

	value := reflect.ValueOf(data)
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, false, nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		signed := value.Int()
		if signed < 0 || target.OverflowUint(uint64(signed)) {
			return 0, true, fmt.Errorf("value %d overflows %s", signed, target)
		}
		return uint64(signed), true, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if target.OverflowUint(value.Uint()) {
			return 0, true, fmt.Errorf("value %d overflows %s", value.Uint(), target)
		}
		return value.Uint(), true, nil
	case reflect.Float32, reflect.Float64:
		floating := value.Float()
		overflow := floating > float64(unsignedMax(target.Bits()))
		if target.Bits() >= 64 {
			// float64(MaxUint64) rounds to 2^64, so compare against the
			// exclusive upper bound instead of the rounded integer maximum.
			overflow = floating >= math.Exp2(64)
		}
		if math.IsNaN(floating) || math.IsInf(floating, 0) || floating < 0 ||
			math.Trunc(floating) != floating || overflow {
			return 0, true, fmt.Errorf("value %v overflows %s", floating, target)
		}
		return uint64(floating), true, nil
	}
	return 0, false, nil
}

func unsignedMax(bits int) uint64 {
	if bits >= 64 {
		return math.MaxUint64
	}
	return 1<<bits - 1
}

func (g *MapGeneralizer) GetType(obj any) (typ string, err error) {
	typ, err = hessian2.GetJavaName(obj)
	// no error or error is not NilError
	if err == nil || err != hessian2.NilError {
		return
	}

	typ = "java.lang.Object"
	if err == hessian2.NilError {
		logger.Debugf("[Filter][Generic] the type of nil object couldn't be inferred, use the default value, type=%q", typ)
		return
	}

	logger.Debugf("[Filter][Generic] the type of object couldn't be recognized as a POJO, use the default value, objType=%T type=%q", obj, typ)
	return
}

// getGenericIncludeClass retrieves "generic.include.class" config value (fallback to true)
func getGenericIncludeClass() bool {
	cfgList := config.GetEnvInstance().Configuration()
	for e := cfgList.Front(); e != nil; e = e.Next() {
		conf, ok := e.Value.(*config.InmemoryConfiguration)
		if !ok {
			continue
		}

		if exist, val := conf.GetProperty(constant.GenericIncludeClassKey); exist {
			parsed, err := strconv.ParseBool(val)
			if err != nil {
				logger.Warnf("[Filter][Generic] generic.include.class value is invalid, fallback to true, val=%q", val)
				return true
			}
			return parsed
		}
	}

	return true
}

// removeClass recursively removes "class" key from data (returns new copy, no original modify)
// obj: any data (map[string]any/map[any]any/[]any/basic type)
func removeClass(obj any) any {
	switch v := obj.(type) {
	case map[string]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			if k == "class" {
				continue
			}
			m[k] = removeClass(val)
		}
		return m
	case map[any]any:
		m := make(map[any]any, len(v))
		for k, val := range v {
			if key, ok := k.(string); ok && key == "class" {
				continue
			}
			m[k] = removeClass(val)
		}
		return m
	case []any:
		s := make([]any, 0, len(v))
		for _, val := range v {
			s = append(s, removeClass(val))
		}
		return s
	default:
		return obj
	}
}

// objToMap converts an object(any) to a map
func objToMap(obj any) (any, error) {
	if obj == nil {
		return obj, nil
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	// if obj is a POJO, get the struct from the pointer (if it is a pointer)
	pojo, isPojo := obj.(hessian.POJO)
	if isPojo {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
			v = v.Elem()
		}
	}

	switch t.Kind() {
	case reflect.Struct:
		result := make(map[string]any, t.NumField())
		if isPojo {
			result["class"] = pojo.JavaClassName()
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			tag := genericfield.ParseMTag(field)
			if tag.Ignore || tag.OmitEmpty && isEmptyValue(value) {
				continue
			}
			kind := value.Kind()
			if !value.CanInterface() {
				logger.Debugf("[Filter][Generic] objToMap is skipped because it couldn't be converted to interface, field=%v", field)
				continue
			}
			valueIface := value.Interface()
			var generalizedValue any
			var err error
			switch kind {
			case reflect.Pointer:
				if value.IsNil() {
					generalizedValue = nil
					break
				}
				generalizedValue, err = objToMap(valueIface)
			case reflect.Struct, reflect.Slice, reflect.Map:
				if isPrimitive(valueIface) {
					logger.Warnf("[Filter][Generic] %q is primitive. Cross-language transfer (e.g., dubbo-go <-> dubbo-java) may crash. Use basic types like string.", value.Type())
					generalizedValue = valueIface
					break
				}

				generalizedValue, err = objToMap(valueIface)
			default:
				generalizedValue = valueIface
			}
			if err != nil {
				return nil, err
			}
			if tag.Squash {
				squashed, ok := generalizedValue.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("cannot squash non-struct type '%s'", value.Type())
				}
				delete(squashed, "class")
				maps.Copy(result, squashed)
				continue
			}
			result[tag.Name] = generalizedValue
		}
		return result, nil
	case reflect.Array, reflect.Slice:
		value := reflect.ValueOf(obj)
		newTemps := make([]any, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			newTemp, err := objToMap(value.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			newTemps = append(newTemps, newTemp)
		}
		return newTemps, nil
	case reflect.Map:
		newTempMap := make(map[any]any, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			if !iter.Value().CanInterface() {
				continue
			}
			key := iter.Key()
			mapV := iter.Value().Interface()
			generalizedValue, err := objToMap(mapV)
			if err != nil {
				return nil, err
			}
			newTempMap[mapKey(key)] = generalizedValue
		}
		return newTempMap, nil
	case reflect.Pointer:
		return objToMap(v.Elem().Interface())
	default:
		return obj, nil
	}
}

// mapKey converts the map key to interface type
func mapKey(key reflect.Value) any {
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

func isEmptyValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return value.IsNil()
	default:
		return false
	}
}

// isPrimitive determines if the object is primitive
func isPrimitive(obj any) bool {
	if _, ok := obj.(time.Time); ok {
		return true
	}
	return false
}

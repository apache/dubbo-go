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
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
)

// JavaBeanDescriptor type constants
const (
	TypeClass      = 1
	TypeEnum       = 2
	TypeCollection = 3
	TypeMap        = 4
	TypeArray      = 5
	TypePrimitive  = 6
	TypeBean       = 7
)

// JavaBeanDescriptor corresponds to org.apache.dubbo.common.beanutil.JavaBeanDescriptor
type JavaBeanDescriptor struct {
	ClassName  string      `json:"className"`
	Type       int         `json:"type"`
	Properties map[any]any `json:"properties"`
}

// JavaClassName implements hessian.POJO
func (d *JavaBeanDescriptor) JavaClassName() string {
	return "org.apache.dubbo.common.beanutil.JavaBeanDescriptor"
}

func init() {
	hessian.RegisterPOJO(&JavaBeanDescriptor{})
}

var (
	beanGeneralizer     Generalizer
	beanGeneralizerOnce sync.Once
)

func GetBeanGeneralizer() Generalizer {
	beanGeneralizerOnce.Do(func() {
		beanGeneralizer = &BeanGeneralizer{}
	})
	return beanGeneralizer
}

// BeanGeneralizer converts objects to/from JavaBeanDescriptor format
type BeanGeneralizer struct{}

func (g *BeanGeneralizer) Generalize(obj any) (any, error) {
	if obj == nil {
		return nil, nil
	}
	return g.toDescriptor(obj), nil
}

func (g *BeanGeneralizer) Realize(obj any, typ reflect.Type) (any, error) {
	if obj == nil {
		return nil, nil
	}
	return GetMapGeneralizer().Realize(g.fromDescriptor(obj), typ)
}

func (g *BeanGeneralizer) GetType(obj any) (typ string, err error) {
	typ, err = hessian2.GetJavaName(obj)
	// no error or error is not NilError
	if err == nil || err != hessian2.NilError {
		return
	}

	// fallback to java.lang.Object for nil or unrecognized types
	typ = "java.lang.Object"
	err = nil
	return
}

// toDescriptor converts Go object to JavaBeanDescriptor
func (g *BeanGeneralizer) toDescriptor(obj any) *JavaBeanDescriptor {
	if obj == nil {
		return nil
	}

	v := reflect.ValueOf(obj)
	// Handle typed nil pointer (e.g., (*T)(nil) stored in an any)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	t := v.Type()

	className := "java.lang.Object"
	if pojo, ok := obj.(hessian.POJO); ok {
		className = pojo.JavaClassName()
	}

	desc := &JavaBeanDescriptor{ClassName: className, Properties: make(map[any]any)}

	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		desc.Type = TypePrimitive
		desc.Properties["value"] = v.Interface()

	case reflect.Slice, reflect.Array:
		desc.Type = TypeArray
		for i := 0; i < v.Len(); i++ {
			desc.Properties[i] = g.toDescriptor(v.Index(i).Interface())
		}

	case reflect.Map:
		desc.Type = TypeMap
		iter := v.MapRange()
		for iter.Next() {
			desc.Properties[g.toDescriptor(iter.Key().Interface())] = g.toDescriptor(iter.Value().Interface())
		}

	case reflect.Struct:
		desc.Type = TypeBean
		for i := 0; i < t.NumField(); i++ {
			if fv := v.Field(i); fv.CanInterface() {
				desc.Properties[toUnexport(t.Field(i).Name)] = g.toDescriptor(fv.Interface())
			}
		}

	default:
		desc.Type = TypeBean
	}

	return desc
}

// fromDescriptor converts JavaBeanDescriptor to map for MapGeneralizer.Realize
func (g *BeanGeneralizer) fromDescriptor(obj any) any {
	desc, ok := obj.(*JavaBeanDescriptor)
	if !ok {
		if m, ok := obj.(map[any]any); ok {
			desc = &JavaBeanDescriptor{Properties: make(map[any]any)}
			if v, ok := m["className"].(string); ok {
				desc.ClassName = v
			}
			if v, ok := toInt(m["type"]); ok {
				desc.Type = v
			}
			if v, ok := m["properties"].(map[any]any); ok {
				desc.Properties = v
			}
		} else {
			return obj
		}
	}

	switch desc.Type {
	case TypePrimitive:
		return desc.Properties["value"]
	case TypeEnum:
		return desc.Properties["name"]
	case TypeArray, TypeCollection:
		// Normalize numeric keys (int, int32, int64) to indices when rebuilding arrays/collections.
		// Java Hessian decodes integer keys as int32, so we need to handle type conversion.
		maxIndex := -1
		for k := range desc.Properties {
			if idx, ok := toInt(k); ok && idx > maxIndex {
				maxIndex = idx
			}
		}
		if maxIndex < 0 {
			return []any{}
		}
		result := make([]any, maxIndex+1)
		for k, v := range desc.Properties {
			if idx, ok := toInt(k); ok && idx >= 0 && idx < len(result) {
				result[idx] = g.fromDescriptor(v)
			}
		}
		return result
	case TypeMap:
		result := make(map[any]any)
		for k, v := range desc.Properties {
			result[g.fromDescriptor(k)] = g.fromDescriptor(v)
		}
		return result
	case TypeBean:
		result := make(map[string]any)
		if desc.ClassName != "" {
			result["class"] = desc.ClassName
		}
		for k, v := range desc.Properties {
			if ks, ok := k.(string); ok {
				result[ks] = g.fromDescriptor(v)
			}
		}
		return result
	}
	return obj
}

// toInt converts various integer types to int.
// Returns the converted value and true if successful, otherwise 0 and false.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

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

func (g *BeanGeneralizer) GetType(obj any) (string, error) {
	return hessian2.GetJavaName(obj)
}

// toDescriptor converts Go object to JavaBeanDescriptor
func (g *BeanGeneralizer) toDescriptor(obj any) *JavaBeanDescriptor {
	if obj == nil {
		return nil
	}

	v := reflect.ValueOf(obj)
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
				name := t.Field(i).Name
				desc.Properties[string(name[0]|0x20)+name[1:]] = g.toDescriptor(fv.Interface())
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
			if v, ok := m["type"].(int32); ok {
				desc.Type = int(v)
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
		result := make([]any, len(desc.Properties))
		for i := 0; i < len(desc.Properties); i++ {
			result[i] = g.fromDescriptor(desc.Properties[i])
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

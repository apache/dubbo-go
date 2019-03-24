// Copyright (c) 2016 ~ 2019, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hessian

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	InvalidJavaEnum JavaEnum = -1
)

// !!! Pls attention that Every field name should be upper case.
// Otherwise the app may panic.
type POJO interface {
	JavaClassName() string // 获取对应的java classs的package name
}

type POJOEnum interface {
	POJO
	String() string
	EnumValue(string) JavaEnum
}

type JavaEnum int32

type JavaEnumClass struct {
	name string
}

type classInfo struct {
	javaName      string
	fieldNameList []string
	buffer        []byte // encoded buffer
}

type structInfo struct {
	typ      reflect.Type
	goName   string
	javaName string
	index    int // classInfoList index
	inst     interface{}
}

type POJORegistry struct {
	sync.RWMutex
	classInfoList []classInfo           // {class name, field name list...} list
	j2g           map[string]string     // java class name --> go struct name
	registry      map[string]structInfo // go class name --> go struct info
}

var (
	pojoRegistry = POJORegistry{
		j2g:      make(map[string]string),
		registry: make(map[string]structInfo),
	}
	pojoType     = reflect.TypeOf((*POJO)(nil)).Elem()
	javaEnumType = reflect.TypeOf((*POJOEnum)(nil)).Elem()
)

// 解析struct
func showPOJORegistry() {
	pojoRegistry.Lock()
	for k, v := range pojoRegistry.registry {
		fmt.Println("-->> show Registered types <<----")
		fmt.Println(k, v)
	}
	pojoRegistry.Unlock()
}

// Register a POJO instance.
// The return value is -1 if @o has been registered.
//
// # definition for an object (compact map)
// class-def  ::= 'C' string int string*
func RegisterPOJO(o POJO) int {
	var (
		ok bool
		b  []byte
		i  int
		n  int
		f  string
		l  []string
		t  structInfo
		c  classInfo
		v  reflect.Value
	)

	pojoRegistry.Lock()
	defer pojoRegistry.Unlock()
	if _, ok = pojoRegistry.registry[o.JavaClassName()]; !ok {
		v = reflect.ValueOf(o)
		switch v.Kind() {
		case reflect.Struct:
			t.typ = v.Type()
		case reflect.Ptr:
			t.typ = v.Elem().Type()
		default:
			t.typ = reflect.TypeOf(o)
		}
		t.goName = t.typ.String()
		t.javaName = o.JavaClassName()
		t.inst = o
		pojoRegistry.j2g[t.javaName] = t.goName

		b = b[:0]
		b = encByte(b, BC_OBJECT_DEF)
		b = encString(t.javaName, b)
		l = l[:0]
		n = t.typ.NumField()
		b = encInt32(int32(n), b)
		for i = 0; i < n; i++ {
			f = strings.ToLower(t.typ.Field(i).Name)
			l = append(l, f)
			b = encString(f, b)
		}

		c = classInfo{javaName: t.javaName, fieldNameList: l}
		c.buffer = append(c.buffer, b[:]...)
		t.index = len(pojoRegistry.classInfoList)
		pojoRegistry.classInfoList = append(pojoRegistry.classInfoList, c)
		pojoRegistry.registry[t.goName] = t
		i = t.index
	} else {
		i = -1
	}

	return i
}

// Register a value type JavaEnum variable.
func RegisterJavaEnum(o POJOEnum) int {
	var (
		ok bool
		b  []byte
		i  int
		n  int
		f  string
		l  []string
		t  structInfo
		c  classInfo
		v  reflect.Value
	)

	pojoRegistry.Lock()
	defer pojoRegistry.Unlock()
	if _, ok = pojoRegistry.registry[o.JavaClassName()]; !ok {
		v = reflect.ValueOf(o)
		switch v.Kind() {
		case reflect.Struct:
			t.typ = v.Type()
		case reflect.Ptr:
			t.typ = v.Elem().Type()
		default:
			t.typ = reflect.TypeOf(o)
		}
		t.goName = t.typ.String()
		t.javaName = o.JavaClassName()
		t.inst = o
		pojoRegistry.j2g[t.javaName] = t.goName

		b = b[:0]
		b = encByte(b, BC_OBJECT_DEF)
		b = encString(t.javaName, b)
		l = l[:0]
		n = 1
		b = encInt32(int32(n), b)
		f = strings.ToLower("name")
		l = append(l, f)
		b = encString(f, b)

		c = classInfo{javaName: t.javaName, fieldNameList: l}
		c.buffer = append(c.buffer, b[:]...)
		t.index = len(pojoRegistry.classInfoList)
		pojoRegistry.classInfoList = append(pojoRegistry.classInfoList, c)
		pojoRegistry.registry[t.goName] = t
		i = t.index
	} else {
		i = -1
	}

	return i
}

// check if go struct name @goName has been registered or not.
func checkPOJORegistry(goName string) (int, bool) {
	var (
		ok bool
		s  structInfo
	)
	pojoRegistry.RLock()
	s, ok = pojoRegistry.registry[goName]
	pojoRegistry.RUnlock()

	return s.index, ok
}

// @typeName is class's java name
func getStructInfo(javaName string) (structInfo, bool) {
	var (
		ok bool
		g  string
		s  structInfo
	)

	pojoRegistry.RLock()
	g, ok = pojoRegistry.j2g[javaName]
	if ok {
		s, ok = pojoRegistry.registry[g]
	}
	pojoRegistry.RUnlock()

	return s, ok
}

func getStructDefByIndex(idx int) (reflect.Type, classInfo, error) {
	var (
		ok      bool
		clsName string
		cls     classInfo
		s       structInfo
	)

	pojoRegistry.RLock()
	defer pojoRegistry.RUnlock()

	if len(pojoRegistry.classInfoList) <= idx || idx < 0 {
		return nil, cls, jerrors.Errorf("illegal class index @idx %d", idx)
	}
	cls = pojoRegistry.classInfoList[idx]
	clsName = pojoRegistry.j2g[cls.javaName]
	s, ok = pojoRegistry.registry[clsName]
	if !ok {
		return nil, cls, jerrors.Errorf("can not find go type name %s in registry", clsName)
	}

	return s.typ, cls, nil
}

// Create a new instance by its struct name is @goName.
// the return value is nil if @o has been registered.
func createInstance(goName string) interface{} {
	var (
		ok bool
		s  structInfo
	)

	pojoRegistry.RLock()
	s, ok = pojoRegistry.registry[goName]
	pojoRegistry.RUnlock()
	if !ok {
		return nil
	}

	return reflect.New(s.typ).Interface()
}

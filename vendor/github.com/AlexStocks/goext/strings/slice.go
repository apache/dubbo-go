// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// http://blog.csdn.net/siddontang/article/details/23541587
// reflect.StringHeader和reflect.SliceHeader的结构体只相差末尾一个字段(cap)
// vitess代码，一种很hack的做法，string和slice的转换只需要拷贝底层的指针，而不是内存拷贝。
package gxstrings

import (
	"reflect"
	"unsafe"
)

func String(b []byte) (s string) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pstring.Data = pbytes.Data
	pstring.Len = pbytes.Len
	return
}

func Slice(s string) (b []byte) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pbytes.Data = pstring.Data
	pbytes.Len = pstring.Len
	pbytes.Cap = pstring.Len
	return
}

// returns &s[0], which is not allowed in go
func StringPointer(s string) unsafe.Pointer {
	p := (*reflect.StringHeader)(unsafe.Pointer(&s))
	return unsafe.Pointer(p.Data)
}

// returns &b[0], which is not allowed in go
func BytePointer(b []byte) unsafe.Pointer {
	p := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	return unsafe.Pointer(p.Data)
}

// var (
// 	typeOfBytes = reflect.TypeOf([]byte(nil))
// )

// func CheckByteArray1(v interface{}) bool {
// 	vv := reflect.ValueOf(v)
// 	switch vv.Kind() {
// 	case reflect.Slice:
// 		if vv.Type() == typeOfBytes {
// 			return true
// 		}
// 	}
//
// 	return false
// }

// func CheckByteArray2(v interface{}) bool {
// 	_, ok := v.([]byte)
// 	return ok
// }

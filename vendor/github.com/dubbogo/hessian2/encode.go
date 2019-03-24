/*
 *
 *  * Copyright 2012-2016 Viant.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 *  * use this file except in compliance with the License. You may obtain a copy of
 *  * the License at
 *  *
 *  * http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 *  * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 *  * License for the specific language governing permissions and limitations under
 *  * the License.
 *
 */

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
	"bytes"
	"math"
	"reflect"
	"time"
	"unicode/utf8"
	"unsafe"
)

import (
	jerrors "github.com/juju/errors"
)

// used to ref object,list,map
type _refElem struct {
	// record the kind of target, objects are the same only if the address and kind are the same
	kind  reflect.Kind

	// ref index
	index int
}

// nil bool int8 int32 int64 float32 float64 time.Time
// string []byte []interface{} map[interface{}]interface{}
// array object struct

type Encoder struct {
	classInfoList []classInfo
	buffer        []byte
	refMap        map[unsafe.Pointer]_refElem
}

func NewEncoder() *Encoder {
	var buffer = make([]byte, 64)

	return &Encoder{
		buffer: buffer[:0],
		refMap: make(map[unsafe.Pointer]_refElem, 7),
	}
}

func (e *Encoder) Buffer() []byte {
	return e.buffer[:]
}

func (e *Encoder) Append(buf []byte) {
	e.buffer = append(e.buffer, buf[:]...)
}

// If @v can not be encoded, the return value is nil. At present only struct may can not be encoded.
func (e *Encoder) Encode(v interface{}) error {
	if v == nil {
		e.buffer = encNull(e.buffer)
		return nil
	}

	switch v.(type) {
	case nil:
		e.buffer = encNull(e.buffer)
		return nil

	case bool:
		e.buffer = encBool(v.(bool), e.buffer)

	case int8:
		e.buffer = encInt32(v.(int32), e.buffer)

	case int16:
		e.buffer = encInt32(v.(int32), e.buffer)

	case int32:
		e.buffer = encInt32(v.(int32), e.buffer)

	case int:
		// if v.(int) >= -2147483648 && v.(int) <= 2147483647 {
		// 	b = encInt32(int32(v.(int)), b)
		// } else {
		// 	b = encInt64(int64(v.(int)), b)
		// }
		// 把int统一按照int64处理，这样才不会导致decode的时候出现" reflect: Call using int32 as type int64 [recovered]"这种panic
		e.buffer = encInt64(int64(v.(int)), e.buffer)

	case int64:
		e.buffer = encInt64(v.(int64), e.buffer)

	case time.Time:
		e.buffer = encDateInMs(v.(time.Time), e.buffer)
		// e.buffer = encDateInMimute(v.(time.Time), e.buffer)

	case float32:
		e.buffer = encFloat(float64(v.(float32)), e.buffer)

	case float64:
		e.buffer = encFloat(v.(float64), e.buffer)

	case string:
		e.buffer = encString(v.(string), e.buffer)

	case []byte:
		e.buffer = encBinary(v.([]byte), e.buffer)

	case map[interface{}]interface{}:
		return e.encUntypedMap(v.(map[interface{}]interface{}))

	default:
		t := UnpackPtrType(reflect.TypeOf(v))
		switch t.Kind() {
		case reflect.Struct:
			if p, ok := v.(POJO); ok {
				return e.encStruct(p)
			} else {
				return jerrors.Errorf("struct type not Support! %s[%v] is not a instance of POJO!", t.String(), v)
			}
		case reflect.Slice, reflect.Array:
			return e.encUntypedList(v)
		case reflect.Map: // 进入这个case，就说明map可能是map[string]int这种类型
			return e.encMap(v)
		default:
			return jerrors.Errorf("type not supported! %s", t.Kind().String())
		}
	}

	return nil
}

//=====================================
//对各种数据类型的编码
//=====================================

func encByte(b []byte, t ...byte) []byte {
	return append(b, t...)
}

//encRef encode ref index
func encRef(b []byte, index int) []byte {
	return encInt32(int32(index), append(b, BC_REF))
}

/////////////////////////////////////////
// Null
/////////////////////////////////////////
func encNull(b []byte) []byte {
	return append(b, BC_NULL)
}

/////////////////////////////////////////
// Bool
/////////////////////////////////////////

// # boolean true/false
// ::= 'T'
// ::= 'F'
func encBool(v bool, b []byte) []byte {
	var c byte = BC_FALSE
	if v == true {
		c = BC_TRUE
	}

	return append(b, c)
}

/////////////////////////////////////////
// Int32
/////////////////////////////////////////

// # 32-bit signed integer
// ::= 'I' b3 b2 b1 b0
// ::= [x80-xbf]             # -x10 to x3f
// ::= [xc0-xcf] b0          # -x800 to x7ff
// ::= [xd0-xd7] b1 b0       # -x40000 to x3ffff
// hessian-lite/src/main/java/com/alibaba/com/alibaba/com/caucho/hessian/io/Hessian2Output.java:642 WriteInt
func encInt32(v int32, b []byte) []byte {
	if int32(INT_DIRECT_MIN) <= v && v <= int32(INT_DIRECT_MAX) {
		return encByte(b, byte(v+int32(BC_INT_ZERO)))
	} else if int32(INT_BYTE_MIN) <= v && v <= int32(INT_BYTE_MAX) {
		return encByte(b, byte(int32(BC_INT_BYTE_ZERO)+v>>8), byte(v))
	} else if int32(INT_SHORT_MIN) <= v && v <= int32(INT_SHORT_MAX) {
		return encByte(b, byte(v>>16+int32(BC_INT_SHORT_ZERO)), byte(v>>8), byte(v))
	}

	return encByte(b, byte('I'), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

/////////////////////////////////////////
// Int64
/////////////////////////////////////////

// # 64-bit signed long integer
// ::= 'L' b7 b6 b5 b4 b3 b2 b1 b0
// ::= [xd8-xef]             # -x08 to x0f
// ::= [xf0-xff] b0          # -x800 to x7ff
// ::= [x38-x3f] b1 b0       # -x40000 to x3ffff
// ::= x59 b3 b2 b1 b0       # 32-bit integer cast to long
// hessian-lite/src/main/java/com/alibaba/com/alibaba/com/caucho/hessian/io/Hessian2Output.java:642 WriteLong
func encInt64(v int64, b []byte) []byte {
	if int64(LONG_DIRECT_MIN) <= v && v <= int64(LONG_DIRECT_MAX) {
		return encByte(b, byte(v+int64(BC_LONG_ZERO)))
	} else if int64(LONG_BYTE_MIN) <= v && v <= int64(LONG_BYTE_MAX) {
		return encByte(b, byte(int64(BC_LONG_BYTE_ZERO)+(v>>8)), byte(v))
	} else if int64(LONG_SHORT_MIN) <= v && v <= int64(LONG_SHORT_MAX) {
		return encByte(b, byte(int64(BC_LONG_SHORT_ZERO)+(v>>16)), byte(v>>8), byte(v))
	} else if 0x80000000 <= v && v <= 0x7fffffff {
		return encByte(b, BC_LONG_INT, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}

	return encByte(b, 'L', byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

/////////////////////////////////////////
// Date
/////////////////////////////////////////

// # time in UTC encoded as 64-bit long milliseconds since epoch
// ::= x4a b7 b6 b5 b4 b3 b2 b1 b0
// ::= x4b b3 b2 b1 b0       # minutes since epoch
func encDateInMs(v time.Time, b []byte) []byte {
	b = append(b, BC_DATE)
	return append(b, PackInt64(v.UnixNano()/1e6)...)
}

func encDateInMimute(v time.Time, b []byte) []byte {
	b = append(b, BC_DATE_MINUTE)
	return append(b, PackInt32(int32(v.UnixNano()/60e9))...)
}

/////////////////////////////////////////
// Double
/////////////////////////////////////////

// # 64-bit IEEE double
// ::= 'D' b7 b6 b5 b4 b3 b2 b1 b0
// ::= x5b                   # 0.0
// ::= x5c                   # 1.0
// ::= x5d b0                # byte cast to double (-128.0 to 127.0)
// ::= x5e b1 b0             # short cast to double
// ::= x5f b3 b2 b1 b0       # 32-bit float cast to double
func encFloat(v float64, b []byte) []byte {
	fv := float64(int64(v))
	if fv == v {
		iv := int64(v)
		switch iv {
		case 0:
			return encByte(b, BC_DOUBLE_ZERO)
		case 1:
			return encByte(b, BC_DOUBLE_ONE)
		}
		if iv >= -0x80 && iv < 0x80 {
			return encByte(b, BC_DOUBLE_BYTE, byte(iv))
		} else if iv >= -0x8000 && iv < 0x8000 {
			return encByte(b, BC_DOUBLE_BYTE, byte(iv>>8), byte(iv))
		}

		goto END
	}

END:
	bits := uint64(math.Float64bits(v))
	return encByte(b, BC_DOUBLE, byte(bits>>56), byte(bits>>48), byte(bits>>40),
		byte(bits>>32), byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

/////////////////////////////////////////
// String
/////////////////////////////////////////

func Slice(s string) (b []byte) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pbytes.Data = pstring.Data
	pbytes.Len = pstring.Len
	pbytes.Cap = pstring.Len
	return
}

// # UTF-8 encoded character string split into 64k chunks
// ::= x52 b1 b0 <utf8-data> string  # non-final chunk
// ::= 'S' b1 b0 <utf8-data>         # string of length 0-65535
// ::= [x00-x1f] <utf8-data>         # string of length 0-31
// ::= [x30-x34] <utf8-data>         # string of length 0-1023
func encString(v string, b []byte) []byte {
	var (
		vBuf = *bytes.NewBufferString(v)
		vLen = utf8.RuneCountInString(v)

		vChunk = func(length int) {
			for i := 0; i < length; i++ {
				if r, s, err := vBuf.ReadRune(); s > 0 && err == nil {
					// b = append(b, []byte(string(r))...)
					b = append(b, Slice(string(r))...) // 直接基于r的内存空间把它转换为[]byte
				}
			}
		}
	)

	if v == "" {
		return encByte(b, BC_STRING_DIRECT)
	}

	for {
		vLen = utf8.RuneCount(vBuf.Bytes())
		if vLen == 0 {
			break
		}
		if vLen > CHUNK_SIZE {
			b = encByte(b, BC_STRING_CHUNK)
			b = encByte(b, PackUint16(uint16(CHUNK_SIZE))...)
			vChunk(CHUNK_SIZE)
		} else {
			if vLen <= int(STRING_DIRECT_MAX) {
				b = encByte(b, byte(vLen+int(BC_STRING_DIRECT)))
			} else if vLen <= int(STRING_SHORT_MAX) {
				b = encByte(b, byte((vLen>>8)+int(BC_STRING_SHORT)), byte(vLen))
			} else {
				b = encByte(b, BC_STRING)
				b = encByte(b, PackUint16(uint16(vLen))...)
			}
			vChunk(vLen)
		}
	}

	return b
}

/////////////////////////////////////////
// Binary, []byte
/////////////////////////////////////////

// # 8-bit binary data split into 64k chunks
// ::= x41(A) b1 b0 <binary-data> binary # non-final chunk
// ::= x42(B) b1 b0 <binary-data>        # final chunk
// ::= [x20-x2f] <binary-data>           # binary data of length 0-15
// ::= [x34-x37] <binary-data>           # binary data of length 0-1023
func encBinary(v []byte, b []byte) []byte {
	var (
		length  uint16
		vLength int
	)

	if len(v) == 0 {
		//return encByte(b, BC_BINARY_DIRECT)
		return encByte(b, BC_NULL)
	}

	vLength = len(v)
	for vLength > 0 {
		if vLength > CHUNK_SIZE {
			length = CHUNK_SIZE
			b = encByte(b, byte(BC_BINARY_CHUNK), byte(length>>8), byte(length))
		} else {
			length = uint16(vLength)
			if vLength <= int(BINARY_DIRECT_MAX) {
				b = encByte(b, byte(int(BC_BINARY_DIRECT)+vLength))
			} else if vLength <= int(BINARY_SHORT_MAX) {
				b = encByte(b, byte(int(BC_BINARY_SHORT)+vLength>>8), byte(vLength))
			} else {
				b = encByte(b, byte(BC_BINARY), byte(vLength>>8), byte(vLength))
			}
		}

		b = append(b, v[:length]...)
		v = v[length:]
		vLength = len(v)
	}

	return b
}

/////////////////////////////////////////
// List
/////////////////////////////////////////

// # list/vector
// ::= x55 type value* 'Z'   # variable-length list
// ::= 'V' type int value*   # fixed-length list
// ::= x57 value* 'Z'        # variable-length untyped list
// ::= x58 int value*        # fixed-length untyped list
// ::= [x70-77] type value*  # fixed-length typed list
// ::= [x78-7f] value*       # fixed-length untyped list
func (e *Encoder) encUntypedList(v interface{}) error {
	var (
		err error
	)

	value := reflect.ValueOf(v)

	// check ref
	if n, ok := e.checkRefMap(value); ok {
		e.buffer = encRef(e.buffer, n)
		return nil
	}

	e.buffer = encByte(e.buffer, BC_LIST_FIXED_UNTYPED) // x58
	e.buffer = encInt32(int32(value.Len()), e.buffer)
	for i := 0; i < value.Len(); i++ {
		if err = e.Encode(value.Index(i).Interface()); err != nil {
			return err
		}
	}

	return nil
}

/////////////////////////////////////////
// map/object
/////////////////////////////////////////

// ::= 'M' type (value value)* 'Z'  # key, value map pairs
// ::= 'H' (value value)* 'Z'       # untyped key, value
func (e *Encoder) encUntypedMap(m map[interface{}]interface{}) error {
	if len(m) == 0 {
		return nil
	}

	// check ref
	if n, ok := e.checkRefMap(reflect.ValueOf(m)); ok {
		e.buffer = encRef(e.buffer, n)
		return nil
	}

	var err error
	e.buffer = encByte(e.buffer, BC_MAP_UNTYPED)
	for k, v := range m {
		if err = e.Encode(k); err != nil {
			return err
		}
		if err = e.Encode(v); err != nil {
			return err
		}
	}

	e.buffer = encByte(e.buffer, BC_END) // 'Z'

	return nil
}

func getMapKey(key reflect.Value, t reflect.Type) (interface{}, error) {
	switch t.Kind() {
	case reflect.Bool:
		return key.Bool(), nil

	case reflect.Int8:
		return int8(key.Int()), nil
	case reflect.Int16:
		return int16(key.Int()), nil
	case reflect.Int32:
		return int32(key.Int()), nil
	case reflect.Int:
		return int(key.Int()), nil
	case reflect.Int64:
		return key.Int(), nil

	case reflect.Uint8:
		return byte(key.Uint()), nil
	case reflect.Uint16:
		return uint16(key.Uint()), nil
	case reflect.Uint32:
		return uint32(key.Uint()), nil
	case reflect.Uint:
		return uint(key.Uint()), nil
	case reflect.Uint64:
		return key.Uint(), nil

	case reflect.Float32:
		return float32(key.Float()), nil
	case reflect.Float64:
		return float64(key.Float()), nil

	case reflect.Uintptr:
		return key.UnsafeAddr(), nil

	case reflect.String:
		return key.String(), nil
	}

	return nil, jerrors.Errorf("unsupported map key kind %s", t.Kind().String())
}

func (e *Encoder) encMap(m interface{}) error {
	var (
		err   error
		k     interface{}
		typ   reflect.Type
		value reflect.Value
		keys  []reflect.Value
	)

	value = reflect.ValueOf(m)

	// check ref
	if n, ok := e.checkRefMap(value); ok {
		e.buffer = encRef(e.buffer, n)
		return nil
	}

	value = UnpackPtrValue(value)
	// check nil map
	if value.Kind() == reflect.Ptr && !value.Elem().IsValid() {
		e.buffer = encNull(e.buffer)
		return nil
	}

	keys = value.MapKeys()
	if len(keys) == 0 {
		// fix: set nil for empty map
		e.buffer = encNull(e.buffer)
		return nil
	}

	typ = value.Type().Key()
	e.buffer = encByte(e.buffer, BC_MAP_UNTYPED)
	for i := 0; i < len(keys); i++ {
		k, err = getMapKey(keys[i], typ)
		if err != nil {
			return jerrors.Annotatef(err, "getMapKey(idx:%d, key:%+v)", i, keys[i])
		}
		if err = e.Encode(k); err != nil {
			return jerrors.Annotatef(err, "failed to encode map key(idx:%d, key:%+v)", i, keys[i])
		}
		entryValueObj := value.MapIndex(keys[i]).Interface()
		if err = e.Encode(entryValueObj); err != nil {
			return jerrors.Annotatef(err, "failed to encode map value(idx:%d, key:%+v, value:%+v)", i, k, entryValueObj)
		}
	}
	e.buffer = encByte(e.buffer, BC_END)

	return nil
}

// get @v go struct name
func typeof(v interface{}) string {
	return reflect.TypeOf(v).String()
}

/////////////////////////////////////////
// map/object
/////////////////////////////////////////

//class-def  ::= 'C' string int string* //  mandatory type string, the number of fields, and the field names.
//object     ::= 'O' int value* // class-def id, value list
//           ::= [x60-x6f] value* // class-def id, value list
//
//Object serialization
//
//class Car {
//  String color;
//  String model;
//}
//
//out.writeObject(new Car("red", "corvette"));
//out.writeObject(new Car("green", "civic"));
//
//---
//
//C                        # object definition (#0)
//  x0b example.Car        # type is example.Car
//  x92                    # two fields
//  x05 color              # color field name
//  x05 model              # model field name
//
//O                        # object def (long form)
//  x90                    # object definition #0
//  x03 red                # color field value
//  x08 corvette           # model field value
//
//x60                      # object def #0 (short form)
//  x05 green              # color field value
//  x05 civic              # model field value
//
//enum Color {
//  RED,
//  GREEN,
//  BLUE,
//}
//
//out.writeObject(Color.RED);
//out.writeObject(Color.GREEN);
//out.writeObject(Color.BLUE);
//out.writeObject(Color.GREEN);
//
//---
//
//C                         # class definition #0
//  x0b example.Color       # type is example.Color
//  x91                     # one field
//  x04 name                # enumeration field is "name"
//
//x60                       # object #0 (class def #0)
//  x03 RED                 # RED value
//
//x60                       # object #1 (class def #0)
//  x90                     # object definition ref #0
//  x05 GREEN               # GREEN value
//
//x60                       # object #2 (class def #0)
//  x04 BLUE                # BLUE value
//
//x51 x91                   # object ref #1, i.e. Color.GREEN
func (e *Encoder) encStruct(v POJO) error {
	var (
		ok     bool
		i      int
		idx    int
		num    int
		err    error
		clsDef classInfo
	)

	vv := reflect.ValueOf(v)
	// check ref
	if n, ok := e.checkRefMap(vv); ok {
		e.buffer = encRef(e.buffer, n)
		return nil
	}

	vv = UnpackPtr(vv)
	// check nil pointer
	if !vv.IsValid() {
		e.buffer = encNull(e.buffer)
		return nil
	}

	// write object definition
	idx = -1
	for i = range e.classInfoList {
		if v.JavaClassName() == e.classInfoList[i].javaName {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx, ok = checkPOJORegistry(typeof(v))
		if !ok { // 不存在
			idx = RegisterPOJO(v)
		}
		_, clsDef, _ = getStructDefByIndex(idx)
		idx = len(e.classInfoList)
		e.classInfoList = append(e.classInfoList, clsDef)
		e.buffer = append(e.buffer, clsDef.buffer...)
	}

	// write object instance
	if byte(idx) <= OBJECT_DIRECT_MAX {
		e.buffer = encByte(e.buffer, byte(idx)+BC_OBJECT_DIRECT)
	} else {
		e.buffer = encByte(e.buffer, BC_OBJECT)
		e.buffer = encInt32(int32(idx), e.buffer)
	}

	num = vv.NumField()
	for i = 0; i < num; i++ {
		field := vv.Field(i)
		fieldName := field.Type().String()
		if err = e.Encode(field.Interface()); err != nil {
			return jerrors.Annotatef(err, "failed to encode field: %s, %+v", fieldName, field.Interface())
		}
	}

	return nil
}

// return the order number of ref object if found ,
// otherwise, add the object into the encode ref map
func (e *Encoder) checkRefMap(v reflect.Value) (int, bool) {
	var (
		kind reflect.Kind
		addr unsafe.Pointer
	)

	if v.Kind() == reflect.Ptr {
		for v.Elem().Kind() == reflect.Ptr {
			v = v.Elem()
		}
		kind = v.Elem().Kind()
		if kind == reflect.Slice || kind == reflect.Map {
			addr = unsafe.Pointer(v.Elem().Pointer())
		} else {
			addr = unsafe.Pointer(v.Pointer())
		}
	} else {
		kind = v.Kind()
		switch kind {
		case reflect.Slice, reflect.Map:
			addr = unsafe.Pointer(v.Pointer())
		default:
			addr = unsafe.Pointer(PackPtr(v).Pointer())
		}
	}

	if elem, ok := e.refMap[addr]; ok {
		// the array addr is equal to the first elem, which must ignore
		if elem.kind == kind {
			return elem.index, ok
		}
		return 0, false
	}

	n := len(e.refMap)
	e.refMap[addr] = _refElem{kind, n}
	return 0, false
}

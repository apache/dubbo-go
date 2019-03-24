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

/*
decoder implement hessian 2 protocol, It follows java hessian package standard.
It assume that you using the java name convention
baisca difference between java and go
fully qualify java class name is composed of package + class name
Go assume upper case of field name is exportable and java did not have that constrain
but in general java using camo camlecase. So it did conversion of field name from
the first letter of from upper to lower case
typMap{string]reflect.Type contain full java package+class name and go relfect.Type
must provide in order to correctly decode to galang interface
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
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

// _refHolder is used to record decode list, the address of which may change when appending more element.
type _refHolder struct {
	// destinations
	destinations []reflect.Value

	value reflect.Value
}

var _refHolderType = reflect.TypeOf(_refHolder{})

// change ref value
func (h *_refHolder) change(v reflect.Value) {
	if h.value.CanAddr() && v.CanAddr() && h.value.Pointer() == v.Pointer() {
		return
	}
	h.value = v
}

// notice all destinations ref to the value
func (h *_refHolder) notify() {
	for _, dest := range h.destinations {
		SetValue(dest, h.value)
	}
}

// add destination
func (h *_refHolder) add(dest reflect.Value) {
	h.destinations = append(h.destinations, dest)
}

type Decoder struct {
	reader        *bufio.Reader
	refs          []interface{}
	classInfoList []classInfo
}

var (
	ErrNotEnoughBuf    = jerrors.Errorf("not enough buf")
	ErrIllegalRefIndex = jerrors.Errorf("illegal ref index")
)

func NewDecoder(b []byte) *Decoder {
	return &Decoder{reader: bufio.NewReader(bytes.NewReader(b))}
}

/////////////////////////////////////////
// utilities
/////////////////////////////////////////

// 读取当前字节,指针不前移
func (d *Decoder) peekByte() byte {
	return d.peek(1)[0]
}

// 添加引用
func (d *Decoder) appendRefs(v interface{}) *_refHolder {
	var holder *_refHolder
	vv := EnsurePackValue(v)
	// only slice and array need ref holder , for its address changes when decoding
	if vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		holder = &_refHolder{
			value: vv,
		}
		// pack holder value
		v = reflect.ValueOf(holder)
	}

	d.refs = append(d.refs, v)
	return holder
}

// 获取缓冲长度
func (d *Decoder) len() int {
	d.peek(1) //需要先读一下资源才能得到已缓冲的长度
	return d.reader.Buffered()
}

// 读取 Decoder 结构中的一个字节,并后移一个字节
func (d *Decoder) readByte() (byte, error) {
	return d.reader.ReadByte()
}

// 前移一个字节
func (d *Decoder) unreadByte() error {
	return d.reader.UnreadByte()
}

// 读取指定长度的字节,并后移len(b)个字节
func (d *Decoder) next(b []byte) (int, error) {
	return d.reader.Read(b)
}

// 读取指定长度字节,指针不后移
func (d *Decoder) peek(n int) []byte {
	b, _ := d.reader.Peek(n)
	return b
}

// 读取len(s)的 utf8 字符
func (d *Decoder) nextRune(s []rune) []rune {
	var (
		n   int
		i   int
		r   rune
		ri  int
		err error
	)

	n = len(s)
	s = s[:0]
	for i = 0; i < n; i++ {
		if r, ri, err = d.reader.ReadRune(); err == nil && ri > 0 {
			s = append(s, r)
		}
	}

	return s
}

// 读取数据类型描述,用于 list 和 map
func (d *Decoder) decType() (string, error) {
	var (
		err error
		arr [1]byte
		buf []byte
		tag byte
		idx int32
		typ reflect.Type
	)

	buf = arr[:1]
	if _, err = io.ReadFull(d.reader, buf); err != nil {
		return "", jerrors.Trace(err)
	}
	tag = buf[0]
	if (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
		(tag >= 0x30 && tag <= 0x33) || (tag == BC_STRING) || (tag == BC_STRING_CHUNK) {
		return d.decString(int32(tag))
	}

	if idx, err = d.decInt32(TAG_READ); err != nil {
		return "", jerrors.Trace(err)
	}

	typ, _, err = d.getStructDefByIndex(int(idx))
	if err == nil {
		return typ.String(), nil
	}

	return "", err
}

// 解析 hessian 数据包
func (d *Decoder) Decode() (interface{}, error) {
	var (
		err error
		tag byte
	)

	tag, err = d.readByte()
	if err == io.EOF {
		return nil, err
	}

	switch {
	case tag == BC_END:
		// return EOF error for end flag 'Z'
		return nil, io.EOF

	case tag == BC_NULL: // 'N': //null
		return nil, nil

	case tag == BC_TRUE: // 'T': //true
		return true, nil

	case tag == BC_FALSE: //'F': //false
		return false, nil

	case tag == BC_REF: // 'R': //ref, 一个整数，用以指代前面的list 或者 map
		return d.decRef(int32(tag))

	case (0x80 <= tag && tag <= 0xbf) || (0xc0 <= tag && tag <= 0xcf) ||
		(0xd0 <= tag && tag <= 0xd7) || tag == BC_INT: //'I': //int
		return d.decInt32(int32(tag))

	case (tag >= 0xd8 && tag <= 0xef) || (tag >= 0xf0 && tag <= 0xff) ||
		(tag >= 0x38 && tag <= 0x3f) || (tag == BC_LONG_INT) || (tag == BC_LONG): //'L': //long
		return d.decInt64(int32(tag))

	case (tag == BC_DATE_MINUTE) || (tag == BC_DATE): //'d': //date
		return d.decDate(int32(tag))

	case (tag == BC_DOUBLE_ZERO) || (tag == BC_DOUBLE_ONE) || (tag == BC_DOUBLE_BYTE) ||
		(tag == BC_DOUBLE_SHORT) || (tag == BC_DOUBLE_MILL) || (tag == BC_DOUBLE): //'D': //double
		return d.decDouble(int32(tag))

	// case 'S', 's', 'X', 'x': //string,xml
	case (tag == BC_STRING_CHUNK || tag == BC_STRING) ||
		(tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
		(tag >= 0x30 && tag <= 0x33):
		return d.decString(int32(tag))

		// case 'B', 'b': //binary
	case (tag == BC_BINARY) || (tag == BC_BINARY_CHUNK) || (tag >= 0x20 && tag <= 0x2f) ||
		(tag >= BC_BINARY_SHORT && tag <= 0x3f):
		return d.decBinary(int32(tag))

	// case 'V': //list
	case (tag >= BC_LIST_DIRECT && tag <= 0x77) || (tag == BC_LIST_FIXED || tag == BC_LIST_VARIABLE) ||
		(tag >= BC_LIST_DIRECT_UNTYPED && tag <= 0x7f) ||
		(tag == BC_LIST_FIXED_UNTYPED || tag == BC_LIST_VARIABLE_UNTYPED):
		return d.decList(int32(tag))

	case (tag == BC_MAP) || (tag == BC_MAP_UNTYPED):
		return d.decMap(int32(tag))

	case (tag == BC_OBJECT_DEF) || (tag == BC_OBJECT) ||
		(BC_OBJECT_DIRECT <= tag && tag <= (BC_OBJECT_DIRECT+OBJECT_DIRECT_MAX)):
		return d.decObject(int32(tag))

	default:
		return nil, jerrors.Errorf("Invalid type: %v,>>%v<<<", string(tag), d.peek(d.len()))
	}
}

/////////////////////////////////////////
// Int32
/////////////////////////////////////////

// # 32-bit signed integer
// ::= 'I' b3 b2 b1 b0
// ::= [x80-xbf]             # -x10 to x3f
// ::= [xc0-xcf] b0          # -x800 to x7ff
// ::= [xd0-xd7] b1 b0       # -x40000 to x3ffff
func (d *Decoder) decInt32(flag int32) (int32, error) {
	var (
		err error
		tag byte
		buf [4]byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	//direct integer
	case tag >= 0x80 && tag <= 0xbf:
		return int32(tag - BC_INT_ZERO), nil

	case tag >= 0xc0 && tag <= 0xcf:
		if _, err = io.ReadFull(d.reader, buf[:1]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int32(tag-BC_INT_BYTE_ZERO)<<8 + int32(buf[0]), nil

	case tag >= 0xd0 && tag <= 0xd7:
		if _, err = io.ReadFull(d.reader, buf[:2]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int32(tag-BC_INT_SHORT_ZERO)<<16 + int32(buf[0])<<8 + int32(buf[1]), nil

	case tag == BC_INT:
		if _, err := io.ReadFull(d.reader, buf[:4]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int32(buf[0])<<24 + int32(buf[1])<<16 + int32(buf[2])<<8 + int32(buf[3]), nil

	default:
		return 0, jerrors.Errorf("decInt32 integer wrong tag:%#x", tag)
	}
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
// ref: hessian-lite/src/main/java/com/alibaba/com/caucho/hessian/io/Hessian2Input.java:readLong
func (d *Decoder) decInt64(flag int32) (int64, error) {
	var (
		err error
		tag byte
		buf [8]byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_NULL:
		return int64(0), nil

	case tag == BC_FALSE:
		return int64(0), nil

	case tag == BC_TRUE:
		return int64(1), nil

		// direct integer
	case tag >= 0x80 && tag <= 0xbf:
		return int64(tag - BC_INT_ZERO), nil

		// byte int
	case tag >= 0xc0 && tag <= 0xcf:
		if _, err = io.ReadFull(d.reader, buf[:1]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int64(tag-BC_INT_BYTE_ZERO)<<8 + int64(buf[0]), nil

		// short int
	case tag >= 0xd0 && tag <= 0xd7:
		if _, err = io.ReadFull(d.reader, buf[:2]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int64(tag-BC_INT_SHORT_ZERO)<<16 + int64(buf[0])<<8 + int64(buf[1]), nil

	case tag == BC_DOUBLE_BYTE:
		tag, _ = d.readByte()
		return int64(tag), nil

	case tag == BC_DOUBLE_SHORT:
		if _, err = io.ReadFull(d.reader, buf[:2]); err != nil {
			return 0, jerrors.Trace(err)
		}

		return int64(int(buf[0])<<8 + int(buf[1])), nil

	case tag == BC_INT: // 'I'
		i32, err := d.decInt32(TAG_READ)
		return int64(i32), err

	case tag == BC_LONG_INT: // x59
		i32, err := d.decInt32(TAG_READ)
		return int64(i32), err

		// direct long
	case tag >= 0xd8 && tag <= 0xef:
		return int64(tag - BC_LONG_ZERO), nil

		// byte long
	case tag >= 0xf0 && tag <= 0xff:
		if _, err = io.ReadFull(d.reader, buf[:1]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int64(tag-BC_LONG_BYTE_ZERO)<<8 + int64(buf[0]), nil

		// short long
	case tag >= 0x38 && tag <= 0x3f: // ['8',  '?']
		if _, err := io.ReadFull(d.reader, buf[:2]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int64(tag-BC_LONG_SHORT_ZERO)<<16 + int64(buf[0])<<8 + int64(buf[1]), nil
		// return int64(tag-BC_LONG_SHORT_ZERO)<<16 + int64(buf[0])*256 + int64(buf[1]), nil

	case tag == BC_LONG: // 'L'
		if _, err := io.ReadFull(d.reader, buf[:8]); err != nil {
			return 0, jerrors.Trace(err)
		}
		return int64(buf[0])<<56 + int64(buf[1])<<48 + int64(buf[2])<<40 + int64(buf[3])<<32 +
			int64(buf[4])<<24 + int64(buf[5])<<16 + int64(buf[6])<<8 + int64(buf[7]), nil

	case tag == BC_DOUBLE_ZERO:
		return int64(0), nil

	case tag == BC_DOUBLE_ONE:
		return int64(1), nil

	case tag == BC_DOUBLE_MILL:
		i64, err := d.decInt32(TAG_READ)
		return int64(i64), jerrors.Trace(err)

	default:
		return 0, jerrors.Errorf("decInt64 long wrong tag:%#x", tag)
	}
}

/////////////////////////////////////////
// Date
/////////////////////////////////////////

// # time in UTC encoded as 64-bit long milliseconds since epoch
// ::= x4a b7 b6 b5 b4 b3 b2 b1 b0
// ::= x4b b3 b2 b1 b0       # minutes since epoch
func (d *Decoder) decDate(flag int32) (time.Time, error) {
	var (
		err error
		l   int
		tag byte
		buf [8]byte
		s   []byte
		i64 int64
		t   time.Time
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_DATE: //'d': //date
		s = buf[:8]
		l, err = d.next(s)
		if err != nil {
			return t, err
		}
		if l != 8 {
			return t, ErrNotEnoughBuf
		}
		i64 = UnpackInt64(s)
		return time.Unix(i64/1000, i64%1000*10e5), nil
		// return time.Unix(i64/1000, i64*100), nil

	case tag == BC_DATE_MINUTE:
		s = buf[:4]
		l, err = d.next(s)
		if err != nil {
			return t, err
		}
		if l != 4 {
			return t, ErrNotEnoughBuf
		}
		i64 = int64(UnpackInt32(s))
		return time.Unix(i64*60, 0), nil

	default:
		return t, jerrors.Errorf("decDate Invalid type: %v", tag)
	}
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
func (d *Decoder) decDouble(flag int32) (interface{}, error) {
	var (
		err error
		tag byte
		buf [8]byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}
	switch tag {
	case BC_LONG_INT:
		return d.decInt32(TAG_READ)

	case BC_DOUBLE_ZERO:
		return float64(0), nil

	case BC_DOUBLE_ONE:
		return float64(1), nil

	case BC_DOUBLE_BYTE:
		tag, _ = d.readByte()
		return float64(tag), nil

	case BC_DOUBLE_SHORT:
		if _, err = io.ReadFull(d.reader, buf[:2]); err != nil {
			return nil, jerrors.Trace(err)
		}

		return float64(int(buf[0])<<8 + int(buf[1])), nil

	case BC_DOUBLE_MILL:
		i, _ := d.decInt32(TAG_READ)
		return float64(i), nil

	case BC_DOUBLE:
		if _, err = io.ReadFull(d.reader, buf[:8]); err != nil {
			return nil, jerrors.Trace(err)
		}

		bits := binary.BigEndian.Uint64(buf[:8])
		datum := math.Float64frombits(bits)
		return datum, nil
	}

	return nil, jerrors.Errorf("decDouble parse double wrong tag:%d-%#x", int(tag), tag)
}

/////////////////////////////////////////
// String
/////////////////////////////////////////

// # UTF-8 encoded character string split into 64k chunks
// ::= x52 b1 b0 <utf8-data> string  # non-final chunk
// ::= 'S' b1 b0 <utf8-data>         # string of length 0-65535
// ::= [x00-x1f] <utf8-data>         # string of length 0-31
// ::= [x30-x34] <utf8-data>         # string of length 0-1023
func (d *Decoder) getStringLength(tag byte) (int32, error) {
	var (
		err    error
		buf    [2]byte
		length int32
	)

	switch {
	case tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX:
		return int32(tag - 0x00), nil

	case tag >= 0x30 && tag <= 0x33:
		_, err = io.ReadFull(d.reader, buf[:1])
		if err != nil {
			return -1, jerrors.Trace(err)
		}

		length = int32(tag-0x30)<<8 + int32(buf[0])
		return length, nil

	case tag == BC_STRING_CHUNK || tag == BC_STRING:
		_, err = io.ReadFull(d.reader, buf[:2])
		if err != nil {
			return -1, jerrors.Trace(err)
		}
		length = int32(buf[0])<<8 + int32(buf[1])
		return length, nil

	default:
		return -1, jerrors.Trace(err)
	}
}

func getRune(reader io.Reader) (rune, int, error) {
	var (
		runeNil rune
		typ     reflect.Type
	)

	typ = reflect.TypeOf(reader.(interface{}))

	switch {
	case typ == reflect.TypeOf(&bufio.Reader{}):
		byteReader := reader.(interface{}).(*bufio.Reader)
		return byteReader.ReadRune()

	case typ == reflect.TypeOf(&bytes.Buffer{}):
		byteReader := reader.(interface{}).(*bytes.Buffer)
		return byteReader.ReadRune()

	case typ == reflect.TypeOf(&bytes.Reader{}):
		byteReader := reader.(interface{}).(*bytes.Reader)
		return byteReader.ReadRune()

	default:
		return runeNil, 0, nil
	}
}

// hessian-lite/src/main/java/com/alibaba/com/caucho/hessian/io/Hessian2Input.java : readString
func (d *Decoder) decString(flag int32) (string, error) {
	var (
		tag    byte
		length int32
		last   bool
		s      string
		r      rune
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == byte(BC_NULL):
		return STRING_NIL, nil

	case tag == byte(BC_TRUE):
		return STRING_TRUE, nil

	case tag == byte(BC_FALSE):
		return STRING_FALSE, nil

	case (0x80 <= tag && tag <= 0xbf) || (0xc0 <= tag && tag <= 0xcf) ||
		(0xd0 <= tag && tag <= 0xd7) || tag == BC_INT ||
		(tag >= 0xd8 && tag <= 0xef) || (tag >= 0xf0 && tag <= 0xff) ||
		(tag >= 0x38 && tag <= 0x3f) || (tag == BC_LONG_INT) || (tag == BC_LONG):
		i64, err := d.decInt64(int32(tag))
		if err != nil {
			return "", jerrors.Annotatef(err, "tag:%+v", tag)
		}

		return strconv.Itoa(int(i64)), nil

	case tag == byte(BC_DOUBLE_ZERO):
		return STRING_ZERO, nil

	case tag == byte(BC_DOUBLE_ONE):
		return STRING_ONE, nil

	case tag == byte(BC_DOUBLE_BYTE) || tag == byte(BC_DOUBLE_SHORT):
		f, err := d.decDouble(int32(tag))
		if err != nil {
			return "", jerrors.Annotatef(err, "tag:%+v", tag)
		}

		return strconv.FormatFloat(f.(float64), 'E', -1, 64), nil
	}

	last = true
	if (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
		(tag >= 0x30 && tag <= 0x33) ||
		(tag == BC_STRING_CHUNK || tag == BC_STRING) {

		if tag == BC_STRING_CHUNK {
			last = false
		} else {
			last = true
		}

		l, err := d.getStringLength(tag)
		if err != nil {
			return s, jerrors.Trace(err)
		}
		length = l
		runeDate := make([]rune, length)
		for i := 0; ; {
			if int32(i) == length {
				if last {
					return string(runeDate), nil
				}

				b, _ := d.readByte()
				switch {
				case (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
					(tag >= 0x30 && tag <= 0x33) ||
					(tag == BC_STRING_CHUNK || tag == BC_STRING):

					if b == BC_STRING_CHUNK {
						last = false
					} else {
						last = true
					}

					l, err := d.getStringLength(b)
					if err != nil {
						return s, jerrors.Trace(err)
					}
					length += l
					bs := make([]rune, length)
					copy(bs, runeDate)
					runeDate = bs

				default:
					return s, jerrors.Trace(err)
				}

			} else {
				r, _, err = d.reader.ReadRune()
				if err != nil {
					return s, jerrors.Trace(err)
				}
				runeDate[i] = r
				i++
			}
		}

		return string(runeDate), nil
	}

	return s, jerrors.Errorf("unknown string tag %#x\n", tag)
}

/////////////////////////////////////////
// Binary, []byte
/////////////////////////////////////////

// # 8-bit binary data split into 64k chunks
// ::= x41('A') b1 b0 <binary-data> binary # non-final chunk
// ::= x42('B') b1 b0 <binary-data>        # final chunk
// ::= [x20-x2f] <binary-data>  # binary data of length 0-15
// ::= [x34-x37] <binary-data>  # binary data of length 0-1023
func (d *Decoder) getBinaryLength(tag byte) (int, error) {
	var (
		err error
		buf [2]byte
	)

	if tag >= BC_BINARY_DIRECT && tag <= INT_DIRECT_MAX { // [0x20, 0x2f]
		return int(tag - BC_BINARY_DIRECT), nil
	}

	if tag >= BC_BINARY_SHORT && tag <= byte(0x37) { // [0x34, 0x37]
		_, err = io.ReadFull(d.reader, buf[:1])
		if err != nil {
			return 0, jerrors.Trace(err)
		}

		return int(tag-BC_BINARY_SHORT)<<8 + int(buf[0]), nil
	}

	if tag != BC_BINARY_CHUNK && tag != BC_BINARY {
		return 0, jerrors.Errorf("illegal binary tag:%d", tag)
	}

	_, err = io.ReadFull(d.reader, buf[:2])
	if err != nil {
		return 0, jerrors.Trace(err)
	}

	return int(buf[0])<<8 + int(buf[1]), nil
}

func (d *Decoder) decBinary(flag int32) ([]byte, error) {
	var (
		err    error
		tag    byte
		length int
		chunk  [CHUNK_SIZE]byte
		data   []byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readBufByte()
	}

	if tag == BC_NULL {
		return []byte(""), nil
	}

	data = make([]byte, 0, CHUNK_SIZE<<1)
	for tag == BC_BINARY_CHUNK {
		length, err = d.getBinaryLength(tag)
		if err != nil || CHUNK_SIZE < length {
			return nil, jerrors.Annotatef(err, "getBinaryLength(tag:%d) = length:%d", tag, length)
		}
		_, err = io.ReadFull(d.reader, chunk[:length])
		if err != nil {
			return nil, jerrors.Annotatef(err, "decBinary->io.ReadFull(len:%d)", length)
		}
		data = append(data, chunk[:length]...)
		tag, _ = d.readBufByte()
	}

	length, err = d.getBinaryLength(tag)
	if err != nil || CHUNK_SIZE < length {
		return nil, jerrors.Annotatef(err, "decBinary->getBinaryLength(tag:%d) = length:%d", tag, length)
	}
	_, err = io.ReadFull(d.reader, chunk[:length])
	if err != nil {
		return nil, jerrors.Annotatef(err, "decBinary->io.ReadFull(len:%d)", length)
	}

	return append(data, chunk[:length]...), nil
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

func (d *Decoder) readBufByte() (byte, error) {
	var (
		err error
		buf [1]byte
	)

	_, err = io.ReadFull(d.reader, buf[:1])
	if err != nil {
		return 0, jerrors.Trace(err)
	}

	return buf[0], nil
}

func listFixedTypedLenTag(tag byte) bool {
	return tag >= _listFixedTypedLenTagMin && tag <= _listFixedTypedLenTagMax
}

// Include 3 formats:
// list ::= x55 type value* 'Z'   # variable-length list
//      ::= 'V' type int value*   # fixed-length list
//      ::= [x70-77] type value*  # fixed-length typed list
func typedListTag(tag byte) bool {
	return tag == BC_LIST_FIXED || tag == BC_LIST_VARIABLE || listFixedTypedLenTag(tag)
}

func listFixedUntypedLenTag(tag byte) bool {
	return tag >= _listFixedUntypedLenTagMin && tag <= _listFixedUntypedLenTagMax
}

// Include 3 formats:
//      ::= x57 value* 'Z'        # variable-length untyped list
//      ::= x58 int value*        # fixed-length untyped list
//      ::= [x78-7f] value*       # fixed-length untyped list
func untypedListTag(tag byte) bool {
	return tag == BC_LIST_FIXED_UNTYPED || tag == BC_LIST_VARIABLE_UNTYPED || listFixedUntypedLenTag(tag)
}

//decList read list
func (d *Decoder) decList(flag int32) (interface{}, error) {
	var (
		err error
		tag byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, err = d.readByte()
		if err != nil {
			return nil, jerrors.Trace(err)
		}
	}

	switch {
	case tag == BC_NULL:
		return nil, nil
	case tag == BC_REF:
		return d.decRef(int32(tag))
	case typedListTag(tag):
		return d.readTypedList(tag)
	case untypedListTag(tag):
		return d.readUntypedList(tag)
	default:
		return nil, jerrors.Errorf("error list tag: 0x%x", tag)
	}
}

// readTypedList read typed list
// Include 3 formats:
// list ::= x55 type value* 'Z'   # variable-length list
//      ::= 'V' type int value*   # fixed-length list
//      ::= [x70-77] type value*  # fixed-length typed list
func (d *Decoder) readTypedList(tag byte) (interface{}, error) {
	listTyp, err := d.decString(TAG_READ)
	if err != nil {
		return nil, jerrors.Errorf("error to read list type[%s]: %v", listTyp, err)
	}

	isVariableArr := tag == BC_LIST_VARIABLE

	length := -1
	if listFixedTypedLenTag(tag) {
		length = int(tag - _listFixedTypedLenTagMin)
	} else if tag == BC_LIST_FIXED {
		ii, err := d.decInt32(TAG_READ)
		if err != nil {
			return nil, jerrors.Trace(err)
		}
		length = int(ii)
	} else if isVariableArr {
		length = 0
	} else {
		return nil, jerrors.Errorf("error typed list tag: 0x%x", tag)
	}

	// return when no element
	if length < 0 {
		return nil, nil
	}

	arr := make([]interface{}, length)
	aryValue := reflect.ValueOf(arr)
	holder := d.appendRefs(aryValue)

	for j := 0; j < length || isVariableArr; j++ {
		it, err := d.Decode()
		if err != nil {
			if err == io.EOF && isVariableArr {
				break
			}
			return nil, jerrors.Trace(err)
		}

		if it == nil {
			break
		}

		v := EnsureRawValue(it)
		if isVariableArr {
			aryValue = reflect.Append(aryValue, v)
			holder.change(aryValue)
		} else {
			SetValue(aryValue.Index(j), v)
		}
	}

	return holder, nil
}

//readUntypedList read untyped list
// Include 3 formats:
//      ::= x57 value* 'Z'        # variable-length untyped list
//      ::= x58 int value*        # fixed-length untyped list
//      ::= [x78-7f] value*       # fixed-length untyped list
func (d *Decoder) readUntypedList(tag byte) (interface{}, error) {
	isVariableArr := tag == BC_LIST_VARIABLE_UNTYPED

	var length int
	if listFixedUntypedLenTag(tag) {
		length = int(tag - _listFixedUntypedLenTagMin)
	} else if tag == BC_LIST_FIXED_UNTYPED {
		ii, err := d.decInt32(TAG_READ)
		if err != nil {
			return nil, jerrors.Trace(err)
		}
		length = int(ii)
	} else if isVariableArr {
		length = 0
	} else {
		return nil, jerrors.Errorf("error untyped list tag: %x", tag)
	}

	ary := make([]interface{}, length)
	aryValue := reflect.ValueOf(ary)
	holder := d.appendRefs(aryValue)

	for j := 0; j < length || isVariableArr; j++ {
		it, err := d.Decode()
		if err != nil {
			if err == io.EOF && isVariableArr {
				continue
			}
			return nil, jerrors.Trace(err)
		}

		if isVariableArr {
			aryValue = reflect.Append(aryValue, EnsureRawValue(it))
			holder.change(aryValue)
		} else {
			ary[j] = it
		}
	}

	return holder, nil
}

/////////////////////////////////////////
// Map
/////////////////////////////////////////

// ::= 'M' type (value value)* 'Z'  # key, value map pairs
// ::= 'H' (value value)* 'Z'       # untyped key, value
func (d *Decoder) decMapByValue(value reflect.Value) error {
	var (
		tag        byte
		err        error
		entryKey   interface{}
		entryValue interface{}
	)

	//tag, _ = d.readBufByte()
	tag, err = d.readByte()
	// check error
	if err != nil {
		return jerrors.Trace(err)
	}

	switch tag {
	case BC_NULL:
		// null map tag check
		return nil
	case BC_REF:
		refObj, err := d.decRef(int32(tag))
		if err != nil {
			return jerrors.Trace(err)
		}
		SetValue(value, EnsurePackValue(refObj))
		return nil
	case BC_MAP:
		d.decString(TAG_READ) // read map type , ignored
	case BC_MAP_UNTYPED:
		//do nothing
	default:
		return jerrors.Errorf("expect map header, but get %x", tag)
	}

	m := reflect.MakeMap(UnpackPtrType(value.Type()))
	// pack with pointer, so that to ref the same map
	m = PackPtr(m)
	d.appendRefs(m)

	//read key and value
	for {
		entryKey, err = d.Decode()
		if err != nil {
			// EOF means the end flag 'Z' of map is already read
			if err == io.EOF {
				break
			} else {
				return jerrors.Trace(err)
			}
		}
		if entryKey == nil {
			break
		}
		entryValue, err = d.Decode()
		// fix: check error
		if err != nil {
			return jerrors.Trace(err)
		}
		m.Elem().SetMapIndex(EnsurePackValue(entryKey), EnsurePackValue(entryValue))
	}

	SetValue(value, m)

	return nil
}

func (d *Decoder) decMap(flag int32) (interface{}, error) {
	var (
		err        error
		tag        byte
		ok         bool
		k          interface{}
		v          interface{}
		t          string
		keyName    string
		methodName string
		key        interface{}
		value      interface{}
		inst       interface{}
		m          map[interface{}]interface{}
		fieldValue reflect.Value
		args       []reflect.Value
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_NULL:
		return nil, nil
	case tag == BC_REF:
		return d.decRef(int32(tag))
	case tag == BC_MAP:
		if t, err = d.decType(); err != nil {
			return nil, err
		}
		if _, ok = checkPOJORegistry(t); ok {
			m = make(map[interface{}]interface{}) // 此处假设了map的定义形式，这是不对的
			d.appendRefs(m)

			// d.decType() // 忽略
			for d.peekByte() != byte('z') {
				k, err = d.Decode()
				if err != nil {
					if err == io.EOF {
						break
					}

					return nil, err
				}
				v, err = d.Decode()
				if err != nil {
					return nil, err
				}
				m[k] = v
			}
			_, err = d.readByte()
			// check error
			if err != nil {
				return nil, jerrors.Trace(err)
			}

			return m, nil
		} else {
			inst = createInstance(t)
			d.appendRefs(inst)

			for d.peekByte() != 'z' {
				if key, err = d.Decode(); err != nil {
					return nil, err
				}
				if value, err = d.Decode(); err != nil {
					return nil, err
				}
				//set value of the struct to Zero
				if fieldValue = reflect.ValueOf(value); fieldValue.IsValid() {
					keyName = key.(string)
					if keyName[0] >= 'a' { //convert to Upper
						methodName = "Set" + string(keyName[0]-32) + keyName[1:]
					} else {
						methodName = "Set" + keyName
					}

					args = args[:0]
					args = append(args, fieldValue)
					reflect.ValueOf(inst).MethodByName(methodName).Call(args)
				}
			}

			return inst, nil
		}

	case tag == BC_MAP_UNTYPED:
		m = make(map[interface{}]interface{})
		d.appendRefs(m)
		for d.peekByte() != byte(BC_END) {
			k, err = d.Decode()
			if err != nil {
				if err == io.EOF {
					break
				}

				return nil, err
			}
			v, err = d.Decode()
			if err != nil {
				return nil, err
			}
			m[k] = v
		}
		_, err = d.readByte()
		// check error
		if err != nil {
			return nil, jerrors.Trace(err)
		}
		return m, nil

	default:
		return nil, jerrors.Errorf("illegal map type tag:%+v", tag)
	}
}

/////////////////////////////////////////
// Object
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
//
//
//
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

func (d *Decoder) decClassDef() (interface{}, error) {
	var (
		err       error
		clsName   string
		fieldNum  int32
		fieldName string
		fieldList []string
	)

	clsName, err = d.decString(TAG_READ)
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	fieldNum, err = d.decInt32(TAG_READ)
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	fieldList = make([]string, fieldNum)
	for i := 0; i < int(fieldNum); i++ {
		fieldName, err = d.decString(TAG_READ)
		if err != nil {
			return nil, jerrors.Annotatef(err, "decClassDef->decString, filed num:%d, index:%d", fieldNum, i)
		}
		fieldList[i] = fieldName
	}

	return classInfo{javaName: clsName, fieldNameList: fieldList}, nil
}

func findField(name string, typ reflect.Type) (int, error) {
	for i := 0; i < typ.NumField(); i++ {
		str := typ.Field(i).Name
		if strings.Compare(str, name) == 0 {
			return i, nil
		}
		// str1 := strings.Title(name)
		str1 := strings.ToLower(str)
		if strings.Compare(name, str1) == 0 {
			return i, nil
		}
	}

	return 0, jerrors.Errorf("failed to find field %s", name)
}

func (d *Decoder) decInstance(typ reflect.Type, cls classInfo) (interface{}, error) {
	if typ.Kind() != reflect.Struct {
		return nil, jerrors.Errorf("wrong type expect Struct but get:%s", typ.String())
	}

	vRef := reflect.New(typ)
	// add pointer ref so that ref the same object
	d.appendRefs(vRef)

	vv := vRef.Elem()
	for i := 0; i < len(cls.fieldNameList); i++ {
		fieldName := cls.fieldNameList[i]

		index, err := findField(fieldName, typ)
		if err != nil {
			return nil, jerrors.Errorf("can not find field %s", fieldName)
		}
		field := vv.Field(index)
		if !field.CanSet() {
			return nil, jerrors.Errorf("decInstance CanSet false for field %s", fieldName)
		}

		// get field type from type object, not do that from value
		fldTyp := UnpackPtrType(field.Type())

		// unpack pointer to enable value setting
		fldRawValue := UnpackPtrValue(field)

		kind := fldTyp.Kind()
		switch {
		case kind == reflect.String:
			str, err := d.decString(TAG_READ)
			if err != nil {
				return nil, jerrors.Annotatef(err, "decInstance->ReadString: %s", fieldName)
			}
			fldRawValue.SetString(str)

		case kind == reflect.Int32 || kind == reflect.Int16:
			num, err := d.decInt32(TAG_READ)
			if err != nil {
				// java enum
				if fldRawValue.Type().Implements(javaEnumType) {
					d.unreadByte() // enum解析，上面decInt64已经读取一个字节，所以这里需要回退一个字节
					s, err := d.Decode()
					if err != nil {
						return nil, jerrors.Annotatef(err, "decInstance->decObject field name:%s", fieldName)
					}
					enumValue, _ := s.(JavaEnum)
					num = int32(enumValue)
				} else {
					return nil, jerrors.Annotatef(err, "decInstance->ParseInt, field name:%s", fieldName)
				}
			}

			fldRawValue.SetInt(int64(num))

		case kind == reflect.Int || kind == reflect.Int64 || kind == reflect.Uint64:
			num, err := d.decInt64(TAG_READ)
			if err != nil {
				if fldTyp.Implements(javaEnumType) {
					d.unreadByte() // enum解析，上面decInt64已经读取一个字节，所以这里需要回退一个字节
					s, err := d.Decode()
					if err != nil {
						return nil, jerrors.Annotatef(err, "decInstance->decObject field name:%s", fieldName)
					}
					enumValue, _ := s.(JavaEnum)
					num = int64(enumValue)
				} else {
					return nil, jerrors.Annotatef(err, "decInstance->decInt64 field name:%s", fieldName)
				}
			}

			fldRawValue.SetInt(num)

		case kind == reflect.Bool:
			b, err := d.Decode()
			if err != nil {
				return nil, jerrors.Annotatef(err, "decInstance->Decode field name:%s", fieldName)
			}
			fldRawValue.SetBool(b.(bool))

		case kind == reflect.Float32 || kind == reflect.Float64:
			num, err := d.decDouble(TAG_READ)
			if err != nil {
				return nil, jerrors.Annotatef(err, "decInstance->decDouble field name:%s", fieldName)
			}
			fldRawValue.SetFloat(num.(float64))

		case kind == reflect.Map:
			// decode map should use the original field value for correct value setting
			err := d.decMapByValue(field)
			if err != nil {
				return nil, jerrors.Annotatef(err, "decInstance->decMapByValue field name: %s", fieldName)
			}

		case kind == reflect.Slice || kind == reflect.Array:
			m, err := d.decList(TAG_READ)
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, jerrors.Trace(err)
			}

			// set slice separately
			err = SetSlice(fldRawValue, m)
			if err != nil {
				return nil, err
			}
		case kind == reflect.Struct:
			var (
				err error
				s   interface{}
			)
			if fldRawValue.Type().String() == "time.Time" {
				s, err = d.decDate(TAG_READ)
				if err != nil {
					return nil, jerrors.Trace(err)
				}
				fldRawValue.Set(reflect.ValueOf(s))
			} else {
				s, err = d.decObject(TAG_READ)
				if err != nil {
					return nil, jerrors.Trace(err)
				}
				if s != nil {
					// set value which accepting pointers
					SetValue(fldRawValue, EnsurePackValue(s))
				}
			}

		default:
			return nil, jerrors.Errorf("unknown struct member type: %v", kind)
		}
	} // end for

	return vRef, nil
}

func (d *Decoder) appendClsDef(cd classInfo) {
	d.classInfoList = append(d.classInfoList, cd)
}

func (d *Decoder) getStructDefByIndex(idx int) (reflect.Type, classInfo, error) {
	var (
		ok      bool
		clsName string
		cls     classInfo
		s       structInfo
	)

	if len(d.classInfoList) <= idx || idx < 0 {
		return nil, cls, jerrors.Errorf("illegal class index @idx %d", idx)
	}
	cls = d.classInfoList[idx]
	s, ok = getStructInfo(cls.javaName)
	if !ok {
		return nil, cls, jerrors.Errorf("can not find go type name %s in registry", clsName)
	}

	return s.typ, cls, nil
}

func (d *Decoder) decEnum(javaName string, flag int32) (JavaEnum, error) {
	var (
		err       error
		enumName  string
		ok        bool
		info      structInfo
		enumValue JavaEnum
	)
	enumName, err = d.decString(TAG_READ) // java enum class member is "name"
	if err != nil {
		return InvalidJavaEnum, jerrors.Annotate(err, "decString for decJavaEnum")
	}
	info, ok = getStructInfo(javaName)
	if !ok {
		return InvalidJavaEnum, jerrors.Errorf("getStructInfo(javaName:%s) = false", javaName)
	}

	enumValue = info.inst.(POJOEnum).EnumValue(enumName)
	d.appendRefs(enumValue)
	return enumValue, nil
}

func (d *Decoder) decObject(flag int32) (interface{}, error) {
	var (
		tag byte
		idx int32
		err error
		typ reflect.Type
		cls classInfo
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_NULL:
		return nil, nil
	case tag == BC_REF:
		return d.decRef(int32(tag))
	case tag == BC_OBJECT_DEF:
		clsDef, err := d.decClassDef()
		if err != nil {
			return nil, jerrors.Annotate(err, "decObject->decClassDef byte double")
		}
		cls, _ = clsDef.(classInfo)
		//add to slice
		d.appendClsDef(cls)

		return d.Decode()

	case tag == BC_OBJECT:
		idx, err = d.decInt32(TAG_READ)
		if err != nil {
			return nil, err
		}

		typ, cls, err = d.getStructDefByIndex(int(idx))
		if err != nil {
			return nil, err
		}
		if typ.Implements(javaEnumType) {
			return d.decEnum(cls.javaName, TAG_READ)
		}

		return d.decInstance(typ, cls)

	case BC_OBJECT_DIRECT <= tag && tag <= (BC_OBJECT_DIRECT+OBJECT_DIRECT_MAX):
		typ, cls, err = d.getStructDefByIndex(int(tag - BC_OBJECT_DIRECT))
		if err != nil {
			return nil, err
		}
		if typ.Implements(javaEnumType) {
			return d.decEnum(cls.javaName, TAG_READ)
		}

		return d.decInstance(typ, cls)

	default:
		return nil, jerrors.Errorf("decObject illegal object type tag:%+v", tag)
	}
}

/////////////////////////////////////////
// Ref
/////////////////////////////////////////

// # value reference (e.g. circular trees and graphs)
// ref        ::= x51 int            # reference to nth map/list/object
func (d *Decoder) decRef(flag int32) (interface{}, error) {
	var (
		err error
		tag byte
		i   int32
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_REF:
		i, err = d.decInt32(TAG_READ)
		if err != nil {
			return nil, err
		}

		if len(d.refs) <= int(i) {
			return nil, ErrIllegalRefIndex
		}
		// return the exact ref object, which maybe a _refHolder
		return d.refs[i], nil

	default:
		return nil, jerrors.Errorf("decRef illegal ref type tag:%+v", tag)
	}
}

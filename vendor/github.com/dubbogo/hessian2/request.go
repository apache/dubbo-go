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
	"encoding/binary"
	"reflect"
	"strconv"
	"strings"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

/////////////////////////////////////////
// dubbo
/////////////////////////////////////////

/**
 * 协议头是16字节的定长数据
 * 2字节magic字符串0xdabb,0-7高位，8-15低位
 * 1字节的消息标志位。16-20序列id,21 event,22 two way,23请求或响应标识
 * 1字节状态。当消息类型为响应时，设置响应状态。24-31位。
 * 8字节，消息ID,long类型，32-95位。
 * 4字节，消息长度，96-127位
 **/
const (
	// header length.
	HEADER_LENGTH = 16

	// magic header
	MAGIC      = uint16(0xdabb)
	MAGIC_HIGH = byte(0xda)
	MAGIC_LOW  = byte(0xbb)

	// message flag.
	FLAG_REQUEST = byte(0x80)
	FLAG_TWOWAY  = byte(0x40)
	FLAG_EVENT   = byte(0x20) // for heartbeat
	SERIAL_MASK  = 0x1f

	DUBBO_VERSION = "2.5.4"
	DEFAULT_LEN   = 8388608 // 8 * 1024 * 1024 default body max length
)

var (
	DubboHeaderBytes     = [HEADER_LENGTH]byte{MAGIC_HIGH, MAGIC_LOW, FLAG_REQUEST | FLAG_TWOWAY}
	DubboHeartbeatHeader = [HEADER_LENGTH]byte{MAGIC_HIGH, MAGIC_LOW, FLAG_REQUEST | FLAG_TWOWAY | FLAG_EVENT | 0x0F}
)

// com.alibaba.dubbo.common.utils.ReflectUtils.ReflectUtils.java line245 getDesc
func getArgType(v interface{}) string {
	if v == nil {
		return "V"
	}

	switch v.(type) {
	// 基本类型的序列化tag
	case nil:
		return "V"
	case bool:
		return "Z"
	case byte:
		return "B"
	case int8:
		return "B"
	case int16:
		return "S"
	case uint16: // 相当于Java的Char
		return "C"
	// case rune:
	//	return "C"
	case int:
		return "I"
	case int32:
		return "I"
	case int64:
		return "J"
	case time.Time:
		return "java.util.Date"
	case float32:
		return "F"
	case float64:
		return "D"
	case string:
		return "java.lang.String"
	case []byte:
		return "[B"
	case map[interface{}]interface{}:
		// return  "java.util.HashMap"
		return "java.util.Map"

	//  复杂类型的序列化tag
	default:
		t := reflect.TypeOf(v)
		if reflect.Ptr == t.Kind() {
			t = reflect.TypeOf(reflect.ValueOf(v).Elem())
		}
		switch t.Kind() {
		case reflect.Struct:
			return "java.lang.Object"
		case reflect.Slice, reflect.Array:
			// return "java.util.ArrayList"
			return "java.util.List"
		case reflect.Map: // 进入这个case，就说明map可能是map[string]int这种类型
			return "java.util.Map"
		default:
			return ""
		}
	}

	return "java.lang.RuntimeException"
}

func getArgsTypeList(args []interface{}) (string, error) {
	var (
		typ   string
		types string
	)

	for i := range args {
		typ = getArgType(args[i])
		if typ == "" {
			return types, jerrors.Errorf("cat not get arg %#v type", args[i])
		}
		if !strings.Contains(typ, ".") {
			types += typ
		} else {
			// java.util.List -> Ljava/util/List;
			types += "L" + strings.Replace(typ, ".", "/", -1) + ";"
		}
	}

	return types, nil
}

// dubbo-remoting/dubbo-remoting-api/src/main/java/com/alibaba/dubbo/remoting/exchange/codec/ExchangeCodec.java
// v2.5.4 line 204 encodeRequest
func PackRequest(service Service, header DubboHeader, params interface{}) ([]byte, error) {
	var (
		err           error
		types         string
		byteArray     []byte
		encoder       Encoder
		version       string
		pkgLen        int
		serviceParams map[string]string
	)

	args, ok := params.([]interface{})
	if !ok {
		return nil, jerrors.Errorf("@params is not of type: []interface{}")
	}

	hb := header.Type == Heartbeat

	//////////////////////////////////////////
	// byteArray
	//////////////////////////////////////////
	// magic
	if hb {
		byteArray = append(byteArray, DubboHeartbeatHeader[:]...)
	} else {
		byteArray = append(byteArray, DubboHeaderBytes[:]...)
	}
	// serialization id, two way flag, event, request/response flag
	// java 中标识一个class的ID
	byteArray[2] |= byte(header.ID & SERIAL_MASK)
	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))
	encoder.Append(byteArray[:HEADER_LENGTH])

	// com.alibaba.dubbo.rpc.protocol.dubbo.DubboCodec.DubboCodec.java line144 encodeRequestData
	//////////////////////////////////////////
	// body
	//////////////////////////////////////////
	if hb {
		encoder.Encode(nil)
		goto END
	}

	// dubbo version + path + version + method
	encoder.Encode(DUBBO_VERSION)
	encoder.Encode(service.Target)
	encoder.Encode(service.Version)
	encoder.Encode(service.Method)

	// args = args type list + args value list
	if types, err = getArgsTypeList(args); err != nil {
		return nil, jerrors.Annotatef(err, " PackRequest(args:%+v)", args)
	}
	encoder.Encode(types)
	for _, v := range args {
		encoder.Encode(v)
	}

	serviceParams = make(map[string]string)
	serviceParams[PATH_KEY] = service.Path
	serviceParams[INTERFACE_KEY] = service.Interface
	if len(version) != 0 {
		serviceParams[VERSION_KEY] = version
	}
	if service.Timeout != 0 {
		serviceParams[TIMEOUT_KEY] = strconv.Itoa(int(service.Timeout / time.Millisecond))
	}

	encoder.Encode(serviceParams)

END:
	byteArray = encoder.Buffer()
	pkgLen = len(byteArray)
	if pkgLen > int(DEFAULT_LEN) { // 8M
		return nil, jerrors.Errorf("Data length %d too large, max payload %d", pkgLen, DEFAULT_LEN)
	}
	// byteArray{body length}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen-HEADER_LENGTH))

	return byteArray, nil
}

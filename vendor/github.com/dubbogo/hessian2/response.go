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
)

import (
	jerrors "github.com/juju/errors"
)

// hessian decode respone
func UnpackResponseHeaer(buf []byte, header *DubboHeader) error {
	// length := len(buf)
	// // hessianCodec.ReadHeader has check the header length
	// if length < HEADER_LENGTH {
	// 	return ErrHeaderNotEnough
	// }

	if buf[0] != byte(MAGIC_HIGH) && buf[1] != byte(MAGIC_LOW) {
		return ErrIllegalPackage
	}

	// Header{serialization id(5 bit), event, two way, req/response}
	if header.SerialID = buf[2] & SERIAL_MASK; header.SerialID == byte(0x00) {
		return jerrors.Errorf("serialization ID:%v", header.SerialID)
	}

	flag := buf[2] & FLAG_EVENT
	if flag != byte(0x00) {
		header.Type |= Heartbeat
	}
	flag = buf[2] & FLAG_TWOWAY
	if flag != byte(0x00) {
		header.Type |= Response
	}
	flag = buf[2] & FLAG_REQUEST
	if flag != byte(0x00) {
		return jerrors.Errorf("response flag:%v", flag)
	}

	// Header{status}
	var err error
	if buf[3] != Response_OK {
		err = ErrJavaException
		// return jerrors.Errorf("Response not OK, java exception:%s", string(buf[18:length-1]))
	}

	// Header{req id}
	header.ID = int64(binary.BigEndian.Uint64(buf[4:]))

	// Header{body len}
	header.BodyLen = int(binary.BigEndian.Uint32(buf[12:]))
	if header.BodyLen < 0 {
		return ErrIllegalPackage
	}

	return err
}

// hessian decode response body
func UnpackResponseBody(buf []byte, body interface{}) error {
	// body
	decoder := NewDecoder(buf[:])
	rspType, err := decoder.Decode()
	if err != nil {
		return jerrors.Trace(err)
	}

	switch rspType {
	case RESPONSE_WITH_EXCEPTION:
		expt, err := decoder.Decode()
		if err != nil {
			return jerrors.Trace(err)
		}
		return jerrors.Errorf("got exception: %+v", expt)

	case RESPONSE_VALUE:
		rsp, err := decoder.Decode()
		if err != nil {
			return jerrors.Trace(err)
		}
		return jerrors.Trace(ReflectResponse(rsp, body))

	case RESPONSE_NULL_VALUE:
		return jerrors.New("Received null")
	}

	return nil
}

func cpSlice(in, out interface{}) error {
	inSlice := reflect.ValueOf(in)
	if inSlice.IsNil() {
		return jerrors.New("@in is nil")
	}

	outSlice := reflect.ValueOf(out)
	for outSlice.Kind() == reflect.Ptr {
		outSlice = outSlice.Elem()
	}

	outSlice.Set(reflect.MakeSlice(outSlice.Type(), inSlice.Len(), inSlice.Len()))
	//outElemKind := outSlice.Type().Elem().Kind()
	for i := 0; i < outSlice.Len(); i++ {
		inSliceValue := inSlice.Index(i)
		if outSlice.Index(i).Kind() == reflect.Struct {
			//if inSliceValue.Kind() == reflect.Ptr && inSliceValue.Kind() != outElemKind {
			//	inSliceValue = inSliceValue.Elem()
			//}
			inSliceValue = inSliceValue.Interface().(reflect.Value)
		} else {
			inSliceValue = reflect.ValueOf(inSliceValue)
		}
		if !inSliceValue.Type().AssignableTo(outSlice.Index(i).Type()) {
			return jerrors.Errorf("in element type %s can not assign to out element type %s",
				inSliceValue.Type().Name(), outSlice.Type().Name())
		}
		outSlice.Index(i).Set(inSliceValue)
	}

	return nil
}

func cpMap(in, out interface{}) error {
	inMapValue := reflect.ValueOf(in)
	if inMapValue.IsNil() {
		return jerrors.New("@in is nil")
	}
	if !inMapValue.CanInterface() {
		return jerrors.New("@in's Interface can not be used.")
	}
	inMap := inMapValue.Interface().(map[interface{}]interface{})

	outMap := reflect.ValueOf(out)
	for outMap.Kind() == reflect.Ptr {
		outMap = outMap.Elem()
	}

	outMap.Set(reflect.MakeMap(outMap.Type()))
	outKeyType := outMap.Type().Key()
	outKeyKind := outKeyType.Kind()
	outValueType := outMap.Type().Elem()
	outValueKind := outValueType.Kind()
	var inKey, inValue reflect.Value
	for k := range inMap {
		if outKeyKind != reflect.Struct {
			inKey = reflect.ValueOf(k)
		} else {
			inKey = k.(reflect.Value)
		}
		if outValueKind != reflect.Struct {
			inValue = reflect.ValueOf(inMap[k])
		} else {
			inValue = inMap[k].(reflect.Value)
		}
		if !inKey.Type().AssignableTo(outKeyType) {
			return jerrors.Errorf("in Key:{type:%s, value:%#v} can not assign to out Key:{type:%s} ",
				inKey.Type().Name(), inKey, outKeyType.Name())
		}
		if !inValue.Type().AssignableTo(outValueType) {
			return jerrors.Errorf("in Value:{type:%s, value:%#v} can not assign to out value:{type:%s}",
				inValue.Type().Name(), inValue, outValueType.Name())
		}
		outMap.SetMapIndex(inKey, inValue)
	}

	return nil
}

// reflect return value
func ReflectResponse(in interface{}, out interface{}) error {
	if in == nil {
		return jerrors.Errorf("@in is nil")
	}

	if out == nil {
		return jerrors.Errorf("@out is nil")
	}
	if reflect.TypeOf(out).Kind() != reflect.Ptr {
		return jerrors.Errorf("@out should be a pointer")
	}

	if inV, ok := in.(reflect.Value); ok {
		in = inV.Interface()
	}

	inType := reflect.TypeOf(in)
	switch inType.Kind() {
	case reflect.Bool:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(bool)))
	case reflect.Int8:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(int8)))
	case reflect.Int16:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(int16)))
	case reflect.Int32:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(int32)))
	case reflect.Int64:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(int64)))
	case reflect.Float32:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(float32)))
	case reflect.Float64:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(float64)))
	case reflect.String:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(string)))
	case reflect.Ptr:
		reflect.ValueOf(out).Elem().Set(reflect.ValueOf(in.(reflect.Value).Elem().Interface()))
	case reflect.Struct:
		reflect.ValueOf(out).Elem().Set(in.(reflect.Value)) // reflect.ValueOf(in.(reflect.Value)))
	case reflect.Slice, reflect.Array:
		return cpSlice(in, out)
	case reflect.Map:
		return cpMap(in, out)
	}

	return nil
}

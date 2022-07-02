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

package hessian2

import (
	"encoding/binary"
	"math"
	"reflect"
	"strconv"
	"strings"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go-hessian2/java_exception"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

// DubboResponse dubbo response
type DubboResponse struct {
	RspObj      interface{}
	Exception   error
	Attachments map[string]interface{}
}

// NewResponse create a new DubboResponse
func NewResponse(rspObj interface{}, exception error, attachments map[string]interface{}) *DubboResponse {
	if attachments == nil {
		attachments = make(map[string]interface{}, 8)
	}
	return &DubboResponse{
		RspObj:      rspObj,
		Exception:   exception,
		Attachments: attachments,
	}
}

// EnsureResponse check body type, make sure it's a DubboResponse or package it as a DubboResponse
func EnsureResponse(body interface{}) *DubboResponse {
	if res, ok := body.(*DubboResponse); ok {
		return res
	}
	if exp, ok := body.(error); ok {
		return NewResponse(nil, exp, nil)
	}
	return NewResponse(body, nil, nil)
}

// https://github.com/apache/dubbo/blob/dubbo-2.7.1/dubbo-remoting/dubbo-remoting-api/src/main/java/org/apache/dubbo/remoting/exchange/codec/ExchangeCodec.java#L256
// hessian encode response
func packResponse(header DubboHeader, ret interface{}) ([]byte, error) {
	var byteArray []byte

	response := EnsureResponse(ret)

	hb := header.Type == PackageHeartbeat

	// magic
	if hb {
		byteArray = append(byteArray, DubboResponseHeartbeatHeader[:]...)
	} else {
		byteArray = append(byteArray, DubboResponseHeaderBytes[:]...)
	}
	// set serialID, identify serialization types, eg: fastjson->6, hessian2->2
	byteArray[2] |= header.SerialID & SERIAL_MASK
	// response status
	if header.ResponseStatus != 0 {
		byteArray[3] = header.ResponseStatus
	}

	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))

	// body
	encoder := hessian.NewEncoder()
	encoder.Append(byteArray[:HEADER_LENGTH])

	if header.ResponseStatus == Response_OK {
		if hb {
			if err := encoder.Encode(nil); err != nil {
				logger.Warnf("Encode(nil) = %v", err)
			}
		} else {
			atta := isSupportResponseAttachment(response.Attachments[DUBBO_VERSION_KEY])

			var resWithException, resValue, resNullValue int32
			if atta {
				resWithException = RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS
				resValue = RESPONSE_VALUE_WITH_ATTACHMENTS
				resNullValue = RESPONSE_NULL_VALUE_WITH_ATTACHMENTS
			} else {
				resWithException = RESPONSE_WITH_EXCEPTION
				resValue = RESPONSE_VALUE
				resNullValue = RESPONSE_NULL_VALUE
			}

			if response.Exception != nil { // throw error
				err := encoder.Encode(resWithException)
				if err != nil {
					return nil, perrors.Errorf("encoding response failed: %v", err)
				}
				if t, ok := response.Exception.(java_exception.Throwabler); ok {
					err = encoder.Encode(t)
				} else {
					err = encoder.Encode(java_exception.NewThrowable(response.Exception.Error()))
				}
				if err != nil {
					return nil, perrors.Errorf("encoding exception failed: %v", err)
				}
			} else {
				if response.RspObj == nil {
					if err := encoder.Encode(resNullValue); err != nil {
						return nil, perrors.Errorf("encoding null value failed: %v", err)
					}
				} else {
					if err := encoder.Encode(resValue); err != nil {
						return nil, perrors.Errorf("encoding response value failed: %v", err)
					}
					if err := encoder.Encode(response.RspObj); err != nil {
						return nil, perrors.Errorf("encoding response failed: %v", err)
					}
				}
			}

			// attachments
			if atta {
				if err := encoder.Encode(response.Attachments); err != nil {
					return nil, perrors.Errorf("encoding response attachements failed: %v", err)
				}
			}
		}
	} else {
		var err error
		if response.Exception != nil { // throw error
			err = encoder.Encode(response.Exception.Error())
		} else {
			err = encoder.Encode(response.RspObj)
		}
		if err != nil {
			return nil, perrors.Errorf("encoding error failed: %v", err)
		}
	}

	byteArray = encoder.Buffer()
	byteArray = hessian.EncNull(byteArray) // if not, "java client" will throw exception  "unexpected end of file"
	pkgLen := len(byteArray)
	if pkgLen > int(DEFAULT_LEN) { // recommand 8M
		logger.Warnf("Data length %d too large, recommand max payload %d. "+
			"Dubbo java can't handle the package whose size greater than %d!!!", pkgLen, DEFAULT_LEN, DEFAULT_LEN)
	}
	// byteArray{body length}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen-HEADER_LENGTH))
	return byteArray, nil
}

// hessian decode response body
func unpackResponseBody(decoder *hessian.Decoder, resp interface{}) error {
	// body
	if decoder == nil {
		return perrors.Errorf("@decoder is nil")
	}
	rspType, err := decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}

	response := EnsureResponse(resp)

	switch rspType {
	case RESPONSE_WITH_EXCEPTION, RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS:
		expt, err := decoder.Decode()
		if err != nil {
			return perrors.WithStack(err)
		}
		if rspType == RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS {
			attachments, err := decoder.Decode()
			if err != nil {
				return perrors.WithStack(err)
			}
			if v, ok := attachments.(map[interface{}]interface{}); ok {
				atta := ToMapStringInterface(v)
				response.Attachments = atta
			} else {
				return perrors.Errorf("get wrong attachments: %+v", attachments)
			}
		}

		if e, ok := expt.(error); ok {
			response.Exception = e
		} else {
			response.Exception = perrors.Errorf("got exception: %+v", expt)
		}
		return nil

	case RESPONSE_VALUE, RESPONSE_VALUE_WITH_ATTACHMENTS:
		rsp, err := decoder.Decode()
		if err != nil {
			return perrors.WithStack(err)
		}
		if rspType == RESPONSE_VALUE_WITH_ATTACHMENTS {
			attachments, err := decoder.Decode()
			if err != nil {
				return perrors.WithStack(err)
			}
			if v, ok := attachments.(map[interface{}]interface{}); ok {
				response.Attachments = ToMapStringInterface(v)
			} else {
				return perrors.Errorf("get wrong attachments: %+v", attachments)
			}
		}

		response.RspObj = rsp

		return nil

	case RESPONSE_NULL_VALUE, RESPONSE_NULL_VALUE_WITH_ATTACHMENTS:
		if rspType == RESPONSE_NULL_VALUE_WITH_ATTACHMENTS {
			attachments, err := decoder.Decode()
			if err != nil {
				return perrors.WithStack(err)
			}
			if v, ok := attachments.(map[interface{}]interface{}); ok {
				atta := ToMapStringInterface(v)
				response.Attachments = atta
			} else {
				return perrors.Errorf("get wrong attachments: %+v", attachments)
			}
		}
		return nil
	}

	return nil
}

// CopySlice copy from inSlice to outSlice
func CopySlice(inSlice, outSlice reflect.Value) error {
	if inSlice.IsNil() {
		return perrors.New("@in is nil")
	}
	if inSlice.Kind() != reflect.Slice {
		return perrors.Errorf("@in is not slice, but %v", inSlice.Kind())
	}

	for outSlice.Kind() == reflect.Ptr {
		outSlice = outSlice.Elem()
	}

	size := inSlice.Len()
	outSlice.Set(reflect.MakeSlice(outSlice.Type(), size, size))

	for i := 0; i < size; i++ {
		inSliceValue := inSlice.Index(i)
		if !inSliceValue.Type().AssignableTo(outSlice.Index(i).Type()) {
			return perrors.Errorf("in element type [%s] can not assign to out element type [%s]",
				inSliceValue.Type().String(), outSlice.Type().String())
		}
		outSlice.Index(i).Set(inSliceValue)
	}

	return nil
}

// CopyMap copy from in map to out map
func CopyMap(inMapValue, outMapValue reflect.Value) error {
	if inMapValue.IsNil() {
		return perrors.New("@in is nil")
	}
	if !inMapValue.CanInterface() {
		return perrors.New("@in's Interface can not be used.")
	}
	if inMapValue.Kind() != reflect.Map {
		return perrors.Errorf("@in is not map, but %v", inMapValue.Kind())
	}

	outMapType := hessian.UnpackPtrType(outMapValue.Type())
	hessian.SetValue(outMapValue, reflect.MakeMap(outMapType))

	outKeyType := outMapType.Key()

	outMapValue = hessian.UnpackPtrValue(outMapValue)
	outValueType := outMapValue.Type().Elem()

	for _, inKey := range inMapValue.MapKeys() {
		inValue := inMapValue.MapIndex(inKey)

		if !inKey.Type().AssignableTo(outKeyType) {
			return perrors.Errorf("in Key:{type:%s, value:%#v} can not assign to out Key:{type:%s} ",
				inKey.Type().String(), inKey, outKeyType.String())
		}
		if !inValue.Type().AssignableTo(outValueType) {
			return perrors.Errorf("in Value:{type:%s, value:%#v} can not assign to out value:{type:%s}",
				inValue.Type().String(), inValue, outValueType.String())
		}
		outMapValue.SetMapIndex(inKey, inValue)
	}

	return nil
}

// ReflectResponse reflect return value
// TODO response object should not be copied again to another object, it should be the exact type of the object
func ReflectResponse(in interface{}, out interface{}) error {
	if in == nil {
		return perrors.Errorf("@in is nil")
	}

	if out == nil {
		return perrors.Errorf("@out is nil")
	}
	if reflect.TypeOf(out).Kind() != reflect.Ptr {
		return perrors.Errorf("@out should be a pointer")
	}

	inValue := hessian.EnsurePackValue(in)
	outValue := hessian.EnsurePackValue(out)

	outType := outValue.Type().String()
	if outType == "interface {}" || outType == "*interface {}" {
		hessian.SetValue(outValue, inValue)
		return nil
	}

	switch inValue.Type().Kind() {
	case reflect.Slice, reflect.Array:
		return CopySlice(inValue, outValue)
	case reflect.Map:
		return CopyMap(inValue, outValue)
	default:
		hessian.SetValue(outValue, inValue)
	}

	return nil
}

var versionInt = make(map[string]int)

// https://github.com/apache/dubbo/blob/dubbo-2.7.1/dubbo-common/src/main/java/org/apache/dubbo/common/Version.java#L96
// isSupportResponseAttachment is for compatibility among some dubbo version
func isSupportResponseAttachment(ver interface{}) bool {
	version, ok := ver.(string)
	if !ok || len(version) == 0 {
		return false
	}

	v, ok := versionInt[version]
	if !ok {
		v = version2Int(version)
		if v == -1 {
			return false
		}
	}

	if v >= 2001000 && v <= 2060200 { // 2.0.10 ~ 2.6.2
		return false
	}
	return v >= LOWEST_VERSION_FOR_RESPONSE_ATTACHMENT
}

func version2Int(ver interface{}) int {
	version, ok := ver.(string)
	if !ok || len(version) == 0 {
		return 0
	}
	v := 0
	varr := strings.Split(version, ".")
	length := len(varr)
	for key, value := range varr {
		v0, err := strconv.Atoi(value)
		if err != nil {
			return -1
		}
		v += v0 * int(math.Pow10((length-key-1)*2))
	}
	if length == 3 {
		return v * 100
	}
	return v
}

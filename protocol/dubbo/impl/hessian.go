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

package impl

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go-hessian2/java_exception"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type HessianSerializer struct{}

func (h HessianSerializer) Marshal(p DubboPackage) ([]byte, error) {
	encoder := hessian.NewEncoder()
	if p.IsRequest() {
		return marshalRequest(encoder, p)
	}
	return marshalResponse(encoder, p)
}

func (h HessianSerializer) Unmarshal(input []byte, p *DubboPackage) error {
	if p.IsHeartBeat() {
		return nil
	}
	if p.IsRequest() {
		return unmarshalRequestBody(input, p)
	}
	return unmarshalResponseBody(input, p)
}

func marshalResponse(encoder *hessian.Encoder, p DubboPackage) ([]byte, error) {
	header := p.Header
	response := EnsureResponsePayload(p.Body)
	if header.ResponseStatus == Response_OK {
		if p.IsHeartBeat() {
			_ = encoder.Encode(nil)
		} else {
			var version string
			if attachmentVersion, ok := response.Attachments[DUBBO_VERSION_KEY]; ok {
				version = attachmentVersion.(string)
			}
			atta := isSupportResponseAttachment(version)

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
				_ = encoder.Encode(resWithException)
				if t, ok := response.Exception.(java_exception.Throwabler); ok {
					_ = encoder.Encode(t)
				} else {
					_ = encoder.Encode(java_exception.NewThrowable(response.Exception.Error()))
				}
			} else {
				if response.RspObj == nil {
					_ = encoder.Encode(resNullValue)
				} else {
					_ = encoder.Encode(resValue)
					_ = encoder.Encode(response.RspObj) // result
				}
			}

			if atta {
				_ = encoder.Encode(response.Attachments) // attachments
			}
		}
	} else {
		if response.Exception != nil { // throw error
			_ = encoder.Encode(response.Exception.Error())
		} else {
			_ = encoder.Encode(response.RspObj)
		}
	}
	bs := encoder.Buffer()
	// encNull
	bs = append(bs, byte('N'))
	return bs, nil
}

func marshalRequest(encoder *hessian.Encoder, p DubboPackage) ([]byte, error) {
	service := p.Service
	request := EnsureRequestPayload(p.Body)
	_ = encoder.Encode(DEFAULT_DUBBO_PROTOCOL_VERSION)
	_ = encoder.Encode(service.Path)
	_ = encoder.Encode(service.Version)
	_ = encoder.Encode(service.Method)

	args, ok := request.Params.([]interface{})

	if !ok {
		logger.Infof("request args are: %+v", request.Params)
		return nil, perrors.Errorf("@params is not of type: []interface{}")
	}
	types, err := GetArgsTypeList(args)
	if err != nil {
		return nil, perrors.Wrapf(err, " PackRequest(args:%+v)", args)
	}
	_ = encoder.Encode(types)
	for _, v := range args {
		_ = encoder.Encode(v)
	}

	request.Attachments[PATH_KEY] = service.Path
	request.Attachments[VERSION_KEY] = service.Version
	if len(service.Group) > 0 {
		request.Attachments[GROUP_KEY] = service.Group
	}
	if len(service.Interface) > 0 {
		request.Attachments[INTERFACE_KEY] = service.Interface
	}
	if service.Timeout != 0 {
		request.Attachments[TIMEOUT_KEY] = strconv.Itoa(int(service.Timeout / time.Millisecond))
	}

	err = encoder.Encode(request.Attachments)
	return encoder.Buffer(), err
}

var versionInt = make(map[string]int)

// https://github.com/apache/dubbo/blob/dubbo-2.7.1/dubbo-common/src/main/java/org/apache/dubbo/common/Version.java#L96
// isSupportResponseAttachment is for compatibility among some dubbo version
func isSupportResponseAttachment(version string) bool {
	if version == "" {
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

func version2Int(version string) int {
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

func unmarshalRequestBody(body []byte, p *DubboPackage) error {
	if p.Body == nil {
		p.SetBody(make([]interface{}, 7))
	}
	decoder := hessian.NewDecoder(body)
	var (
		err                                                     error
		dubboVersion, target, serviceVersion, method, argsTypes interface{}
		args                                                    []interface{}
	)
	req, ok := p.Body.([]interface{})
	if !ok {
		return perrors.Errorf("@reqObj is not of type: []interface{}")
	}
	dubboVersion, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[0] = dubboVersion

	target, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[1] = target

	serviceVersion, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[2] = serviceVersion

	method, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[3] = method

	argsTypes, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[4] = argsTypes

	ats := hessian.DescRegex.FindAllString(argsTypes.(string), -1)
	var arg interface{}
	for i := 0; i < len(ats); i++ {
		arg, err = decoder.Decode()
		if err != nil {
			return perrors.WithStack(err)
		}
		args = append(args, arg)
	}
	req[5] = args

	attachments, err := decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}

	if attachments == nil || attachments == "" {
		attachments = map[interface{}]interface{}{constant.InterfaceKey: target}
	}

	if v, ok := attachments.(map[interface{}]interface{}); ok {
		v[DUBBO_VERSION_KEY] = dubboVersion
		req[6] = ToMapStringInterface(v)
		buildServerSidePackageBody(p)
		return nil
	}
	return perrors.Errorf("get wrong attachments: %+v", attachments)
}

func unmarshalResponseBody(body []byte, p *DubboPackage) error {
	decoder := hessian.NewDecoder(body)
	rspType, err := decoder.Decode()
	if p.Body == nil {
		p.SetBody(&ResponsePayload{})
	}
	if err != nil {
		return perrors.WithStack(err)
	}
	response := EnsureResponsePayload(p.Body)

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
				atta := ToMapStringInterface(v)
				response.Attachments = atta
			} else {
				return perrors.Errorf("get wrong attachments: %+v", attachments)
			}
		}

		return perrors.WithStack(hessian.ReflectResponse(rsp, response.RspObj))

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

func buildServerSidePackageBody(pkg *DubboPackage) {
	req := pkg.GetBody().([]interface{}) // length of body should be 7
	if len(req) > 0 {
		var dubboVersion, argsTypes string
		var args []interface{}
		var attachments map[string]interface{}
		svc := Service{}
		if req[0] != nil {
			dubboVersion = req[0].(string)
		}
		if req[1] != nil {
			svc.Path = req[1].(string)
		}
		if req[2] != nil {
			svc.Version = req[2].(string)
		}
		if req[3] != nil {
			svc.Method = req[3].(string)
		}
		if req[4] != nil {
			argsTypes = req[4].(string)
		}
		if req[5] != nil {
			args = req[5].([]interface{})
		}
		if req[6] != nil {
			attachments = req[6].(map[string]interface{})
		}
		if svc.Path == "" && attachments[constant.PathKey] != nil && len(attachments[constant.PathKey].(string)) > 0 {
			svc.Path = attachments[constant.PathKey].(string)
		}
		if _, ok := attachments[constant.InterfaceKey]; ok {
			svc.Interface = attachments[constant.InterfaceKey].(string)
		} else {
			svc.Interface = svc.Path
		}
		if _, ok := attachments[constant.GroupKey]; ok {
			svc.Group = attachments[constant.GroupKey].(string)
		}
		pkg.SetService(svc)
		pkg.SetBody(map[string]interface{}{
			"dubboVersion": dubboVersion,
			"argsTypes":    argsTypes,
			"args":         args,
			"service":      common.ServiceMap.GetService(DUBBO, svc.Interface, svc.Group, svc.Version), // path as a key
			"attachments":  attachments,
		})
	}
}

func GetArgsTypeList(args []interface{}) (string, error) {
	var (
		typ   string
		types string
	)

	for i := range args {
		typ = getArgType(args[i])
		if typ == "" {
			return types, perrors.Errorf("cat not get arg %#v type", args[i])
		}
		if !strings.Contains(typ, ".") {
			types += typ
		} else if strings.Index(typ, "[") == 0 {
			types += strings.Replace(typ, ".", "/", -1)
		} else {
			// java.util.List -> Ljava/util/List;
			types += "L" + strings.Replace(typ, ".", "/", -1) + ";"
		}
	}

	return types, nil
}

func getArgType(v interface{}) string {
	if v == nil {
		return "V"
	}

	switch v := v.(type) {
	// Serialized tags for base types
	case nil:
		return "V"
	case bool:
		return "Z"
	case []bool:
		return "[Z"
	case byte:
		return "B"
	case []byte:
		return "[B"
	case int8:
		return "B"
	case []int8:
		return "[B"
	case int16:
		return "S"
	case []int16:
		return "[S"
	case uint16: // Equivalent to Char of Java
		return "C"
	case []uint16:
		return "[C"
	// case rune:
	//	return "C"
	case int:
		return "J"
	case []int:
		return "[J"
	case int32:
		return "I"
	case []int32:
		return "[I"
	case int64:
		return "J"
	case []int64:
		return "[J"
	case time.Time:
		return "java.util.Date"
	case []time.Time:
		return "[Ljava.util.Date"
	case float32:
		return "F"
	case []float32:
		return "[F"
	case float64:
		return "D"
	case []float64:
		return "[D"
	case string:
		return "java.lang.String"
	case []string:
		return "[Ljava.lang.String;"
	case []hessian.Object:
		return "[Ljava.lang.Object;"
	case map[interface{}]interface{}:
		// return  "java.util.HashMap"
		return "java.util.Map"
	case hessian.POJOEnum:
		return v.(hessian.POJOEnum).JavaClassName()
	case *int8:
		return "java.lang.Byte"
	case *int16:
		return "java.lang.Short"
	case *uint16:
		return "java.lang.Character"
	case *int:
		return "java.lang.Long"
	case *int32:
		return "java.lang.Integer"
	case *int64:
		return "java.lang.Long"
	case *float32:
		return "java.lang.Float"
	case *float64:
		return "java.lang.Double"
	//  Serialized tags for complex types
	default:
		t := reflect.TypeOf(v)
		if reflect.Ptr == t.Kind() {
			t = reflect.TypeOf(reflect.ValueOf(v).Elem())
		}
		switch t.Kind() {
		case reflect.Struct:
			p, ok := v.(hessian.Param)
			if ok {
				return p.JavaParamName()
			}
			v, ok := v.(hessian.POJO)
			if ok {
				return v.JavaClassName()
			}
			return "java.lang.Object"
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Struct {
				return "[Ljava.lang.Object;"
			}
			// return "java.util.ArrayList"
			return "java.util.List"
		case reflect.Map: // Enter here, map may be map[string]int
			return "java.util.Map"
		default:
			return ""
		}
	}

	// unreachable
	// return "java.lang.RuntimeException"
}

func ToMapStringInterface(origin map[interface{}]interface{}) map[string]interface{} {
	dest := make(map[string]interface{}, len(origin))
	for k, v := range origin {
		if kv, ok := k.(string); ok {
			if v == nil {
				dest[kv] = ""
				continue
			}
			dest[kv] = v
		}
	}
	return dest
}

func init() {
	SetSerializer("hessian2", HessianSerializer{})
}

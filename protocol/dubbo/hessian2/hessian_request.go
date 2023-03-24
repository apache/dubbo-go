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
	"strconv"
	"strings"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

func getArgType(v interface{}) string {
	return GetClassDesc(v)
}

func getArgsTypeList(args []interface{}) (string, error) {
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

type DubboRequest struct {
	Params      interface{}
	Attachments map[string]interface{}
}

// NewRequest create a new DubboRequest
func NewRequest(params interface{}, atta map[string]interface{}) *DubboRequest {
	if atta == nil {
		atta = make(map[string]interface{})
	}
	return &DubboRequest{
		Params:      params,
		Attachments: atta,
	}
}

func EnsureRequest(body interface{}) *DubboRequest {
	if req, ok := body.(*DubboRequest); ok {
		return req
	}
	return NewRequest(body, nil)
}

func packRequest(service Service, header DubboHeader, req interface{}) ([]byte, error) {
	var (
		err       error
		types     string
		byteArray []byte
		pkgLen    int
	)

	request := EnsureRequest(req)

	args, ok := request.Params.([]interface{})
	if !ok {
		return nil, perrors.Errorf("@params is not of type: []interface{}")
	}

	hb := header.Type == PackageHeartbeat

	//////////////////////////////////////////
	// byteArray
	//////////////////////////////////////////
	// magic
	switch header.Type {
	case PackageHeartbeat:
		byteArray = append(byteArray, DubboRequestHeartbeatHeader[:]...)
	case PackageRequest_TwoWay:
		byteArray = append(byteArray, DubboRequestHeaderBytesTwoWay[:]...)
	default:
		byteArray = append(byteArray, DubboRequestHeaderBytes[:]...)
	}

	// serialization id, two way flag, event, request/response flag
	// SerialID is id of serialization approach in java dubbo
	byteArray[2] |= header.SerialID & SERIAL_MASK
	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))

	encoder := hessian.NewEncoder()
	encoder.Append(byteArray[:HEADER_LENGTH])

	//////////////////////////////////////////
	// body
	//////////////////////////////////////////
	if hb {
		_ = encoder.Encode(nil)
		goto END
	}

	// dubbo version + path + version + method
	if err = encoder.Encode(DEFAULT_DUBBO_PROTOCOL_VERSION); err != nil {
		logger.Warnf("Encode(DEFAULT_DUBBO_PROTOCOL_VERSION) = error: %v", err)
	}
	if err = encoder.Encode(service.Path); err != nil {
		logger.Warnf("Encode(service.Path) = error: %v", err)
	}
	if err = encoder.Encode(service.Version); err != nil {
		logger.Warnf("Encode(service.Version) = error: %v", err)
	}
	if err = encoder.Encode(service.Method); err != nil {
		logger.Warnf("Encode(service.Method) = error: %v", err)
	}

	// args = args type list + args value list
	if types, err = getArgsTypeList(args); err != nil {
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

	_ = encoder.Encode(request.Attachments)

END:
	byteArray = encoder.Buffer()
	pkgLen = len(byteArray)
	if pkgLen > int(DEFAULT_LEN) { // recommand 8M
		logger.Warnf("Data length %d too large, recommand max payload %d. "+
			"Dubbo java can't handle the package whose size is greater than %d!!!", pkgLen, DEFAULT_LEN, DEFAULT_LEN)
	}
	// byteArray{body length}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen-HEADER_LENGTH))
	return byteArray, nil
}

// hessian decode request body
func unpackRequestBody(decoder *hessian.Decoder, reqObj interface{}) error {
	if decoder == nil {
		return perrors.Errorf("@decoder is nil")
	}

	req, ok := reqObj.([]interface{})
	if !ok {
		return perrors.Errorf("@reqObj is not of type: []interface{}")
	}
	if len(req) < 7 {
		return perrors.New("length of @reqObj should  be 7")
	}

	var (
		err                                                     error
		dubboVersion, target, serviceVersion, method, argsTypes interface{}
		args                                                    []interface{}
	)

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

	ats := DescRegex.FindAllString(argsTypes.(string), -1)
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
	if v, ok := attachments.(map[interface{}]interface{}); ok {
		v[DUBBO_VERSION_KEY] = dubboVersion
		req[6] = ToMapStringInterface(v)
		return nil
	}

	return perrors.Errorf("get wrong attachments: %+v", attachments)
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

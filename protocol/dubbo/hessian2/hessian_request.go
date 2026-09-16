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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

func getArgType(v any) string {
	return GetClassDesc(v)
}

func getArgsTypeList(args []any) (string, error) {
	var (
		typ   string
		types string
	)

	for i := range args {
		typ = getArgType(args[i])
		if typ == "" {
			return types, fmt.Errorf("cat not get arg %#v type", args[i])
		}
		if !strings.Contains(typ, ".") {
			types += typ
		} else if strings.Index(typ, "[") == 0 {
			types += strings.ReplaceAll(typ, ".", "/")
		} else {
			// java.util.List -> Ljava/util/List;
			types += "L" + strings.ReplaceAll(typ, ".", "/") + ";"
		}
	}

	return types, nil
}

type DubboRequest struct {
	Params      any
	Attachments map[string]any
}

// NewRequest create a new DubboRequest
func NewRequest(params any, atta map[string]any) *DubboRequest {
	if atta == nil {
		atta = make(map[string]any)
	}
	return &DubboRequest{
		Params:      params,
		Attachments: atta,
	}
}

func EnsureRequest(body any) *DubboRequest {
	if req, ok := body.(*DubboRequest); ok {
		return req
	}
	return NewRequest(body, nil)
}

func packRequest(service Service, header DubboHeader, req any) ([]byte, error) {
	var (
		byteArray []byte
		pkgLen    int
	)

	request := EnsureRequest(req)

	args, ok := request.Params.([]any)
	if !ok {
		return nil, fmt.Errorf("@params is not of type: []any")
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
		if err := encoder.Encode(nil); err != nil {
			return nil, fmt.Errorf("failed to encode heartbeat request: %w", err)
		}
	} else {
		if err := encodeRequestBody(encoder, service, request, args); err != nil {
			return nil, err
		}
	}

	byteArray = encoder.Buffer()
	pkgLen = len(byteArray)
	if pkgLen > int(DEFAULT_LEN) { // recommand 8M
		logger.Warnf("[Dubbo][Hessian2] data length %d too large, recommand max payload %d. "+
			"Dubbo java can't handle the package whose size is greater than %d!!!", pkgLen, DEFAULT_LEN, DEFAULT_LEN)
	}
	// byteArray{body length}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen-HEADER_LENGTH))
	return byteArray, nil
}

func encodeRequestBody(encoder *hessian.Encoder, service Service, request *DubboRequest, args []any) error {
	// dubbo version + path + version + method
	if err := encoder.Encode(DEFAULT_DUBBO_PROTOCOL_VERSION); err != nil {
		return fmt.Errorf("failed to encode default dubbo protocol version: %w", err)
	}
	if err := encoder.Encode(service.Path); err != nil {
		return fmt.Errorf("failed to encode service path: %w", err)
	}
	if err := encoder.Encode(service.Version); err != nil {
		return fmt.Errorf("failed to encode service version: %w", err)
	}
	if err := encoder.Encode(service.Method); err != nil {
		return fmt.Errorf("failed to encode service method: %w", err)
	}

	// args = args type list + args value list
	types, err := getArgsTypeList(args)
	if err != nil {
		return fmt.Errorf(" PackRequest(args:%+v): %w", args, err)
	}
	if err := encoder.Encode(types); err != nil {
		return fmt.Errorf("failed to encode argument types: %w", err)
	}
	for _, v := range args {
		if err := encoder.Encode(v); err != nil {
			return fmt.Errorf("failed to encode argument of type %T: %w", v, err)
		}
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

	if err := encoder.Encode(request.Attachments); err != nil {
		return fmt.Errorf("failed to encode request attachments: %w", err)
	}
	return nil
}

// hessian decode request body
func unpackRequestBody(decoder *hessian.Decoder, reqObj any) error {
	if decoder == nil {
		return fmt.Errorf("@decoder is nil")
	}

	req, ok := reqObj.([]any)
	if !ok {
		return fmt.Errorf("@reqObj is not of type: []any")
	}
	if len(req) < 7 {
		return errors.New("length of @reqObj should  be 7")
	}

	var (
		err                                                     error
		dubboVersion, target, serviceVersion, method, argsTypes any
		args                                                    []any
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
	var arg any
	for range ats {
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
	if v, ok := attachments.(map[any]any); ok {
		v[DUBBO_VERSION_KEY] = dubboVersion
		req[6] = ToMapStringInterface(v)
		return nil
	}

	return fmt.Errorf("get wrong attachments: %+v", attachments)
}

func ToMapStringInterface(origin map[any]any) map[string]any {
	dest := make(map[string]any, len(origin))
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

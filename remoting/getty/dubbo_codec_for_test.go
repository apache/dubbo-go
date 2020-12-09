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
package getty

// copy from dubbo/dubbo_codec.go .
// it is used to unit test.
import (
	"bytes"
	"strconv"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
)

func init() {
	codec := &DubboTestCodec{}
	remoting.RegistryCodec("dubbo", codec)
}

type DubboTestCodec struct {
}

// encode request for transport
func (c *DubboTestCodec) EncodeRequest(request *remoting.Request) (*bytes.Buffer, error) {
	if request.Event {
		return c.encodeHeartbeartReqeust(request)
	}

	invoc, ok := request.Data.(*invocation.RPCInvocation)
	if !ok {
		return nil, perrors.Errorf("encode request failed for parameter type :%+v", request)
	}
	tmpInvocation := invoc

	svc := impl.Service{}
	svc.Path = tmpInvocation.AttachmentsByKey(constant.PATH_KEY, "")
	svc.Interface = tmpInvocation.AttachmentsByKey(constant.INTERFACE_KEY, "")
	svc.Version = tmpInvocation.AttachmentsByKey(constant.VERSION_KEY, "")
	svc.Group = tmpInvocation.AttachmentsByKey(constant.GROUP_KEY, "")
	svc.Method = tmpInvocation.MethodName()
	timeout, err := strconv.Atoi(tmpInvocation.AttachmentsByKey(constant.TIMEOUT_KEY, strconv.Itoa(constant.DEFAULT_REMOTING_TIMEOUT)))
	if err != nil {
		// it will be wrapped in readwrite.Write .
		return nil, perrors.WithStack(err)
	}
	svc.Timeout = time.Duration(timeout)

	header := impl.DubboHeader{}
	serialization := tmpInvocation.AttachmentsByKey(constant.SERIALIZATION_KEY, constant.HESSIAN2_SERIALIZATION)
	if serialization == constant.PROTOBUF_SERIALIZATION {
		header.SerialID = constant.S_Proto
	} else {
		header.SerialID = constant.S_Hessian2
	}
	header.ID = request.ID
	if request.TwoWay {
		header.Type = impl.PackageRequest_TwoWay
	} else {
		header.Type = impl.PackageRequest
	}

	pkg := &impl.DubboPackage{
		Header:  header,
		Service: svc,
		Body:    impl.NewRequestPayload(tmpInvocation.Arguments(), tmpInvocation.Attachments()),
		Err:     nil,
		Codec:   impl.NewDubboCodec(nil),
	}

	if err := impl.LoadSerializer(pkg); err != nil {
		return nil, perrors.WithStack(err)
	}

	return pkg.Marshal()
}

// encode heartbeart request
func (c *DubboTestCodec) encodeHeartbeartReqeust(request *remoting.Request) (*bytes.Buffer, error) {
	header := impl.DubboHeader{
		Type:     impl.PackageHeartbeat,
		SerialID: constant.S_Hessian2,
		ID:       request.ID,
	}

	pkg := &impl.DubboPackage{
		Header:  header,
		Service: impl.Service{},
		Body:    impl.NewRequestPayload([]interface{}{}, nil),
		Err:     nil,
		Codec:   impl.NewDubboCodec(nil),
	}

	if err := impl.LoadSerializer(pkg); err != nil {
		return nil, err
	}
	return pkg.Marshal()
}

// encode response
func (c *DubboTestCodec) EncodeResponse(response *remoting.Response) (*bytes.Buffer, error) {
	var ptype = impl.PackageResponse
	if response.IsHeartbeat() {
		ptype = impl.PackageHeartbeat
	}
	resp := &impl.DubboPackage{
		Header: impl.DubboHeader{
			SerialID:       response.SerialID,
			Type:           ptype,
			ID:             response.ID,
			ResponseStatus: response.Status,
		},
	}
	if !response.IsHeartbeat() {
		resp.Body = &impl.ResponsePayload{
			RspObj:      response.Result.(protocol.RPCResult).Rest,
			Exception:   response.Result.(protocol.RPCResult).Err,
			Attachments: response.Result.(protocol.RPCResult).Attrs,
		}
	}

	codec := impl.NewDubboCodec(nil)

	pkg, err := codec.Encode(*resp)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	return bytes.NewBuffer(pkg), nil
}

// Decode data, including request and response.
func (c *DubboTestCodec) Decode(data []byte) (remoting.DecodeResult, int, error) {
	if c.isRequest(data) {
		req, len, err := c.decodeRequest(data)
		if err != nil {
			return remoting.DecodeResult{}, len, perrors.WithStack(err)
		}
		return remoting.DecodeResult{IsRequest: true, Result: req}, len, perrors.WithStack(err)
	} else {
		resp, len, err := c.decodeResponse(data)
		if err != nil {
			return remoting.DecodeResult{}, len, perrors.WithStack(err)
		}
		return remoting.DecodeResult{IsRequest: false, Result: resp}, len, perrors.WithStack(err)
	}
}

func (c *DubboTestCodec) isRequest(data []byte) bool {
	if data[2]&byte(0x80) == 0x00 {
		return false
	}
	return true
}

// decode request
func (c *DubboTestCodec) decodeRequest(data []byte) (*remoting.Request, int, error) {
	var request *remoting.Request = nil
	buf := bytes.NewBuffer(data)
	pkg := impl.NewDubboPackage(buf)
	pkg.SetBody(make([]interface{}, 7))
	err := pkg.Unmarshal()
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			//FIXME
			return nil, 0, originErr
		}
		return request, 0, perrors.WithStack(err)
	}
	request = &remoting.Request{
		ID:       pkg.Header.ID,
		SerialID: pkg.Header.SerialID,
		TwoWay:   pkg.Header.Type&impl.PackageRequest_TwoWay != 0x00,
		Event:    pkg.Header.Type&impl.PackageHeartbeat != 0x00,
	}
	if (pkg.Header.Type & impl.PackageHeartbeat) == 0x00 {
		// convert params of request
		req := pkg.Body.(map[string]interface{})

		//invocation := request.Data.(*invocation.RPCInvocation)
		var methodName string
		var args []interface{}
		attachments := make(map[string]interface{})
		if req[impl.DubboVersionKey] != nil {
			//dubbo version
			request.Version = req[impl.DubboVersionKey].(string)
		}
		//path
		attachments[constant.PATH_KEY] = pkg.Service.Path
		//version
		attachments[constant.VERSION_KEY] = pkg.Service.Version
		//method
		methodName = pkg.Service.Method
		args = req[impl.ArgsKey].([]interface{})
		attachments = req[impl.AttachmentsKey].(map[string]interface{})
		invoc := invocation.NewRPCInvocationWithOptions(invocation.WithAttachments(attachments),
			invocation.WithArguments(args), invocation.WithMethodName(methodName))
		request.Data = invoc

	}
	return request, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

// decode response
func (c *DubboTestCodec) decodeResponse(data []byte) (*remoting.Response, int, error) {
	buf := bytes.NewBuffer(data)
	pkg := impl.NewDubboPackage(buf)
	response := &remoting.Response{}
	err := pkg.Unmarshal()
	if err != nil {
		originErr := perrors.Cause(err)
		// if the data is very big, so the receive need much times.
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, originErr
		}
		return nil, 0, perrors.WithStack(err)
	}
	response = &remoting.Response{
		ID: pkg.Header.ID,
		//Version:  pkg.Header.,
		SerialID: pkg.Header.SerialID,
		Status:   pkg.Header.ResponseStatus,
		Event:    (pkg.Header.Type & impl.PackageHeartbeat) != 0,
	}
	var error error
	if pkg.Header.Type&impl.PackageHeartbeat != 0x00 {
		if pkg.Header.Type&impl.PackageResponse != 0x00 {
			if pkg.Err != nil {
				error = pkg.Err
			}
		} else {
			response.Status = hessian.Response_OK
			//reply(session, p, hessian.PackageHeartbeat)
		}
		return response, hessian.HEADER_LENGTH + pkg.Header.BodyLen, error
	}
	rpcResult := &protocol.RPCResult{}
	response.Result = rpcResult
	if pkg.Header.Type&impl.PackageRequest == 0x00 {
		if pkg.Err != nil {
			rpcResult.Err = pkg.Err
		} else if pkg.Body.(*impl.ResponsePayload).Exception != nil {
			rpcResult.Err = pkg.Body.(*impl.ResponsePayload).Exception
			response.Error = rpcResult.Err
		}
		rpcResult.Attrs = pkg.Body.(*impl.ResponsePayload).Attachments
		rpcResult.Rest = pkg.Body.(*impl.ResponsePayload).RspObj
	}

	return response, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

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

package dubbo

import (
	"bytes"
	"strconv"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/impl"
	invct "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// SerialID serial ID
type SerialID byte

func init() {
	codec := &DubboCodec{}
	remoting.RegistryCodec("dubbo", codec)
}

// DubboCodec  implements remoting.Codec
type DubboCodec struct{}

// EncodeRequest encodes request for transport
func (c *DubboCodec) EncodeRequest(request *remoting.Request) (*bytes.Buffer, error) {
	if request.Event {
		return c.encodeHeartbeartReqeust(request)
	}

	invoc, ok := request.Data.(*protocol.Invocation)
	if !ok {
		err := perrors.Errorf("encode request failed for parameter type :%+v", request)
		logger.Errorf(err.Error())
		return nil, err
	}
	invocation := *invoc

	svc := impl.Service{}
	svc.Path = invocation.GetAttachmentWithDefaultValue(constant.PathKey, "")
	svc.Interface = invocation.GetAttachmentWithDefaultValue(constant.InterfaceKey, "")
	svc.Version = invocation.GetAttachmentWithDefaultValue(constant.VersionKey, "")
	svc.Group = invocation.GetAttachmentWithDefaultValue(constant.GroupKey, "")
	svc.Method = invocation.MethodName()
	timeout, err := strconv.Atoi(invocation.GetAttachmentWithDefaultValue(constant.TimeoutKey, strconv.Itoa(constant.DefaultRemotingTimeout)))
	if err != nil {
		// it will be wrapped in readwrite.Write .
		return nil, perrors.WithStack(err)
	}
	svc.Timeout = time.Duration(timeout)

	header := impl.DubboHeader{}
	serialization := invocation.GetAttachmentWithDefaultValue(constant.SerializationKey, constant.Hessian2Serialization)
	if serialization == constant.ProtobufSerialization {
		header.SerialID = constant.SProto
	} else {
		header.SerialID = constant.SHessian2
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
		Body:    impl.NewRequestPayload(invocation.Arguments(), invocation.Attachments()),
		Err:     nil,
		Codec:   impl.NewDubboCodec(nil),
	}

	if err := impl.LoadSerializer(pkg); err != nil {
		return nil, perrors.WithStack(err)
	}

	return pkg.Marshal()
}

// encode heartbeat request
func (c *DubboCodec) encodeHeartbeartReqeust(request *remoting.Request) (*bytes.Buffer, error) {
	header := impl.DubboHeader{
		Type:     impl.PackageHeartbeat,
		SerialID: constant.SHessian2,
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

// EncodeResponse encodes response
func (c *DubboCodec) EncodeResponse(response *remoting.Response) (*bytes.Buffer, error) {
	ptype := impl.PackageResponse
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
func (c *DubboCodec) Decode(data []byte) (*remoting.DecodeResult, int, error) {
	dataLen := len(data)
	if dataLen < impl.HEADER_LENGTH { // check whether header bytes is enough or not
		return nil, 0, nil
	}
	if c.isRequest(data) {
		req, length, err := c.decodeRequest(data)
		if err != nil {
			return nil, length, perrors.WithStack(err)
		}
		if req == ((*remoting.Request)(nil)) {
			return nil, length, err
		}
		return &remoting.DecodeResult{IsRequest: true, Result: req}, length, perrors.WithStack(err)
	}

	rsp, length, err := c.decodeResponse(data)
	if err != nil {
		return nil, length, perrors.WithStack(err)
	}
	if rsp == ((*remoting.Response)(nil)) {
		return nil, length, err
	}
	return &remoting.DecodeResult{IsRequest: false, Result: rsp}, length, perrors.WithStack(err)
}

func (c *DubboCodec) isRequest(data []byte) bool {
	return data[2]&byte(0x80) != 0x00
}

// decode request
func (c *DubboCodec) decodeRequest(data []byte) (*remoting.Request, int, error) {
	var request *remoting.Request
	buf := bytes.NewBuffer(data)
	pkg := impl.NewDubboPackage(buf)
	pkg.SetBody(make([]interface{}, 7))
	err := pkg.Unmarshal()
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough { // this is impossible, as dubbo_codec.go:DubboCodec::Decode() line 167
			return nil, 0, nil
		}
		if originErr == hessian.ErrBodyNotEnough {
			return nil, hessian.HEADER_LENGTH + pkg.GetBodyLen(), nil
		}
		logger.Errorf("pkg.Unmarshal(len(@data):%d) = error:%+v", buf.Len(), err)

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

		// invocation := request.Data.(*invocation.RPCInvocation)
		var methodName string
		var args []interface{}
		attachments := make(map[string]interface{})
		if req[impl.DubboVersionKey] != nil {
			// dubbo version
			request.Version = req[impl.DubboVersionKey].(string)
		}
		// path
		attachments[constant.PathKey] = pkg.Service.Path
		// version
		attachments[constant.VersionKey] = pkg.Service.Version
		// method
		methodName = pkg.Service.Method
		args = req[impl.ArgsKey].([]interface{})
		attachments = req[impl.AttachmentsKey].(map[string]interface{})
		invoc := invct.NewRPCInvocationWithOptions(invct.WithAttachments(attachments),
			invct.WithArguments(args), invct.WithMethodName(methodName))
		request.Data = invoc

	}
	return request, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

// decode response
func (c *DubboCodec) decodeResponse(data []byte) (*remoting.Response, int, error) {
	buf := bytes.NewBuffer(data)
	pkg := impl.NewDubboPackage(buf)
	err := pkg.Unmarshal()
	if err != nil {
		originErr := perrors.Cause(err)
		// if the data is very big, so the receive need much times.
		if originErr == hessian.ErrHeaderNotEnough { // this is impossible, as dubbo_codec.go:DubboCodec::Decode() line 167
			return nil, 0, nil
		}
		if originErr == hessian.ErrBodyNotEnough {
			return nil, hessian.HEADER_LENGTH + pkg.GetBodyLen(), nil
		}

		logger.Warnf("pkg.Unmarshal(len(@data):%d) = error:%+v", buf.Len(), err)
		return nil, 0, perrors.WithStack(err)
	}
	response := &remoting.Response{
		ID: pkg.Header.ID,
		// Version:  pkg.Header.,
		SerialID: pkg.Header.SerialID,
		Status:   pkg.Header.ResponseStatus,
		Event:    (pkg.Header.Type & impl.PackageHeartbeat) != 0,
	}
	var pkgerr error
	if pkg.Header.Type&impl.PackageHeartbeat != 0x00 {
		if pkg.Header.Type&impl.PackageResponse != 0x00 {
			logger.Debugf("get rpc heartbeat response{header: %#v, body: %#v}", pkg.Header, pkg.Body)
			if pkg.Err != nil {
				logger.Errorf("rpc heartbeat response{error: %#v}", pkg.Err)
				pkgerr = pkg.Err
			}
		} else {
			logger.Debugf("get rpc heartbeat request{header: %#v, service: %#v, body: %#v}", pkg.Header, pkg.Service, pkg.Body)
			response.Status = hessian.Response_OK
			// reply(session, p, hessian.PackageHeartbeat)
		}
		return response, hessian.HEADER_LENGTH + pkg.Header.BodyLen, pkgerr
	}
	logger.Debugf("get rpc response{header: %#v, body: %#v}", pkg.Header, pkg.Body)
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

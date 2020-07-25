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
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"strconv"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
)

//SerialID serial ID
type SerialID byte

const (
	// S_Dubbo dubbo serial id
	S_Dubbo SerialID = 2
)

func init() {
	codec := &DubboCodec{}
	// this is for registry dubboCodec of dubbo protocol
	remoting.RegistryCodec("dubbo", codec)
}

// DubboPackage is for hessian encode/decode. If we refactor hessian, it will also be refactored.
//type DubboPackage struct {
//	Header  hessian.DubboHeader
//	Service hessian.Service
//	Body    interface{}
//	Err     error
//}

// String of DubboPackage
//func (p DubboPackage) String() string {
//	return fmt.Sprintf("DubboPackage: Header-%v, Path-%v, Body-%v", p.Header, p.Service, p.Body)
//}

//  nolint
//func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
//	codec := hessian.NewHessianCodec(nil)
//
//	pkg, err := codec.Write(p.Service, p.Header, p.Body)
//	if err != nil {
//		return nil, perrors.WithStack(err)
//	}
//
//	return bytes.NewBuffer(pkg), nil
//}

// nolint
//func (p *DubboPackage) Unmarshal(buf *bytes.Buffer, resp *remoting.Response) error {
//	// fix issue https://github.com/apache/dubbo-go/issues/380
//	bufLen := buf.Len()
//	if bufLen < hessian.HEADER_LENGTH {
//		return perrors.WithStack(hessian.ErrHeaderNotEnough)
//	}
//
//	codec := hessian.NewHessianCodec(bufio.NewReaderSize(buf, bufLen))
//
//	// read header
//	err := codec.ReadHeader(&p.Header)
//	if err != nil {
//		return perrors.WithStack(err)
//	}
//
//	if resp != nil { // for client
//		if (p.Header.Type & hessian.PackageRequest) != 0x00 {
//			// size of this array must be '7'
//			// https://github.com/apache/dubbo-go-hessian2/blob/master/request.go#L272
//			p.Body = make([]interface{}, 7)
//		} else {
//			pendingRsp := remoting.GetPendingResponse(remoting.SequenceType(p.Header.ID))
//			if pendingRsp == nil {
//				return perrors.Errorf("client.GetPendingResponse(%v) = nil", p.Header.ID)
//			}
//			p.Body = &hessian.Response{RspObj: pendingRsp.Reply}
//		}
//	}
//	// read body
//	err = codec.ReadBody(p.Body)
//	return perrors.WithStack(err)
//}

// DubboCodec. It is implements remoting.Codec
type DubboCodec struct {
}

// encode request for transport
func (c *DubboCodec) EncodeRequest(request *remoting.Request) (*bytes.Buffer, error) {
	if request.Event {
		return c.encodeHeartbeartReqeust(request)
	}

	invoc, ok := request.Data.(*protocol.Invocation)
	if !ok {
		logger.Errorf("encode request failed for parameter type :%+v", request)
		return nil, perrors.Errorf("encode request failed for parameter type :%+v", request)
	}
	invocation := *invoc

	svc := impl.Service{}
	svc.Path = invocation.AttachmentsByKey(constant.PATH_KEY, "")
	svc.Interface = invocation.AttachmentsByKey(constant.INTERFACE_KEY, "")
	svc.Version = invocation.AttachmentsByKey(constant.VERSION_KEY, "")
	svc.Group = invocation.AttachmentsByKey(constant.GROUP_KEY, "")
	svc.Method = invocation.MethodName()
	timeout, err := strconv.Atoi(invocation.AttachmentsByKey(constant.TIMEOUT_KEY, strconv.Itoa(constant.DEFAULT_REMOTING_TIMEOUT)))
	if err != nil {
		// it will be wrapped in readwrite.Write .
		return nil, perrors.WithStack(err)
	}
	svc.Timeout = time.Duration(timeout)

	header := impl.DubboHeader{}
	serialization := invocation.AttachmentsByKey(constant.SERIALIZATION_KEY, constant.HESSIAN2_SERIALIZATION)
	if serialization == constant.HESSIAN2_SERIALIZATION {
		header.SerialID = constant.S_Hessian2
	} else if serialization == constant.PROTOBUF_SERIALIZATION {
		header.SerialID = constant.S_Proto
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

// encode heartbeart request
func (c *DubboCodec) encodeHeartbeartReqeust(request *remoting.Request) (*bytes.Buffer, error) {
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
func (c *DubboCodec) EncodeResponse(response *remoting.Response) (*bytes.Buffer, error) {
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
		resp.Body = &hessian.Response{
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
func (c *DubboCodec) Decode(data []byte) (remoting.DecodeResult, int, error) {
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

func (c *DubboCodec) isRequest(data []byte) bool {
	if data[2]&byte(0x80) == 0x00 {
		return false
	}
	return true
}

// decode request
func (c *DubboCodec) decodeRequest(data []byte) (*remoting.Request, int, error) {
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
		req := pkg.Body.([]interface{}) // length of body should be 7
		if len(req) > 0 {
			//invocation := request.Data.(*invocation.RPCInvocation)
			var methodName string
			var args []interface{}
			var attachments map[string]string = make(map[string]string)
			if req[0] != nil {
				//dubbo version
				request.Version = req[0].(string)
			}
			if req[1] != nil {
				//path
				attachments[constant.PATH_KEY] = req[1].(string)
			}
			if req[2] != nil {
				//version
				attachments[constant.VERSION_KEY] = req[2].(string)
			}
			if req[3] != nil {
				//method
				methodName = req[3].(string)
			}
			if req[4] != nil {
				//ignore argTypes
			}
			if req[5] != nil {
				args = req[5].([]interface{})
			}
			if req[6] != nil {
				attachments = req[6].(map[string]string)
			}
			invoc := invocation.NewRPCInvocationWithOptions(invocation.WithAttachments(attachments),
				invocation.WithArguments(args), invocation.WithMethodName(methodName))
			request.Data = invoc
		}
	}
	return request, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

// decode response
func (c *DubboCodec) decodeResponse(data []byte) (*remoting.Response, int, error) {
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
		logger.Errorf("pkg.Unmarshal(len(@data):%d) = error:%+v", buf.Len(), err)

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
			logger.Debugf("get rpc heartbeat response{header: %#v, body: %#v}", pkg.Header, pkg.Body)
			if pkg.Err != nil {
				logger.Errorf("rpc heartbeat response{error: %#v}", pkg.Err)
				error = pkg.Err
			}
		} else {
			logger.Debugf("get rpc heartbeat request{header: %#v, service: %#v, body: %#v}", pkg.Header, pkg.Service, pkg.Body)
			response.Status = hessian.Response_OK
			//reply(session, p, hessian.PackageHeartbeat)
		}
		return response, hessian.HEADER_LENGTH + pkg.Header.BodyLen, error
	}
	logger.Debugf("get rpc response{header: %#v, body: %#v}", pkg.Header, pkg.Body)
	rpcResult := &protocol.RPCResult{}
	response.Result = rpcResult
	if pkg.Header.Type&impl.PackageRequest == 0x00 {
		if pkg.Err != nil {
			rpcResult.Err = pkg.Err
		} else if pkg.Body.(*hessian.Response).Exception != nil {
			rpcResult.Err = pkg.Body.(*hessian.Response).Exception
			response.Error = rpcResult.Err
		}
		rpcResult.Attrs = pkg.Body.(*hessian.Response).Attachments
		rpcResult.Rest = pkg.Body.(*hessian.Response).RspObj
	}

	return response, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

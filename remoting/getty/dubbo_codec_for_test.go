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
	"bufio"
	"bytes"
	"fmt"
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
	codec := &DubboTestCodec{}
	remoting.RegistryCodec("dubbo", codec)
}

// DubboPackage ...
type DubboPackage struct {
	Header  hessian.DubboHeader
	Service hessian.Service
	Body    interface{}
	Err     error
}

func (p DubboPackage) String() string {
	return fmt.Sprintf("DubboPackage: Header-%v, Path-%v, Body-%v", p.Header, p.Service, p.Body)
}

// Marshal ...
func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	codec := hessian.NewHessianCodec(nil)

	pkg, err := codec.Write(p.Service, p.Header, p.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	return bytes.NewBuffer(pkg), nil
}

// Unmarshal ...
func (p *DubboPackage) Unmarshal(buf *bytes.Buffer, resp *remoting.Response) error {
	// fix issue https://github.com/apache/dubbo-go/issues/380
	bufLen := buf.Len()
	if bufLen < hessian.HEADER_LENGTH {
		return perrors.WithStack(hessian.ErrHeaderNotEnough)
	}

	codec := hessian.NewHessianCodec(bufio.NewReaderSize(buf, bufLen))

	// read header
	err := codec.ReadHeader(&p.Header)
	if err != nil {
		return perrors.WithStack(err)
	}

	if resp != nil { // for client
		if (p.Header.Type & hessian.PackageRequest) != 0x00 {
			// size of this array must be '7'
			// https://github.com/apache/dubbo-go-hessian2/blob/master/request.go#L272
			p.Body = make([]interface{}, 7)
		} else {
			pendingRsp := remoting.GetPendingResponse(remoting.SequenceType(p.Header.ID))
			if pendingRsp == nil {
				return perrors.Errorf("client.GetPendingResponse(%v) = nil", p.Header.ID)
			}
			p.Body = &hessian.Response{RspObj: pendingRsp.Reply}
		}
	}

	// read body
	err = codec.ReadBody(p.Body)
	return perrors.WithStack(err)
}

type DubboTestCodec struct {
}

func (c *DubboTestCodec) EncodeRequest(request *remoting.Request) (*bytes.Buffer, error) {
	if request.Event {
		return c.encodeHeartbeartReqeust(request)
	}

	invoc, ok := request.Data.(*invocation.RPCInvocation)
	if !ok {
		logger.Errorf("encode request failed for parameter type :%+v", request)
		return nil, perrors.Errorf("encode request failed for parameter type :%+v", request)
	}
	invocation := *invoc

	p := &DubboPackage{}
	p.Service.Path = invocation.AttachmentsByKey(constant.PATH_KEY, "")
	p.Service.Interface = invocation.AttachmentsByKey(constant.INTERFACE_KEY, "")
	p.Service.Version = invocation.AttachmentsByKey(constant.VERSION_KEY, "")
	p.Service.Group = invocation.AttachmentsByKey(constant.GROUP_KEY, "")
	p.Service.Method = invocation.MethodName()

	timeout, err := strconv.Atoi(invocation.AttachmentsByKey(constant.TIMEOUT_KEY, "3000"))
	if err != nil {
		// it will be wrapped in readwrite.Write .
		return nil, err
	}
	p.Service.Timeout = time.Duration(timeout)
	//var timeout = request.svcUrl.GetParam(strings.Join([]string{constant.METHOD_KEYS, request.method + constant.RETRIES_KEY}, "."), "")
	//if len(timeout) != 0 {
	//	if t, err := time.ParseDuration(timeout); err == nil {
	//		p.Service.Timeout = t
	//	}
	//}

	p.Header.SerialID = byte(S_Dubbo)
	p.Header.ID = request.ID
	if request.TwoWay {
		p.Header.Type = hessian.PackageRequest_TwoWay
	} else {
		p.Header.Type = hessian.PackageRequest
	}

	p.Body = hessian.NewRequest(invocation.Arguments(), invocation.Attachments())

	codec := hessian.NewHessianCodec(nil)

	pkg, err := codec.Write(p.Service, p.Header, p.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	return bytes.NewBuffer(pkg), nil
}
func (c *DubboTestCodec) encodeHeartbeartReqeust(request *remoting.Request) (*bytes.Buffer, error) {
	pkg := &DubboPackage{}
	pkg.Body = []interface{}{}
	pkg.Header.ID = request.ID
	pkg.Header.Type = hessian.PackageHeartbeat
	pkg.Header.SerialID = byte(S_Dubbo)

	codec := hessian.NewHessianCodec(nil)

	byt, err := codec.Write(pkg.Service, pkg.Header, pkg.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	return bytes.NewBuffer(byt), nil
}
func (c *DubboTestCodec) EncodeResponse(response *remoting.Response) (*bytes.Buffer, error) {
	var ptype = hessian.PackageResponse
	if response.IsHeartbeat() {
		ptype = hessian.PackageHeartbeat
	}
	resp := &DubboPackage{
		Header: hessian.DubboHeader{
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

	//if response.Header.Type&hessian.PackageRequest != 0x00 {
	//	resp.Body = req.Body
	//} else {
	//	resp.Body = nil
	//}
	codec := hessian.NewHessianCodec(nil)

	pkg, err := codec.Write(resp.Service, resp.Header, resp.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	return bytes.NewBuffer(pkg), nil
}
func (c *DubboTestCodec) Decode(data []byte) (remoting.DecodeResult, int, error) {
	if c.isRequest(data) {
		req, len, err := c.decodeRequest(data)
		if err != nil {
			return remoting.DecodeResult{}, len, err
		}
		return remoting.DecodeResult{IsRequest: true, Result: req}, len, err
	} else {
		resp, len, err := c.decodeResponse(data)
		if err != nil {
			return remoting.DecodeResult{}, len, err
		}
		return remoting.DecodeResult{IsRequest: false, Result: resp}, len, err
	}
}
func (c *DubboTestCodec) isRequest(data []byte) bool {
	if data[2]&byte(0x80) == 0x00 {
		return false
	}
	return true
}

func (c *DubboTestCodec) decodeRequest(data []byte) (*remoting.Request, int, error) {
	pkg := &DubboPackage{
		Body: make([]interface{}, 7),
	}
	var request *remoting.Request = nil
	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf, nil)
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
		TwoWay:   pkg.Header.Type&hessian.PackageRequest_TwoWay != 0x00,
		Event:    pkg.Header.Type&hessian.PackageHeartbeat != 0x00,
	}
	if pkg.Header.Type&hessian.PackageHeartbeat == 0x00 {
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
				//argsType
				//invocation.ParameterTypes(constant., req[1].(string))
				//argsTypes = req[4].(string)
			}
			if req[5] != nil {
				args = req[5].([]interface{})
			}
			if req[6] != nil {
				attachments = req[6].(map[string]string)
			}
			//if pkg.Service.Path == "" && len(attachments[constant.PATH_KEY]) > 0 {
			//	pkg.Service.Path = attachments[constant.PATH_KEY]
			//}
			//if _, ok := attachments[constant.INTERFACE_KEY]; ok {
			//	pkg.Service.Interface = attachments[constant.INTERFACE_KEY]
			//} else {
			//	pkg.Service.Interface = pkg.Service.Path
			//}
			//if len(attachments[constant.GROUP_KEY]) > 0 {
			//	pkg.Service.Group = attachments[constant.GROUP_KEY]
			//}
			invoc := invocation.NewRPCInvocationWithOptions(invocation.WithAttachments(attachments),
				invocation.WithArguments(args), invocation.WithMethodName(methodName))
			request.Data = invoc
			//pkg.Body = map[string]interface{}{
			//	"dubboVersion": dubboVersion,
			//	"argsTypes":    argsTypes,
			//	"args":         args,
			//	"service":      common.ServiceMap.GetService("dubbo", pkg.Service.Path), // path as a key
			//	"attachments":  attachments,
			//}
		}
	}
	return request, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

func (c *DubboTestCodec) decodeResponse(data []byte) (*remoting.Response, int, error) {
	pkg := &DubboPackage{}
	buf := bytes.NewBuffer(data)
	response := &remoting.Response{}
	err := pkg.Unmarshal(buf, response)
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, originErr
		}
		logger.Errorf("pkg.Unmarshal(len(@data):%d) = error:%+v", buf.Len(), err)

		return response, 0, perrors.WithStack(err)
	}
	response = &remoting.Response{
		ID: pkg.Header.ID,
		//Version:  pkg.Header.,
		SerialID: pkg.Header.SerialID,
		Status:   pkg.Header.ResponseStatus,
		Event:    (pkg.Header.Type & hessian.PackageHeartbeat) != 0,
	}
	var error error
	if pkg.Header.Type&hessian.PackageHeartbeat != 0x00 {
		if pkg.Header.Type&hessian.PackageResponse != 0x00 {
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
	if pkg.Header.Type&hessian.PackageRequest == 0x00 {
		if pkg.Err != nil {
			rpcResult.Err = pkg.Err
		} else if pkg.Body.(*hessian.Response).Exception != nil {
			rpcResult.Err = pkg.Body.(*hessian.Response).Exception
			response.Error = rpcResult.Err
		}
		rpcResult.Attrs = pkg.Body.(*hessian.Response).Attachments
		rpcResult.Rest = pkg.Body.(*hessian.Response).RspObj
	}

	//h.conn.updateSession(session)
	//pendingResponse := h.conn.pool.rpcClient.removePendingResponse(SequenceType(p.Header.ID))
	//if pendingResponse == nil {
	//	logger.Errorf("failed to get pending response context for response package %s", *p)
	//	return
	//}

	return response, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

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
package remoting

import (
	"bufio"
	"bytes"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
)

////////////////////////////////////////////
// RpcClientPackageHandler
////////////////////////////////////////////

// RpcClientPackageHandler ...
type RpcClientPackageHandler struct {
	client *Client
}

// NewRpcClientPackageHandler ...
func NewRpcClientPackageHandler(client *Client) *RpcClientPackageHandler {
	return &RpcClientPackageHandler{client: client}
}

func (p *RpcClientPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := NewClientResponsePackage(data)
	if err := pkg.ReadHeader(); err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, nil
		}
		logger.Errorf("[RpcClientPackageHandler.Read] ss:%+v, len(@data):%d) = error:%+v ", ss, len(data), err)
		return nil, 0, perrors.WithStack(err)
	}
	if pkg.IsHeartBeat() {
		// heartbeat package doesn't need deserialize
		return pkg, pkg.GetLen(), nil
	}

	if err := impl.LoadSerializer(pkg); err != nil {
		return nil, 0, err
	}

	// load response
	pendingRsp, ok := p.client.PendingResponses.Load(impl.SequenceType(pkg.GetHeader().ID))
	if !ok {
		return nil, 0, perrors.Errorf("client.GetPendingResopnse(%v) = nil", pkg.GetHeader().ID)
	}
	// set package body
	body := impl.NewResponsePayload(pendingRsp.(*PendingResponse).response.Reply, nil, nil)
	pkg.SetBody(body)
	err := pkg.Unmarshal()
	if err != nil {
		return nil, 0, perrors.WithStack(err)
	}
	resp := pkg.Body.(*impl.ResponsePayload)
	pkg.Err = resp.Exception
	pkg.Body = NewResponse(resp.RspObj, resp.Attachments)
	return pkg, pkg.GetLen(), nil
}

func (p *RpcClientPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	req, ok := pkg.(*impl.DubboPackage)
	if !ok {
		return nil, perrors.New("invalid rpc request")
	}
	buf, err := req.Marshal()
	if err != nil {
		logger.Warnf("binary.Write(req{%#v}) = err{%#v}", req, perrors.WithStack(err))
		return nil, perrors.WithStack(err)
	}
	return buf.Bytes(), nil
}

func NewClientResponsePackage(data []byte) *impl.DubboPackage {
	return &impl.DubboPackage{
		Header:  impl.DubboHeader{},
		Service: impl.Service{},
		Body:    &impl.ResponsePayload{},
		Err:     nil,
		Codec:   impl.NewDubboCodec(bufio.NewReaderSize(bytes.NewBuffer(data), len(data))),
	}
}

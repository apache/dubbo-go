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
	"log"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/tools/cli/common"
	"github.com/apache/dubbo-go/tools/cli/protocol"
)

func init() {
	common.SetProtocol("dubbo", NewRpcClientPackageHandler)
}

// RpcClientPackageHandler handle package for client in getty.
type RpcClientPackageHandler struct {
}

func NewRpcClientPackageHandler() protocol.Protocol {
	return RpcClientPackageHandler{}
}

// Read decode @data to DubboPackage.
func (p RpcClientPackageHandler) Read(data []byte, pendingRsp *sync.Map) (interface{}, int, error) {
	pkg := &DubboPackage{}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf, pendingRsp)
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, nil
		}

		return nil, 0, perrors.WithStack(err)
	}

	if pkg.Header.Type&hessian.PackageRequest == 0x00 {
		pkg.Err = pkg.Body.(*hessian.Response).Exception
		pkg.Body = NewResponse(pkg.Body.(*hessian.Response).RspObj, pkg.Body.(*hessian.Response).Attachments)
	}
	dubboRsp, ok := pkg.Body.(*Response)
	if !ok {
		log.Println("error: dubbboRsp.Body assertion err:")
	}

	return dubboRsp.Reply, hessian.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

// Write encode @clientReq to []byte
func (p RpcClientPackageHandler) Write(clientReq *protocol.Request) ([]byte, error) {
	req, ok := parseReq2DubboPkg(clientReq)
	if !ok {
		log.Printf("illegal clientReq:%+v\n", clientReq)
		return nil, perrors.New("invalid rpc request")
	}

	buf, err := req.Marshal()
	if err != nil {
		log.Printf("binary.Write(req{%#v}) = err{%#v}\n", req, perrors.WithStack(err))
		return nil, perrors.WithStack(err)
	}

	return buf.Bytes(), nil
}

func parseReq2DubboPkg(req *protocol.Request) (*DubboPackage, bool) {
	p := &DubboPackage{}

	p.Service.Interface = req.InterfaceID
	p.Service.Version = req.Version
	p.Service.Group = req.Group
	p.Service.Method = req.Method
	p.Service.Timeout = time.Second * 3
	p.Header.SerialID = byte(S_Dubbo)
	p.Service.Path = req.InterfaceID

	atta := make(map[string]string)
	atta["async"] = "false"
	atta["interface"] = req.InterfaceID
	p.Body = hessian.NewRequest(req.Params, atta)
	p.Header.Type = hessian.PackageRequest_TwoWay
	p.Header.ID = int64(req.ID)

	return p, true
}

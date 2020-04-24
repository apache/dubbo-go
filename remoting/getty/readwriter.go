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

import (
	"bytes"
	"github.com/apache/dubbo-go/remoting"
	"reflect"
)

import (
	"github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
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
	//pkg := &DubboPackage{}

	buf := bytes.NewBuffer(data)
	resp, length, err := (*p.client.codec).DecodeResponse(buf)
	//err := pkg.Unmarshal(buf, p.client)
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, nil
		}

		logger.Errorf("pkg.Unmarshal(ss:%+v, len(@data):%d) = error:%+v", ss, len(data), err)

		return nil, length, perrors.WithStack(err)
	}
	//if pkg.Header.Type&hessian.PackageRequest == 0x00 {
	//	pkg.Err = pkg.Body.(*hessian.Response).Exception
	//	pkg.Body = NewResponse(pkg.Body.(*hessian.Response).RspObj, pkg.Body.(*hessian.Response).Attachments)
	//}

	return resp, length, nil
}

func (p *RpcClientPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	req, ok := pkg.(*remoting.Request)
	if !ok {
		logger.Errorf("illegal pkg:%+v\n", pkg)
		return nil, perrors.New("invalid rpc request")
	}

	buf, err := (*p.client.codec).EncodeRequest(req)
	if err != nil {
		logger.Warnf("binary.Write(req{%#v}) = err{%#v}", req, perrors.WithStack(err))
		return nil, perrors.WithStack(err)
	}

	return buf.Bytes(), nil
}

////////////////////////////////////////////
// RpcServerPackageHandler
////////////////////////////////////////////

var (
	rpcServerPkgHandler = &RpcServerPackageHandler{}
)

// RpcServerPackageHandler ...
type RpcServerPackageHandler struct {
	server *Server
}

func (p *RpcServerPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	buf := bytes.NewBuffer(data)
	req, length, err := (*p.server.codec).DecodeRequest(buf)
	//resp,len, err := (*p.).DecodeResponse(buf)

	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, nil
		}

		logger.Errorf("pkg.Unmarshal(ss:%+v, len(@data):%d) = error:%+v", ss, len(data), err)

		return nil, 0, perrors.WithStack(err)
	}

	return req, length, err
}

func (p *RpcServerPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	res, ok := pkg.(*remoting.Response)
	if !ok {
		logger.Errorf("illegal pkg:%+v\n, it is %+v", pkg, reflect.TypeOf(pkg))
		return nil, perrors.New("invalid rpc response")
	}

	buf, err := (*p.server.codec).EncodeResponse(res)
	if err != nil {
		logger.Warnf("binary.Write(res{%#v}) = err{%#v}", res, perrors.WithStack(err))
		return nil, perrors.WithStack(err)
	}

	return buf.Bytes(), nil
}

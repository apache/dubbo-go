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
	"reflect"
)

import (
	"github.com/apache/dubbo-getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol/dubbo/hessian2"
)

////////////////////////////////////////////
// RpcClientPackageHandler
////////////////////////////////////////////

// RpcClientPackageHandler handle package for client in getty.
type RpcClientPackageHandler struct {
	client *Client
}

// NewRpcClientPackageHandler create a RpcClientPackageHandler.
func NewRpcClientPackageHandler(client *Client) *RpcClientPackageHandler {
	return &RpcClientPackageHandler{client: client}
}

// Read decode @data to DubboPackage.
func (p *RpcClientPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := &DubboPackage{}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf, p.client)
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian2.ErrHeaderNotEnough || originErr == hessian2.ErrBodyNotEnough {
			return nil, 0, nil
		}

		logger.Errorf("pkg.Unmarshal(ss:%+v, len(@data):%d) = error:%+v", ss, len(data), err)

		return nil, 0, perrors.WithStack(err)
	}

	if pkg.Header.Type&hessian2.PackageRequest == 0x00 {
		pkg.Err = pkg.Body.(*hessian2.DubboResponse).Exception
		pkg.Body = NewResponse(pkg.Body.(*hessian2.DubboResponse).RspObj, pkg.Body.(*hessian2.DubboResponse).Attachments)
	}

	return pkg, hessian2.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

// Write encode @pkg.
func (p *RpcClientPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	req, ok := pkg.(*DubboPackage)
	if !ok {
		logger.Errorf("illegal pkg:%+v\n", pkg)
		return nil, perrors.New("invalid rpc request")
	}

	buf, err := req.Marshal()
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

// RpcServerPackageHandler handle package for server in getty.
type RpcServerPackageHandler struct{}

// Read decode @data to DubboPackage.
func (p *RpcServerPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := &DubboPackage{
		Body: make([]interface{}, 7),
	}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf)
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian2.ErrHeaderNotEnough || originErr == hessian2.ErrBodyNotEnough {
			return nil, 0, nil
		}

		logger.Errorf("pkg.Unmarshal(ss:%+v, len(@data):%d) = error:%+v", ss, len(data), err)

		return nil, 0, perrors.WithStack(err)
	}

	if pkg.Header.Type&hessian2.PackageHeartbeat == 0x00 {
		// convert params of request
		req := pkg.Body.([]interface{}) // length of body should be 7
		if len(req) > 0 {
			var dubboVersion, argsTypes string
			var args []interface{}
			var attachments map[string]interface{}
			if req[0] != nil {
				dubboVersion = req[0].(string)
			}
			if req[1] != nil {
				pkg.Service.Path = req[1].(string)
			}
			if req[2] != nil {
				pkg.Service.Version = req[2].(string)
			}
			if req[3] != nil {
				pkg.Service.Method = req[3].(string)
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
			if pkg.Service.Path == "" && attachments[constant.PATH_KEY] != nil && len(attachments[constant.PATH_KEY].(string)) > 0 {
				pkg.Service.Path = attachments[constant.PATH_KEY].(string)
			}
			if inter, ok := attachments[constant.INTERFACE_KEY]; ok && inter != nil {
				pkg.Service.Interface = inter.(string)
			} else {
				pkg.Service.Interface = pkg.Service.Path
			}
			if attachments[constant.GROUP_KEY] != nil && len(attachments[constant.GROUP_KEY].(string)) > 0 {
				pkg.Service.Group = attachments[constant.GROUP_KEY].(string)
			}
			pkg.Body = map[string]interface{}{
				"dubboVersion": dubboVersion,
				"argsTypes":    argsTypes,
				"args":         args,
				"service":      common.ServiceMap.GetService(DUBBO, pkg.Service.Path), // path as a key
				"attachments":  attachments,
			}
		}
	}

	return pkg, hessian2.HEADER_LENGTH + pkg.Header.BodyLen, nil
}

// Write encode @pkg.
func (p *RpcServerPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	res, ok := pkg.(*DubboPackage)
	if !ok {
		logger.Errorf("illegal pkg:%+v\n, it is %+v", pkg, reflect.TypeOf(pkg))
		return nil, perrors.New("invalid rpc response")
	}

	buf, err := res.Marshal()
	if err != nil {
		logger.Warnf("binary.Write(res{%#v}) = err{%#v}", res, perrors.WithStack(err))
		return nil, perrors.WithStack(err)
	}

	return buf.Bytes(), nil
}

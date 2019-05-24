// Copyright 2016-2019 Yincheng Fang, Alex Stocks
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dubbo

import (
	"bytes"
	"reflect"
)

import (
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
)

////////////////////////////////////////////
// RpcClientPackageHandler
////////////////////////////////////////////

type RpcClientPackageHandler struct {
	client *Client
}

func NewRpcClientPackageHandler(client *Client) *RpcClientPackageHandler {
	return &RpcClientPackageHandler{client: client}
}

func (p *RpcClientPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	p.client.pendingLock.RLock()
	defer p.client.pendingLock.RUnlock()
	pkg := &DubboPackage{}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf, p.client)
	if err != nil {
		pkg.Err = perrors.WithStack(err) // client will get this err
		return pkg, len(data), nil
	}

	return pkg, len(data), nil
}

func (p *RpcClientPackageHandler) Write(ss getty.Session, pkg interface{}) error {
	req, ok := pkg.(*DubboPackage)
	if !ok {
		logger.Errorf("illegal pkg:%+v\n", pkg)
		return perrors.New("invalid rpc request")
	}

	buf, err := req.Marshal()
	if err != nil {
		logger.Warnf("binary.Write(req{%#v}) = err{%#v}", req, perrors.WithStack(err))
		return perrors.WithStack(err)
	}

	return perrors.WithStack(ss.WriteBytes(buf.Bytes()))
}

////////////////////////////////////////////
// RpcServerPackageHandler
////////////////////////////////////////////

type RpcServerPackageHandler struct {
}

func NewRpcServerPackageHandler() *RpcServerPackageHandler {
	return &RpcServerPackageHandler{}
}

func (p *RpcServerPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := &DubboPackage{
		Body: make([]interface{}, 7),
	}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf)
	if err != nil {
		return nil, 0, perrors.WithStack(err)
	}
	// convert params of request
	req := pkg.Body.([]interface{}) // length of body should be 7
	if len(req) > 0 {
		var dubboVersion, argsTypes string
		var args []interface{}
		var attachments map[interface{}]interface{}
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
			attachments = req[6].(map[interface{}]interface{})
		}
		pkg.Service.Interface = attachments[constant.INTERFACE_KEY].(string)
		pkg.Body = map[string]interface{}{
			"dubboVersion": dubboVersion,
			"argsTypes":    argsTypes,
			"args":         args,
			"service":      common.ServiceMap.GetService(DUBBO, pkg.Service.Interface),
			"attachments":  attachments,
		}
	}

	return pkg, len(data), nil
}

func (p *RpcServerPackageHandler) Write(ss getty.Session, pkg interface{}) error {
	res, ok := pkg.(*DubboPackage)
	if !ok {
		logger.Errorf("illegal pkg:%+v\n, it is %+v", pkg, reflect.TypeOf(pkg))
		return perrors.New("invalid rpc response")
	}

	buf, err := res.Marshal()
	if err != nil {
		logger.Warnf("binary.Write(res{%#v}) = err{%#v}", res, perrors.WithStack(err))
		return perrors.WithStack(err)
	}

	return perrors.WithStack(ss.WriteBytes(buf.Bytes()))
}

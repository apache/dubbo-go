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
	"bufio"
	"bytes"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"reflect"
)

import (
	"github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
)

////////////////////////////////////////////
// RpcServerPackageHandler
////////////////////////////////////////////

var (
	rpcServerPkgHandler = &RpcServerPackageHandler{}
)

// RpcServerPackageHandler ...
type RpcServerPackageHandler struct {
}

func (p *RpcServerPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := NewServerRequestPackage(data)
	if err := pkg.ReadHeader(); err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, nil
		}
		return nil, 0, perrors.WithStack(err)
	}

	if pkg.IsHeartBeat() {
		return pkg, pkg.GetLen(), nil
	}

	if err := loadSerializer(pkg); err != nil {
		return nil, 0, err
	}

	err := pkg.Unmarshal()
	if err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, nil
		}

		logger.Errorf("pkg.Unmarshal(ss:%+v, len(@data):%d) = error:%+v", ss, len(data), err)
		return nil, 0, perrors.WithStack(err)
	}
	return pkg, pkg.GetLen(), nil
}

func (p *RpcServerPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	res, ok := pkg.(*impl.DubboPackage)
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

func loadSerializer(p *impl.DubboPackage) error {
	// NOTE: default serialID is S_Hessian
	serialID := p.Header.SerialID
	if serialID == 0 {
		serialID = constant.S_Hessian2
	}
	serializer, err := extension.GetSerializerById(serialID)
	if err != nil {
		return err
	}
	p.SetSerializer(serializer.(impl.Serializer))
	return nil
}


// server side receive request package, just for deserialization
func NewServerRequestPackage(data []byte) *impl.DubboPackage {
	return &impl.DubboPackage{
		Header:  impl.DubboHeader{},
		Service: impl.Service{},
		Body:    make([]interface{}, 7),
		Err:     nil,
		Codec:   impl.NewDubboCodec(bufio.NewReaderSize(bytes.NewBuffer(data), len(data))),
	}

}
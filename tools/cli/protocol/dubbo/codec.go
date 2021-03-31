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
	e1 "errors"
	"fmt"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

// SerialID serial ID
type SerialID byte

const (
	// S_Dubbo protocol serial id
	S_Dubbo SerialID = 2
)

// CallType call type
type CallType int32

const (
	// CT_UNKNOWN unknown call type
	CT_UNKNOWN CallType = 0
	// CT_OneWay call one way
	CT_OneWay CallType = 1
	// CT_TwoWay call in request/response
	CT_TwoWay CallType = 2
)

////////////////////////////////////////////
// protocol package
////////////////////////////////////////////

// SequenceType sequence type
type SequenceType int64

// nolint
type DubboPackage struct {
	Header  hessian.DubboHeader
	Service hessian.Service
	Body    interface{}
	Err     error
}

// Marshal encode hessian package.
// DubboPackage -> byte
func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	codec := hessian.NewHessianCodec(nil)
	pkg, err := codec.Write(p.Service, p.Header, p.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	return bytes.NewBuffer(pkg), nil
}

// Unmarshal decodes hessian package.
// byte -> DubboPackage
func (p *DubboPackage) Unmarshal(buf *bytes.Buffer, pendingRsp *sync.Map) error {
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
	if p.Header.Type&hessian.PackageRequest != 0x00 {
		p.Body = make([]interface{}, 7)
	} else {
		rspObj, ok := pendingRsp.Load(uint64(p.Header.ID))
		if !ok {
			return e1.New(fmt.Sprintf("seq = %d  not found", p.Header.ID))
		}
		p.Body = &hessian.Response{RspObj: rspObj}
	}

	// read body
	err = codec.ReadBody(p.Body)
	return perrors.WithStack(err)
}

////////////////////////////////////////////
// Response
////////////////////////////////////////////
// Response is protocol protocol response.
type Response struct {
	Reply interface{}
	atta  map[string]string
}

// NewResponse creates a new Response.
func NewResponse(reply interface{}, atta map[string]string) *Response {
	return &Response{
		Reply: reply,
		atta:  atta,
	}
}

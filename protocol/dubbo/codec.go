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
	"fmt"
	"time"
)

import (
	"github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

// serial ID
type SerialID byte

const (
	S_Dubbo SerialID = 2
)

// call type
type CallType int32

const (
	CT_UNKNOWN CallType = 0
	CT_OneWay  CallType = 1
	CT_TwoWay  CallType = 2
)

////////////////////////////////////////////
// dubbo package
////////////////////////////////////////////

type SequenceType int64

type DubboPackage struct {
	Header  hessian.DubboHeader
	Service hessian.Service
	Body    interface{}
	Err     error
}

func (p DubboPackage) String() string {
	return fmt.Sprintf("DubboPackage: Header-%v, Path-%v, Body-%v", p.Header, p.Service, p.Body)
}

func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	codec := hessian.NewHessianCodec(nil)

	pkg, err := codec.Write(p.Service, p.Header, p.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	return bytes.NewBuffer(pkg), nil
}

func (p *DubboPackage) Unmarshal(buf *bytes.Buffer, opts ...interface{}) error {
	codec := hessian.NewHessianCodec(bufio.NewReaderSize(buf, buf.Len()))

	// read header
	err := codec.ReadHeader(&p.Header)
	if err != nil {
		return perrors.WithStack(err)
	}

	if len(opts) != 0 { // for client
		client, ok := opts[0].(*Client)
		if !ok {
			return perrors.Errorf("opts[0] is not of type *Client")
		}

		pendingRsp, ok := client.pendingResponses.Load(SequenceType(p.Header.ID))
		if !ok {
			return perrors.Errorf("client.GetPendingResponse(%v) = nil", p.Header.ID)
		}
		p.Body = &hessian.Response{RspObj: pendingRsp.(*PendingResponse).response.reply}
	}

	// read body
	err = codec.ReadBody(p.Body)
	return perrors.WithStack(err)
}

////////////////////////////////////////////
// PendingResponse
////////////////////////////////////////////

type PendingResponse struct {
	seq       uint64
	err       error
	start     time.Time
	readStart time.Time
	callback  AsyncCallback
	response  *Response
	done      chan struct{}
}

func NewPendingResponse() *PendingResponse {
	return &PendingResponse{
		start:    time.Now(),
		response: &Response{},
		done:     make(chan struct{}),
	}
}

func (r PendingResponse) GetCallResponse() CallResponse {
	return CallResponse{
		Cause:     r.err,
		Start:     r.start,
		ReadStart: r.readStart,
		Reply:     r.response,
	}
}

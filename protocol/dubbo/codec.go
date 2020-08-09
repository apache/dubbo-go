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
	"github.com/apache/dubbo-go/common"
	perrors "github.com/pkg/errors"
)

//SerialID serial ID
type SerialID byte

const (
	// S_Dubbo dubbo serial id
	S_Dubbo SerialID = 2
)

//CallType call type
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
// dubbo package
////////////////////////////////////////////

// SequenceType ...
type SequenceType int64

// DubboPackage ...
type DubboPackage struct {
	Header  hessian.DubboHeader
	Service hessian.Service
	Body    interface{}
	Err     error
}

// String prints dubbo package detail include header、path、body etc.
func (p DubboPackage) String() string {
	return fmt.Sprintf("DubboPackage: Header-%v, Path-%v, Body-%v", p.Header, p.Service, p.Body)
}

// Marshal encode hessian package.
func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	codec := hessian.NewHessianCodec(nil)

	pkg, err := codec.Write(p.Service, p.Header, p.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	return bytes.NewBuffer(pkg), nil
}

// Unmarshal decode hessian package.
func (p *DubboPackage) Unmarshal(buf *bytes.Buffer, opts ...interface{}) error {
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

	if len(opts) != 0 { // for client
		client, ok := opts[0].(*Client)
		if !ok {
			return perrors.Errorf("opts[0] is not of type *Client")
		}

		if p.Header.Type&hessian.PackageRequest != 0x00 {
			// size of this array must be '7'
			// https://github.com/apache/dubbo-go-hessian2/blob/master/request.go#L272
			p.Body = make([]interface{}, 7)
		} else {
			pendingRsp, ok := client.pendingResponses.Load(SequenceType(p.Header.ID))
			if !ok {
				return perrors.Errorf("client.GetPendingResponse(%v) = nil", p.Header.ID)
			}
			p.Body = &hessian.Response{RspObj: pendingRsp.(*PendingResponse).response.reply}
		}
	}

	// read body
	err = codec.ReadBody(p.Body)
	return perrors.WithStack(err)
}

////////////////////////////////////////////
// PendingResponse
////////////////////////////////////////////

// PendingResponse is a pending response.
type PendingResponse struct {
	seq       uint64
	err       error
	start     time.Time
	readStart time.Time
	callback  common.AsyncCallback
	response  *Response
	done      chan struct{}
}

// NewPendingResponse create a PendingResponses.
func NewPendingResponse() *PendingResponse {
	return &PendingResponse{
		start:    time.Now(),
		response: &Response{},
		done:     make(chan struct{}),
	}
}

// GetCallResponse get AsyncCallbackResponse.
func (r PendingResponse) GetCallResponse() common.CallbackResponse {
	return AsyncCallbackResponse{
		Cause:     r.err,
		Start:     r.start,
		ReadStart: r.readStart,
		Reply:     r.response,
	}
}

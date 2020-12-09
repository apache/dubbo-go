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
	"sync"
	"time"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

var (
	// generate request ID for global use
	sequence atomic.Int64

	// store requestID and response
	pendingResponses = new(sync.Map)
)

type SequenceType int64

func init() {
	// init request ID
	sequence.Store(0)
}

func SequenceId() int64 {
	// increse 2 for every request as the same before.
	// We expect that the request from client to server, the requestId is even; but from server to client, the requestId is odd.
	return sequence.Add(2)
}

// this is request for transport layer
type Request struct {
	ID int64
	// protocol version
	Version string
	// serial ID (ignore)
	SerialID byte
	// Data
	Data   interface{}
	TwoWay bool
	Event  bool
}

// NewRequest aims to create Request.
// The ID is auto increase.
func NewRequest(version string) *Request {
	return &Request{
		ID:      SequenceId(),
		Version: version,
	}
}

// this is response for transport layer
type Response struct {
	ID       int64
	Version  string
	SerialID byte
	Status   uint8
	Event    bool
	Error    error
	Result   interface{}
}

// NewResponse create to a new Response.
func NewResponse(id int64, version string) *Response {
	return &Response{
		ID:      id,
		Version: version,
	}
}

// the response is heartbeat
func (response *Response) IsHeartbeat() bool {
	return response.Event && response.Result == nil
}

func (response *Response) Handle() {
	pendingResponse := removePendingResponse(SequenceType(response.ID))
	if pendingResponse == nil {
		logger.Errorf("failed to get pending response context for response package %s", *response)
		return
	}

	pendingResponse.response = response

	if pendingResponse.Callback == nil {
		pendingResponse.Err = pendingResponse.response.Error
		close(pendingResponse.Done)
	} else {
		pendingResponse.Callback(pendingResponse.GetCallResponse())
	}
}

type Options struct {
	// connect timeout
	ConnectTimeout time.Duration
}

//AsyncCallbackResponse async response for dubbo
type AsyncCallbackResponse struct {
	common.CallbackResponse
	Opts      Options
	Cause     error
	Start     time.Time // invoke(call) start time == write start time
	ReadStart time.Time // read start time, write duration = ReadStart - Start
	Reply     interface{}
}

// the client sends requst to server, there is one pendingResponse at client side to wait the response from server
type PendingResponse struct {
	seq       int64
	Err       error
	start     time.Time
	ReadStart time.Time
	Callback  common.AsyncCallback
	response  *Response
	Reply     interface{}
	Done      chan struct{}
}

// NewPendingResponse aims to create PendingResponse.
// Id is always from ID of Request
func NewPendingResponse(id int64) *PendingResponse {
	return &PendingResponse{
		seq:      id,
		start:    time.Now(),
		response: &Response{},
		Done:     make(chan struct{}),
	}
}

func (r *PendingResponse) SetResponse(response *Response) {
	r.response = response
}

// GetCallResponse is used for callback of async.
// It is will return AsyncCallbackResponse.
func (r PendingResponse) GetCallResponse() common.CallbackResponse {
	return AsyncCallbackResponse{
		Cause:     r.Err,
		Start:     r.start,
		ReadStart: r.ReadStart,
		Reply:     r.response,
	}
}

// store response into map
func AddPendingResponse(pr *PendingResponse) {
	pendingResponses.Store(SequenceType(pr.seq), pr)
}

// get and remove response
func removePendingResponse(seq SequenceType) *PendingResponse {
	if pendingResponses == nil {
		return nil
	}
	if presp, ok := pendingResponses.Load(seq); ok {
		pendingResponses.Delete(seq)
		return presp.(*PendingResponse)
	}
	return nil
}

// get response
func GetPendingResponse(seq SequenceType) *PendingResponse {
	if presp, ok := pendingResponses.Load(seq); ok {
		return presp.(*PendingResponse)
	}
	return nil
}

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
	"time"

	"github.com/apache/dubbo-go/common"
	"go.uber.org/atomic"
)

var (
	// generate request ID for global use
	sequence atomic.Uint64
)

func init() {
	// init request ID
	sequence.Store(0)
}

func SequenceId() uint64 {
	// increse 2 for every request.
	return sequence.Add(2)
}

// Request ...
type Request struct {
	ID int64
	// protocol version
	Version string
	// serial ID
	SerialID byte
	// Data
	Data   interface{}
	TwoWay bool
	Event  bool
	// it is used to judge the request is unbroken
	// broken bool
}

// NewRequest
func NewRequest(version string) *Request {
	return &Request{
		ID:      int64(SequenceId()),
		Version: version,
	}
}

// Response ...
type Response struct {
	ID       int64
	Version  string
	SerialID byte
	Status   uint8
	Event    bool
	Error    error
	Result   interface{}
}

// NewResponse
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

// NewPendingResponse ...
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

// GetCallResponse ...
func (r PendingResponse) GetCallResponse() common.CallbackResponse {
	return AsyncCallbackResponse{
		Cause:     r.Err,
		Start:     r.start,
		ReadStart: r.ReadStart,
		Reply:     r.response,
	}
}

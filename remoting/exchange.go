package remoting

import (
	"github.com/apache/dubbo-go/common"
	"time"
)

//// Request ...
//type Request struct {
//	addr   string
//	svcUrl common.URL
//	method string
//	args   interface{}
//	atta   map[string]string
//}
//
//// NewRequest ...
//func NewRequest(addr string, svcUrl common.URL, method string, args interface{}, atta map[string]string) *Request {
//	return &Request{
//		addr:   addr,
//		svcUrl: svcUrl,
//		method: method,
//		args:   args,
//		atta:   atta,
//	}
//}
//
//// Response ...
//type Response struct {
//	reply interface{}
//	atta  map[string]string
//}
//
//// NewResponse ...
//func NewResponse(reply interface{}, atta map[string]string) *Response {
//	return &Response{
//		reply: reply,
//		atta:  atta,
//	}
//}
// Request ...
type Request struct {
	Id       int64
	Version  string
	SerialID byte
	Data     interface{}
	TwoWay   bool
	event    bool
	broken   bool
}

// NewRequest ...
func NewRequest(id int64, version string) *Request {
	return &Request{
		Id:      id,
		Version: version,
	}
}

func (request *Request) SetHeartbeat(isHeartbeat bool) {
	if isHeartbeat {

	}
}

// Response ...
type Response struct {
	Id       int64
	Version  string
	SerialID byte
	Status   uint8
	Event    bool
	Error    error
	Result   interface{}
	Reply    interface{}
}

// NewResponse ...
func NewResponse(id int64, version string) *Response {
	return &Response{
		Id:      id,
		Version: version,
	}
}

func (response *Response) IsHeartbeat() bool {
	return response.Event && response.Result == nil
}

type Options struct {
	// connect timeout
	ConnectTimeout time.Duration
	// request timeout
	//RequestTimeout time.Duration
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

type PendingResponse struct {
	seq       uint64
	err       error
	start     time.Time
	readStart time.Time
	callback  common.AsyncCallback
	response  *Response
	done      chan struct{}
}

// NewPendingResponse ...
func NewPendingResponse() *PendingResponse {
	return &PendingResponse{
		start:    time.Now(),
		response: &Response{},
		done:     make(chan struct{}),
	}
}

// GetCallResponse ...
func (r PendingResponse) GetCallResponse() common.CallbackResponse {
	return AsyncCallbackResponse{
		Cause:     r.err,
		Start:     r.start,
		ReadStart: r.readStart,
		Reply:     r.response,
	}
}

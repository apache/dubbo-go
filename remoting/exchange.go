package remoting

import (
	"github.com/apache/dubbo-go/common"
	"go.uber.org/atomic"
	"time"
)

var (
	sequence atomic.Uint64
)

func init() {
	sequence.Store(0)
}

func SequenceId() uint64 {
	return sequence.Add(2)
}

// Request ...
type Request struct {
	Id       int64
	Version  string
	SerialID byte
	Data     interface{}
	TwoWay   bool
	Event    bool
	broken   bool
}

// NewRequest ...
func NewRequest(version string) *Request {
	return &Request{
		Id:      int64(SequenceId()),
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
	seq       int64
	Err       error
	start     time.Time
	ReadStart time.Time
	callback  common.AsyncCallback
	response  *Response
	Done      chan struct{}
}

// NewPendingResponse ...
func NewPendingResponse() *PendingResponse {
	return &PendingResponse{
		start:    time.Now(),
		response: &Response{},
		Done:     make(chan struct{}),
	}
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

type Client interface {
	//invoke once for connection
	Connect(url common.URL)
	Close()
	Request(request *Request, timeout time.Duration, callback common.AsyncCallback, response *PendingResponse) error
}

type Server interface {
	//invoke once for connection
	Open(url common.URL)
}

type ResponseHandler interface {
	Handler(response *Response)
}


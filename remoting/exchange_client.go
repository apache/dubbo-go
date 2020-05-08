package remoting

import (
	"sync"
	"time"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

var (
	pendingResponses *sync.Map = new(sync.Map)
)

type SequenceType int64

type ExchangeClient struct {
	ConnectTimeout time.Duration
	address        string
	client         Client
}

type Client interface {
	SetExchangeClient(client *ExchangeClient)
	SetResponseHandler(responseHandler ResponseHandler)
	//invoke once for connection
	//ConfigClient()
	Connect(url common.URL)
	Close()
	Request(request *Request, timeout time.Duration, response *PendingResponse) error
}

type ResponseHandler interface {
	Handler(response *Response)
}

func NewExchangeClient(url common.URL, client Client, connectTimeout time.Duration) *ExchangeClient {
	exchangeClient := &ExchangeClient{
		ConnectTimeout: connectTimeout,
		address:        url.Location,
		client:         client,
	}
	client.SetExchangeClient(exchangeClient)
	client.Connect(url)
	client.SetResponseHandler(exchangeClient)
	return exchangeClient
}

func (client *ExchangeClient) Request(invocation *protocol.Invocation, url common.URL, timeout time.Duration,
	result *protocol.RPCResult) error {
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = true

	rsp := NewPendingResponse(request.Id)
	rsp.response = NewResponse(request.Id, "2.0.2")
	rsp.Reply = (*invocation).Reply()
	AddPendingResponse(rsp)
	//rsp.callback = invo

	err := client.client.Request(request, timeout, rsp)
	if err != nil {
		result.Err = err
		return err
	}
	result.Rest = rsp.response.Result
	//result.Attrs = rsp.response.
	return nil
}

func (client *ExchangeClient) AsyncRequest(invocation *protocol.Invocation, url common.URL, timeout time.Duration,
	callback common.AsyncCallback, result *protocol.RPCResult) error {
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = true

	rsp := NewPendingResponse(request.Id)
	rsp.response = NewResponse(request.Id, "2.0.2")
	rsp.Callback = callback
	rsp.Reply = (*invocation).Reply()
	AddPendingResponse(rsp)

	err := client.client.Request(request, timeout, rsp)
	if err != nil {
		result.Err = err
		return err
	}
	result.Rest = rsp.response
	//result.Attrs = rsp.response.
	return nil
}

// oneway
func (client *ExchangeClient) Send(invocation *protocol.Invocation, timeout time.Duration) error {
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = false

	rsp := NewPendingResponse(request.Id)
	rsp.response = NewResponse(request.Id, "2.0.2")

	err := client.client.Request(request, timeout, rsp)
	if err != nil {
		return err
	}
	//result.Attrs = rsp.response.
	return nil
}

func (client *ExchangeClient) Close() {
	client.client.Close()
}

func (client *ExchangeClient) Handler(response *Response) {

	pendingResponse := removePendingResponse(SequenceType(response.Id))
	if pendingResponse == nil {
		logger.Errorf("failed to get pending response context for response package %s", *response)
		return
	}

	pendingResponse.response = response

	if pendingResponse.Callback == nil {
		pendingResponse.Err = pendingResponse.response.Error
		pendingResponse.Done <- struct{}{}
	} else {
		pendingResponse.Callback(pendingResponse.GetCallResponse())
	}
}

func AddPendingResponse(pr *PendingResponse) {
	pendingResponses.Store(SequenceType(pr.seq), pr)
}

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

func GetPendingResponse(seq SequenceType) *PendingResponse {
	if presp, ok := pendingResponses.Load(seq); ok {
		return presp.(*PendingResponse)
	}
	return nil
}

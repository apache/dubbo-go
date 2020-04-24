package remoting

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"sync"
	"time"
)

type SequenceType int64

type ExchangeClient struct {
	ConnectTimeout   time.Duration
	address          string
	client           Client
	pendingResponses *sync.Map
}

func NewExchangeClient(url common.URL, client Client, connectTimeout time.Duration) *ExchangeClient {
	exchangeClient := &ExchangeClient{
		ConnectTimeout: connectTimeout,
		address:        url.Location,
		client:         client,
	}
	client.Connect(url)
	return exchangeClient
}

func (client *ExchangeClient) Request(invocation *protocol.Invocation, url common.URL, timeout time.Duration,
	result *protocol.RPCResult) error {
	requestId := int64(SequenceId())
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = true

	rsp := NewPendingResponse()
	rsp.response = NewResponse(requestId, "2.0.2")
	//rsp.callback = invo

	err := client.client.Request(request, timeout, nil, rsp)
	if err != nil {
		result.Err = err
		return err
	}
	result.Rest = rsp.response
	//result.Attrs = rsp.response.
	return nil
}

func (client *ExchangeClient) AsyncRequest(invocation *protocol.Invocation, url common.URL, timeout time.Duration,
	callback common.AsyncCallback, result *protocol.RPCResult) error {
	requestId := int64(SequenceId())
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = true

	rsp := NewPendingResponse()
	rsp.response = NewResponse(requestId, "2.0.2")
	rsp.callback = callback

	err := client.client.Request(request, timeout, nil, rsp)
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
	requestId := int64(SequenceId())
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = false

	rsp := NewPendingResponse()
	rsp.response = NewResponse(requestId, "2.0.2")

	err := client.client.Request(request, timeout, nil, rsp)
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

	pendingResponse := client.removePendingResponse(SequenceType(response.Id))
	if pendingResponse == nil {
		logger.Errorf("failed to get pending response context for response package %s", *response)
		return
	}

	pendingResponse.response = response

	if pendingResponse.callback == nil {
		pendingResponse.Done <- struct{}{}
	} else {
		pendingResponse.callback(pendingResponse.GetCallResponse())
	}
}

func (client *ExchangeClient) addPendingResponse(pr *PendingResponse) {
	client.pendingResponses.Store(SequenceType(pr.seq), pr)
}

func (client *ExchangeClient) removePendingResponse(seq SequenceType) *PendingResponse {
	if client.pendingResponses == nil {
		return nil
	}
	if presp, ok := client.pendingResponses.Load(seq); ok {
		client.pendingResponses.Delete(seq)
		return presp.(*PendingResponse)
	}
	return nil
}

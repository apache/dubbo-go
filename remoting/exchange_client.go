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
	Connect(url common.URL) error
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
	if client.Connect(url) != nil {
		//retry for a while
		time.Sleep(1 * time.Second)
		if client.Connect(url) != nil {
			return nil
		}
	}
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

	err := client.client.Request(request, timeout, rsp)
	if err != nil {
		result.Err = err
		return err
	}
	result.Rest = rsp.response.Result
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

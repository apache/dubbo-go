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
	"errors"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	uatomic "go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// Client is the interface that wraps SetExchangeClient、 Connect、Close、Request and
// IsAvailable method. It is interface of client for network communication. If you
// use getty as network communication, you should define GettyClient that implements
// this interface.
//
// SetExchangeClient method sets a ExchangeClient instance.
//
// Connect method is to connect url.
//
// Close method is for destroy.
//
// Request method is sending request to server.
//
// IsAvailable method checks whether the client is still available.
type Client interface {
	SetExchangeClient(client *ExchangeClient)
	Connect(url *common.URL) error
	Close()
	Request(request *Request, timeout time.Duration, response *PendingResponse) error
	IsAvailable() bool
}

// ExchangeClient is abstraction level. it is like facade.
type ExchangeClient struct {
	ConnectTimeout time.Duration  // timeout for connecting server
	address        string         // server address for dialing. The format: ip:port
	client         Client         // dealing with the transport
	init           bool           // the tag for init.
	activeNum      uatomic.Uint32 // the number of service using the exchangeClient
}

// NewExchangeClient returns a ExchangeClient.
func NewExchangeClient(url *common.URL, client Client, connectTimeout time.Duration, lazyInit bool) *ExchangeClient {
	exchangeClient := &ExchangeClient{
		ConnectTimeout: connectTimeout,
		address:        url.Location,
		client:         client,
	}
	if !lazyInit {
		if err := exchangeClient.doInit(url); err != nil {
			return nil
		}
	}
	exchangeClient.IncreaseActiveNumber()
	return exchangeClient
}

func (cl *ExchangeClient) doInit(url *common.URL) error {
	if cl.init {
		return nil
	}
	if cl.client.Connect(url) != nil {
		// retry for a while
		time.Sleep(100 * time.Millisecond)
		if cl.client.Connect(url) != nil {
			logger.Errorf("Failed to connect server %+v " + url.Location)
			return errors.New("Failed to connect server " + url.Location)
		}
	}
	// FIXME atomic operation
	cl.init = true
	return nil
}

// IncreaseActiveNumber increase number of service using client.
func (client *ExchangeClient) IncreaseActiveNumber() uint32 {
	return client.activeNum.Add(1)
}

// DecreaseActiveNumber decrease number of service using client.
func (client *ExchangeClient) DecreaseActiveNumber() uint32 {
	return client.activeNum.Sub(1)
}

// GetActiveNumber get number of service using client.
func (client *ExchangeClient) GetActiveNumber() uint32 {
	return client.activeNum.Load()
}

// Request means two way request.
func (client *ExchangeClient) Request(invocation *protocol.Invocation, url *common.URL, timeout time.Duration,
	result *protocol.RPCResult) error {
	if er := client.doInit(url); er != nil {
		return er
	}
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = true

	rsp := NewPendingResponse(request.ID)
	rsp.response = NewResponse(request.ID, "2.0.2")
	rsp.Reply = (*invocation).Reply()
	AddPendingResponse(rsp)

	err := client.client.Request(request, timeout, rsp)
	// request error
	if err != nil {
		result.Err = err
		return err
	}
	if resultTmp, ok := rsp.response.Result.(*protocol.RPCResult); ok {
		result.Rest = resultTmp.Rest
		result.Attrs = resultTmp.Attrs
		result.Err = resultTmp.Err
	} else {
		logger.Warnf("[ExchangeClient.Request] The type of result is unexpected, we want *protocol.RPCResult, "+
			"but we got %T", rsp.response.Result)
	}
	return nil
}

// AsyncRequest async two way request.
func (client *ExchangeClient) AsyncRequest(invocation *protocol.Invocation, url *common.URL, timeout time.Duration,
	callback common.AsyncCallback, result *protocol.RPCResult) error {
	if er := client.doInit(url); er != nil {
		return er
	}
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = true

	rsp := NewPendingResponse(request.ID)
	rsp.response = NewResponse(request.ID, "2.0.2")
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

// Send sends oneway request.
func (client *ExchangeClient) Send(invocation *protocol.Invocation, url *common.URL, timeout time.Duration) error {
	if er := client.doInit(url); er != nil {
		return er
	}
	request := NewRequest("2.0.2")
	request.Data = invocation
	request.Event = false
	request.TwoWay = false

	rsp := NewPendingResponse(request.ID)
	rsp.response = NewResponse(request.ID, "2.0.2")

	err := client.client.Request(request, timeout, rsp)
	if err != nil {
		return err
	}
	return nil
}

// Close close the client.
func (client *ExchangeClient) Close() {
	client.client.Close()
	client.init = false
}

// IsAvailable to check if the underlying network client is available yet.
func (client *ExchangeClient) IsAvailable() bool {
	return client.client.IsAvailable()
}

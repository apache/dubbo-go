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

package client

import (
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/tools/cli/common"
	"github.com/apache/dubbo-go/tools/cli/protocol"
	_ "github.com/apache/dubbo-go/tools/cli/protocol/dubbo"
)

// defaultBufferSize is the tcp read default buffer size
const defaultBufferSize = 1024 * 1024 * 4

// TelnetClient maintain a connection to target
type TelnetClient struct {
	tcpAddr         string
	responseTimeout time.Duration
	protocolName    string
	requestList     []*protocol.Request
	conn            *net.TCPConn
	proto           protocol.Protocol

	sequence         atomic.Uint64
	pendingResponses *sync.Map
	waitNum          atomic.Uint64
}

// NewTelnetClient create a new tcp connection, and create default request
func NewTelnetClient(host string, port int, protocolName, interfaceID, version, group, method string, reqPkg interface{}, timeout int) (*TelnetClient, error) {
	tcpAddr := net.JoinHostPort(host, strconv.Itoa(port))
	resolved := resolveTCPAddr(tcpAddr)
	conn, err := net.DialTCP("tcp", nil, resolved)
	if err != nil {
		return nil, err
	}
	log.Printf("connected to %s:%d\n", host, port)
	log.Printf("try calling interface:%s.%s\n", interfaceID, method)
	log.Printf("with protocol:%s\n\n", protocolName)
	proto := common.GetProtocol(protocolName)

	return &TelnetClient{
		tcpAddr:          tcpAddr,
		conn:             conn,
		responseTimeout:  time.Duration(timeout) * time.Millisecond, //default timeout
		protocolName:     protocolName,
		pendingResponses: &sync.Map{},
		proto:            proto,
		requestList: []*protocol.Request{
			{
				InterfaceID: interfaceID,
				Version:     version,
				Method:      method,
				Group:       group,
				Params:      []interface{}{reqPkg},
			},
		},
	}, nil
}

func resolveTCPAddr(addr string) *net.TCPAddr {
	resolved, error := net.ResolveTCPAddr("tcp", addr)
	if nil != error {
		log.Fatalf("Error occured while resolving TCP address \"%v\": %v\n", addr, error)
	}

	return resolved
}

// ProcessRequests send all requests
func (t *TelnetClient) ProcessRequests(userPkg interface{}) {
	for i, _ := range t.requestList {
		t.processSingleRequest(t.requestList[i], userPkg)
	}
}

// addPendingResponse add a response @model to pending queue
// once the rsp got, the model will be used.
func (t *TelnetClient) addPendingResponse(model interface{}) uint64 {
	seqId := t.sequence.Load()
	t.pendingResponses.Store(seqId, model)
	t.waitNum.Inc()
	t.sequence.Inc()
	return seqId
}

// removePendingResponse delete item from pending queue by @seq
func (t *TelnetClient) removePendingResponse(seq uint64) {
	if t.pendingResponses == nil {
		return
	}
	if _, ok := t.pendingResponses.Load(seq); ok {
		t.pendingResponses.Delete(seq)
		t.waitNum.Dec()
	}
	return
}

// processSingleRequest call one req.
func (t *TelnetClient) processSingleRequest(req *protocol.Request, userPkg interface{}) {
	// proto create package procedure
	req.ID = t.sequence.Load()
	inputData, err := t.proto.Write(req)
	if err != nil {
		log.Fatalln("error: handler.Writer err = ", err)
	}
	startTime := time.Now()

	// init rsp Package and add to pending queue
	seqId := t.addPendingResponse(userPkg)
	defer t.removePendingResponse(seqId)

	requestDataChannel := make(chan []byte, 0)
	responseDataChannel := make(chan []byte, 0)

	// start data transfer procedure
	go t.readInputData(string(inputData), requestDataChannel)
	go t.readServerData(t.conn, responseDataChannel)

	timeAfter := time.After(t.responseTimeout)

	for {
		select {
		case <-timeAfter:
			log.Println("request timeout to:", t.tcpAddr)
			return
		case request := <-requestDataChannel:
			if _, err := t.conn.Write(request); nil != err {
				log.Fatalf("Error occured while writing to TCP socket: %v\n", err)
			}
		case response := <-responseDataChannel:
			rspPkg, _, err := t.proto.Read(response, t.pendingResponses)
			if err != nil {
				log.Fatalln("Error with protocol Read(): ", err)
			}
			t.removePendingResponse(seqId)
			log.Printf("After %dms , Got Rsp:", time.Now().Sub(startTime).Milliseconds())
			common.PrintInterface(rspPkg)
			if t.waitNum.Sub(0) == 0 {
				return
			}
		}
	}
}

func (t *TelnetClient) readInputData(inputData string, toSent chan<- []byte) {
	toSent <- []byte(inputData)
}

func (t *TelnetClient) readServerData(connection *net.TCPConn, received chan<- []byte) {
	buffer := make([]byte, defaultBufferSize)
	var err error
	var n int
	for nil == err {
		n, err = connection.Read(buffer)
		received <- buffer[:n]
	}

	t.assertEOF(err)
}

func (t *TelnetClient) assertEOF(error error) {
	if "EOF" != error.Error() {
		log.Fatalf("Error occured while operating on TCP socket: %v\n", error)
	}
}

// Destroy close the tcp conn
func (t *TelnetClient) Destroy() {
	t.conn.Close()
}

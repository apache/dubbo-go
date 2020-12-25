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

package dubbo3

import (
	"net"
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

// TripleServer is the object that can be started and listening remote request
type TripleServer struct {
	lst        net.Listener
	addr       string
	rpcService common.RPCService
	url        *common.URL
}

// NewTripleServer can create Server with user impled @service and url
func NewTripleServer(url *common.URL, service common.RPCService) *TripleServer {
	return &TripleServer{
		addr:       url.Location,
		rpcService: service,
		url:        url,
	}
}

// Stop
func (t *TripleServer) Stop() {

}

// Start can start a triple server
func (t *TripleServer) Start() {
	logger.Info("tripleServer Start at ", t.addr)
	lst, err := net.Listen("tcp", t.addr)
	if err != nil {
		panic(err)
	}
	t.lst = lst
	t.run()
}

// run can start a loop to accept tcp conn
func (t *TripleServer) run() {
	wg := sync.WaitGroup{}
	for {
		conn, err := t.lst.Accept()
		if err != nil {
			panic(err)
		}
		wg.Add(1)
		go func() {
			t.handleRawConn(conn)
			wg.Done()
		}()
	}
}

// handleRawConn create a H2 Controller to deal with new conn
func (t *TripleServer) handleRawConn(conn net.Conn) error {
	h2Controller, err := NewH2Controller(conn, true, t.rpcService, t.url)
	if err != nil {
		return err
	}
	h2Controller.H2ShakeHand()
	return nil
}

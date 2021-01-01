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
	"github.com/apache/dubbo-go/common"
)

// It is interface of server for network communication.
// If you use getty as network communication, you should define GettyServer that implements this interface.
type Server interface {
	//invoke once for connection
	Start()
	//it is for destroy
	Stop()
}

// This is abstraction level. it is like facade.
type ExchangeServer struct {
	Server Server
	Url    *common.URL
}

// Create ExchangeServer
func NewExchangeServer(url *common.URL, server Server) *ExchangeServer {
	exchangServer := &ExchangeServer{
		Server: server,
		Url:    url,
	}
	return exchangServer
}

// start server
func (server *ExchangeServer) Start() {
	server.Server.Start()
}

// stop server
func (server *ExchangeServer) Stop() {
	server.Server.Stop()
}

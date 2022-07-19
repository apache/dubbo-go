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

package jsonrpc

import (
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	// JSONRPC
	// module name
	JSONRPC = "jsonrpc"
)

func init() {
	extension.SetProtocol(JSONRPC, GetProtocol)
}

var jsonrpcProtocol *JsonrpcProtocol

// JsonrpcProtocol is JSON RPC protocol.
type JsonrpcProtocol struct {
	protocol.BaseProtocol
	serverMap  map[string]*Server
	serverLock sync.Mutex
}

// NewJsonrpcProtocol creates JSON RPC protocol
func NewJsonrpcProtocol() *JsonrpcProtocol {
	return &JsonrpcProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

// Export JSON RPC service for remote invocation
func (jp *JsonrpcProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := strings.TrimPrefix(url.Path, "/")

	exporter := NewJsonrpcExporter(serviceKey, invoker, jp.ExporterMap())
	jp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[JSONRPC protocol] Export service: %s", url.String())

	// start server
	jp.openServer(url)

	return exporter
}

// Refer a remote JSON PRC service from registry
func (jp *JsonrpcProtocol) Refer(url *common.URL) protocol.Invoker {
	rtStr := config.GetConsumerConfig().RequestTimeout
	// the read order of requestTimeout is from url , if nil then from consumer config , if nil then default 3s. requestTimeout can be dynamically updated from config center.
	requestTimeout := url.GetParamDuration(constant.TimeoutKey, rtStr)
	// New Json rpc Invoker
	invoker := NewJsonrpcInvoker(url, NewHTTPClient(&HTTPOptions{
		HandshakeTimeout: time.Second, // todo config timeout config.GetConsumerConfig().ConnectTimeout,
		HTTPTimeout:      requestTimeout,
	}))
	jp.SetInvokers(invoker)
	logger.Infof("[JSONRPC Protocol] Refer service: %s", url.String())
	return invoker
}

// Destroy will destroy all invoker and exporter, so it only is called once.
func (jp *JsonrpcProtocol) Destroy() {
	logger.Infof("jsonrpcProtocol destroy.")

	jp.BaseProtocol.Destroy()

	// stop server
	for key, server := range jp.serverMap {
		delete(jp.serverMap, key)
		server.Stop()
	}
}

func (jp *JsonrpcProtocol) openServer(url *common.URL) {
	_, ok := jp.serverMap[url.Location]
	if !ok {
		_, loadOk := jp.ExporterMap().Load(strings.TrimPrefix(url.Path, "/"))
		if !loadOk {
			panic("[JsonrpcProtocol]" + url.Key() + "is not existing")
		}

		jp.serverLock.Lock()
		_, ok = jp.serverMap[url.Location]
		if !ok {
			srv := NewServer()
			jp.serverMap[url.Location] = srv
			srv.Start(url)
		}
		jp.serverLock.Unlock()
	}
}

// GetProtocol gets JSON RPC protocol.
func GetProtocol() protocol.Protocol {
	if jsonrpcProtocol == nil {
		jsonrpcProtocol = NewJsonrpcProtocol()
	}
	return jsonrpcProtocol
}

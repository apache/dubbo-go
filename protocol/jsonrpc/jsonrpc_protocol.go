// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonrpc

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

const JSONRPC = "jsonrpc"

func init() {
	extension.SetProtocol(JSONRPC, GetProtocol)
}

var jsonrpcProtocol *JsonrpcProtocol

type JsonrpcProtocol struct {
	protocol.BaseProtocol
	serverMap map[string]*Server
}

func NewDubboProtocol() *JsonrpcProtocol {
	return &JsonrpcProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func (jp *JsonrpcProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.Key()
	exporter := NewJsonrpcExporter(serviceKey, invoker, jp.ExporterMap())
	jp.SetExporterMap(serviceKey, exporter)
	log.Info("Export service: %s", url.String())

	// start server
	jp.openServer(url)

	return exporter
}

func (jp *JsonrpcProtocol) Refer(url common.URL) protocol.Invoker {
	invoker := NewJsonrpcInvoker(url, NewHTTPClient(&HTTPOptions{
		HandshakeTimeout: config.GetConsumerConfig().ConnectTimeout,
		HTTPTimeout:      config.GetConsumerConfig().RequestTimeout,
	}))
	jp.SetInvokers(invoker)
	log.Info("Refer service: %s", url.String())
	return invoker
}

func (jp *JsonrpcProtocol) Destroy() {
	log.Info("jsonrpcProtocol destroy.")

	jp.BaseProtocol.Destroy()

	// stop server
	for key, server := range jp.serverMap {
		delete(jp.serverMap, key)
		server.Stop()
	}
}

func (jp *JsonrpcProtocol) openServer(url common.URL) {
	exporter, ok := jp.ExporterMap().Load(url.Key())
	if !ok {
		panic("[JsonrpcProtocol]" + url.Key() + "is not existing")
	}
	srv := NewServer(exporter.(protocol.Exporter))
	jp.serverMap[url.Location] = srv
	srv.Start(url)
}

func GetProtocol() protocol.Protocol {
	if jsonrpcProtocol != nil {
		return jsonrpcProtocol
	}
	return NewDubboProtocol()
}

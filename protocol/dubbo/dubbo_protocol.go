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

package dubbo

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
)

const (
	DUBBO = "dubbo"
)

func init() {
	extension.SetProtocol(DUBBO, GetProtocol)
}

var (
	dubboProtocol *DubboProtocol
)

type DubboProtocol struct {
	protocol.BaseProtocol
	serverMap  map[string]*Server
	serverLock sync.Mutex
}

func NewDubboProtocol() *DubboProtocol {
	return &DubboProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.ServiceKey()
	exporter := NewDubboExporter(serviceKey, invoker, dp.ExporterMap())
	dp.SetExporterMap(serviceKey, exporter)
	logger.Infof("Export service: %s", url.String())

	// start server
	dp.openServer(url)
	return exporter
}

func (dp *DubboProtocol) Refer(url common.URL) protocol.Invoker {
	invoker := NewDubboInvoker(url, NewClient(Options{
		ConnectTimeout: config.GetConsumerConfig().ConnectTimeout,
		RequestTimeout: config.GetConsumerConfig().RequestTimeout,
	}))
	dp.SetInvokers(invoker)
	logger.Infof("Refer service: %s", url.String())
	return invoker
}

func (dp *DubboProtocol) Destroy() {
	logger.Infof("DubboProtocol destroy.")

	dp.BaseProtocol.Destroy()

	// stop server
	for key, server := range dp.serverMap {
		delete(dp.serverMap, key)
		server.Stop()
	}
}

func (dp *DubboProtocol) openServer(url common.URL) {
	_, ok := dp.serverMap[url.Location]
	if !ok {
		_, ok := dp.ExporterMap().Load(url.ServiceKey())
		if !ok {
			panic("[DubboProtocol]" + url.Key() + "is not existing")
		}

		dp.serverLock.Lock()
		_, ok = dp.serverMap[url.Location]
		if !ok {
			srv := NewServer()
			dp.serverMap[url.Location] = srv
			srv.Start(url)
		}
		dp.serverLock.Unlock()
	}
}

func GetProtocol() protocol.Protocol {
	if dubboProtocol == nil {
		dubboProtocol = NewDubboProtocol()
	}
	return dubboProtocol
}

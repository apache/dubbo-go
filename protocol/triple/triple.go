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

package triple

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	// TRIPLE protocol name
	TRIPLE = "triple"
)

var (
	tripleProtocol *TripleProtocol
)

func init() {
	extension.SetProtocol(TRIPLE, GetProtocol)
}

type TripleProtocol struct {
	protocol.BaseProtocol
	serverLock sync.Mutex
	serverMap  map[string]*Server
}

// Export TRIPLE service for remote invocation
func (tp *TripleProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	exporter := NewTripleExporter(serviceKey, invoker, tp.ExporterMap())
	tp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[TRIPLE Protocol] Export service: %s", url.String())
	tp.openServer(url)
	return exporter
}

func (tp *TripleProtocol) openServer(url *common.URL) {
	tp.serverLock.Lock()
	defer tp.serverLock.Unlock()

	if _, ok := tp.serverMap[url.Location]; ok {
		return
	}

	// todo: remove this logic?
	if _, ok := tp.ExporterMap().Load(url.ServiceKey()); !ok {
		panic("[GRPC_NEW Protocol]" + url.Key() + "is not existing")
	}

	srv := NewServer()
	tp.serverMap[url.Location] = srv
	srv.Start(url)
}

// Refer a remote triple service
func (tp *TripleProtocol) Refer(url *common.URL) protocol.Invoker {
	//client, err := NewClient(url)
	//if err != nil {
	//	logger.Warnf("can't dial the server: %s", url.Key())
	//	return nil
	//}
	invoker, err := NewTripleInvoker(url)
	if err != nil {
		logger.Warnf("can't dial the server: %s", url.Key())
		return nil
	}
	tp.SetInvokers(invoker)
	logger.Infof("[TRIPLE Protocol] Refer service: %s", url.String())
	return invoker
}

func (tp *TripleProtocol) Destroy() {
	logger.Infof("GrpcProtocol destroy.")

	tp.serverLock.Lock()
	defer tp.serverLock.Unlock()
	for key, server := range tp.serverMap {
		delete(tp.serverMap, key)
		server.GracefulStop()
	}

	tp.BaseProtocol.Destroy()
}

func NewTripleProtocol() *TripleProtocol {
	return &TripleProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func GetProtocol() protocol.Protocol {
	if tripleProtocol == nil {
		tripleProtocol = NewTripleProtocol()
	}
	return tripleProtocol
}

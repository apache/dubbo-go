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

package grpc_new

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
	"sync"
)

const (
	// GRPC_NEW protocol name
	GRPC_NEW = "grpc_new"
)

var (
	grpcNewProtocol *GrpcNewProtocol
)

func init() {
	extension.SetProtocol(GRPC_NEW, GetProtocol)
}

type GrpcNewProtocol struct {
	protocol.BaseProtocol
	serverLock sync.Mutex
	serverMap  map[string]*Server
}

// Export GRPC_NEW service for remote invocation
func (gnp *GrpcNewProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	exporter := NewGrpcNewExporter(serviceKey, invoker, gnp.ExporterMap())
	gnp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[GRPC_NEW Protocol] Export service: %s", url.String())
	gnp.openServer(url)
	return exporter
}

func (gnp *GrpcNewProtocol) openServer(url *common.URL) {
	gnp.serverLock.Lock()
	defer gnp.serverLock.Unlock()

	if _, ok := gnp.serverMap[url.Location]; ok {
		return
	}

	if _, ok := gnp.ExporterMap().Load(url.ServiceKey()); !ok {
		panic("[GRPC_NEW Protocol]" + url.Key() + "is not existing")
	}

	srv := NewServer()
	gnp.serverMap[url.Location] = srv
	srv.Start(url)
}

// Refer a remote gRPC service
func (gnp *GrpcNewProtocol) Refer(url *common.URL) protocol.Invoker {
	client, err := NewClient(url)
	if err != nil {
		logger.Warnf("can't dial the server: %s", url.Key())
		return nil
	}
	invoker := NewGrpcNewInvoker(url, client)
	gnp.SetInvokers(invoker)
	logger.Infof("[GRPC_NEW Protocol] Refer service: %s", url.String())
	return invoker
}

func (gnp *GrpcNewProtocol) Destroy() {
	logger.Infof("GrpcProtocol destroy.")

	gnp.serverLock.Lock()
	defer gnp.serverLock.Unlock()
	for key, server := range gnp.serverMap {
		delete(gnp.serverMap, key)
		server.GracefulStop()
	}

	gnp.BaseProtocol.Destroy()
}

func NewGrpcNewProtocol() *GrpcNewProtocol {
	return &GrpcNewProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func GetProtocol() protocol.Protocol {
	if grpcNewProtocol == nil {
		grpcNewProtocol = NewGrpcNewProtocol()
	}
	return grpcNewProtocol
}

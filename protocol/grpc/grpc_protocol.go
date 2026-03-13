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

package grpc

import (
	"context"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	// GRPC module name
	GRPC = "grpc"
)

func init() {
	extension.SetProtocol(GRPC, GetProtocol)

	// register graceful shutdown callback
	extension.SetGracefulShutdownCallback(GRPC, func(ctx context.Context) error {
		grpcProto := GetProtocol()
		if grpcProto == nil {
			return nil
		}

		gp, ok := grpcProto.(*GrpcProtocol)
		if !ok {
			return nil
		}

		gp.serverLock.Lock()
		defer gp.serverLock.Unlock()

		for _, server := range gp.serverMap {
			server.SetAllServicesNotServing()
		}

		return nil
	})
}

var grpcProtocol *GrpcProtocol

// GrpcProtocol is gRPC protocol
type GrpcProtocol struct {
	base.BaseProtocol
	serverMap  map[string]*Server
	serverLock sync.Mutex
}

// NewGRPCProtocol creates new gRPC protocol
func NewGRPCProtocol() *GrpcProtocol {
	return &GrpcProtocol{
		BaseProtocol: base.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

// Export gRPC service for remote invocation
func (gp *GrpcProtocol) Export(invoker base.Invoker) base.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	exporter := NewGrpcExporter(serviceKey, invoker, gp.ExporterMap())
	gp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[GRPC Protocol] Export service: %s", url.String())
	srv := gp.openServer(url)
	srv.SetServingStatus(serviceKey, grpc_health_v1.HealthCheckResponse_SERVING)
	return exporter
}

func (gp *GrpcProtocol) openServer(url *common.URL) *Server {
	gp.serverLock.Lock()
	defer gp.serverLock.Unlock()

	if srv, ok := gp.serverMap[url.Location]; ok {
		return srv
	}

	if _, ok := gp.ExporterMap().Load(url.ServiceKey()); !ok {
		panic("[GrpcProtocol]" + url.Key() + "is not existing")
	}

	srv := NewServer()
	gp.serverMap[url.Location] = srv
	srv.Start(url)
	return srv
}

// Refer a remote gRPC service
func (gp *GrpcProtocol) Refer(url *common.URL) base.Invoker {
	client, err := NewClient(url)
	if err != nil {
		logger.Warnf("can't dial the server: %s", url.Key())
		return nil
	}
	invoker := NewGrpcInvoker(url, client)
	gp.SetInvokers(invoker)
	logger.Infof("[GRPC Protcol] Refer service: %s", url.String())
	return invoker
}

// Destroy will destroy gRPC all invoker and exporter, so it only is called once.
func (gp *GrpcProtocol) Destroy() {
	logger.Infof("GrpcProtocol destroy.")

	gp.serverLock.Lock()
	defer gp.serverLock.Unlock()
	for key, server := range gp.serverMap {
		delete(gp.serverMap, key)
		server.GracefulStop()
	}

	gp.BaseProtocol.Destroy()
}

// GetProtocol gets gRPC protocol, will create if null.
func GetProtocol() base.Protocol {
	if grpcProtocol == nil {
		grpcProtocol = NewGRPCProtocol()
	}
	return grpcProtocol
}

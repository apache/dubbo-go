/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
  "sync"

  "github.com/apache/dubbo-go/common"
  "github.com/apache/dubbo-go/common/extension"
  "github.com/apache/dubbo-go/common/logger"
  "github.com/apache/dubbo-go/protocol"
)

const GRPC = "grpc"

func init() {
  extension.SetProtocol(GRPC, GetProtocol)
}

var grpcProtocol *GrpcProtocol

type GrpcProtocol struct {
  protocol.BaseProtocol
  serverMap  map[string]*Server
  serverLock sync.Mutex
}

func NewGRPCProtocol() *GrpcProtocol {
  return nil
}

func (gp *GrpcProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
  url := invoker.GetUrl()
  serviceKey := url.ServiceKey()
  exporter := NewGrpcExporter(serviceKey, invoker, gp.ExporterMap())
  gp.SetExporterMap(serviceKey, exporter)
  logger.Infof("Export service: %s", url.String())
  gp.openServer(url)
  return exporter
}

func (gp *GrpcProtocol) openServer(url common.URL) {
  return
}

func (gp *GrpcProtocol) Refer(url common.URL) protocol.Invoker {
  invoker := NewGrpcInvoker(url, NewClient())
  gp.SetInvokers(invoker)
  logger.Infof("Refer service: %s", url.String())
  return invoker
}

func (gp *GrpcProtocol) Destroy() {
  logger.Infof("GrpcProtocol destroy.")

  gp.BaseProtocol.Destroy()

  for key, server := range gp.serverMap {
    delete(gp.serverMap, key)
    server.Stop()
  }
}

func GetProtocol() protocol.Protocol {
  if grpcProtocol == nil {
    grpcProtocol = NewGRPCProtocol()
  }
  return grpcProtocol
}

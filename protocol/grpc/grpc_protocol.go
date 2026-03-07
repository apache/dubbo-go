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
)

const (
	// GRPC module name
	GRPC = "grpc"
)

func init() {
	// 注册 gRPC 协议到扩展点
	// 这样可以通过 extension.GetProtocol("grpc") 获取 gRPC 协议实例
	extension.SetProtocol(GRPC, GetProtocol)

	// 注册优雅下线回调
	// 当应用触发优雅下线时，会调用此回调通知所有连接的 Consumer 即将关闭
	// 回调机制避免循环导入问题（graceful_shutdown -> protocol/grpc -> dubbo根包 -> client -> graceful_shutdown）
	extension.SetGracefulShutdownCallback(GRPC, func(ctx context.Context) error {
		// 1. 获取 gRPC 协议实例
		// GetProtocol() 会返回全局唯一的 GrpcProtocol 实例
		grpcProto := GetProtocol()
		if grpcProto == nil {
			return nil
		}

		// 2. 类型断言，将通用 Protocol 接口转换为 GrpcProtocol
		// 只有 GrpcProtocol 才有 serverMap 字段
		gp, ok := grpcProto.(*GrpcProtocol)
		if !ok {
			return nil
		}

		// 3. 加锁保护 serverMap（避免并发问题）
		// 因为优雅下线可能被多个信号触发
		gp.serverLock.Lock()
		defer gp.serverLock.Unlock()

		// 4. 遍历所有 gRPC Server，调用 GracefulStop 优雅关闭
		// GracefulStop 会：
		//   - 停止接收新请求
		//   - 等待正在处理的请求完成
		//   - 然后关闭连接
		// 这样 Consumer 会收到通知，知道 Provider 即将关闭
		for _, server := range gp.serverMap {
			server.GracefulStop()
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
	gp.openServer(url)
	return exporter
}

func (gp *GrpcProtocol) openServer(url *common.URL) {
	gp.serverLock.Lock()
	defer gp.serverLock.Unlock()

	if _, ok := gp.serverMap[url.Location]; ok {
		return
	}

	if _, ok := gp.ExporterMap().Load(url.ServiceKey()); !ok {
		panic("[GrpcProtocol]" + url.Key() + "is not existing")
	}

	srv := NewServer()
	gp.serverMap[url.Location] = srv
	srv.Start(url)
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

// GetServerMap returns all gRPC servers
func (gp *GrpcProtocol) GetServerMap() map[string]*Server {
	return gp.serverMap
}

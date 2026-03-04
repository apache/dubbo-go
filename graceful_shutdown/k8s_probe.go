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

package graceful_shutdown

import (
	"github.com/dubbogo/gost/log/logger"
)

// ShutdownK8sProbe 关闭 K8s 探针
// 调用 Triple/gRPC 健康检查服务的 Shutdown 方法
// 将所有服务状态设置为 NOT_SERVING
func ShutdownK8sProbe() {
	logger.Info("Graceful shutdown --- Shutdown K8s probe.")

	// 调用 Triple 健康检查服务的 Shutdown 方法
	err := shutdownTripleHealthProbe()
	if err != nil {
		logger.Warnf("Graceful shutdown --- Shutdown Triple health probe failed: %v", err)
	}

	// TODO: 调用 gRPC 健康检查服务的 Shutdown 方法
	// err = shutdownGrpcHealthProbe()
	// if err != nil {
	// 	logger.Warnf("Graceful shutdown --- Shutdown gRPC health probe failed: %v", err)
	// }

	logger.Info("Graceful shutdown --- K8s probe shutdown completed.")
}

// shutdownTripleHealthProbe 关闭 Triple 健康检查服务
func shutdownTripleHealthProbe() error {
	// TODO: 获取 Triple 健康检查服务并调用 Shutdown 方法
	// healthServer := extension.GetHealthServer("tri")
	// if healthServer != nil {
	// 	return healthServer.Shutdown()
	// }
	logger.Info("Graceful shutdown --- Shutdown Triple health probe (TODO: implement)")
	return nil
}

// shutdownGrpcHealthProbe 关闭 gRPC 健康检查服务
func shutdownGrpcHealthProbe() error {
	// TODO: 获取 gRPC 健康检查服务并调用 Shutdown 方法
	logger.Info("Graceful shutdown --- Shutdown gRPC health probe (TODO: implement)")
	return nil
}

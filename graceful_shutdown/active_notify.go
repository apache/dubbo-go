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
	"time"

	"github.com/dubbogo/gost/log/logger"
)

const (
	// defaultActiveNotifyTimeout 主动通知超时时间
	defaultActiveNotifyTimeout = 3 * time.Second
)

// NotifyLongConnectionConsumers 主动通知长连接 Consumer
// 该函数遍历所有已注册的协议，向每个连接发送关闭通知
// 支持超时控制和重试机制
func NotifyLongConnectionConsumers() {
	logger.Info("Graceful shutdown --- Notify long connection consumers.")

	timeout := defaultActiveNotifyTimeout
	done := make(chan struct{})

	go func() {
		// 通知 Triple 协议 Consumer
		notifyTripleConsumers()
		// 通知 gRPC 协议 Consumer
		notifyGrpcConsumers()
		// 通知 Dubbo 协议 Consumer
		notifyDubboConsumers()
		close(done)
	}()

	select {
	case <-done:
		// 通知完成
		logger.Info("Graceful shutdown --- Notify long connection consumers completed.")
	case <-time.After(timeout):
		// 超时，强制进入下一步
		logger.Warn("Graceful shutdown --- Notify long connection consumers timeout, continuing...")
	}
}

// notifyTripleConsumers 通知 Triple 协议 Consumer
func notifyTripleConsumers() {
	// TODO: 获取 Triple 协议的所有连接
	// TODO: 向每个连接发送关闭通知
	// TODO: 支持重试机制
	logger.Info("Graceful shutdown --- Notify Triple consumers (TODO: implement)")
}

// notifyGrpcConsumers 通知 gRPC 协议 Consumer
func notifyGrpcConsumers() {
	// TODO: 获取 gRPC 协议的所有连接
	// TODO: 向每个连接发送关闭通知
	// TODO: 支持重试机制
	logger.Info("Graceful shutdown --- Notify gRPC consumers (TODO: implement)")
}

// notifyDubboConsumers 通知 Dubbo 协议 Consumer
func notifyDubboConsumers() {
	// TODO: 获取 Dubbo 协议的所有连接
	// TODO: 向每个连接发送关闭通知
	// TODO: 支持重试机制
	logger.Info("Graceful shutdown --- Notify Dubbo consumers (TODO: implement)")
}

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

package extension

import (
	"container/list"
	"context"
)

// GracefulShutdownCallback 优雅下线回调函数
// name: 协议名称，如 "grpc", "tri", "dubbo"
// 返回 error 表示通知失败
type GracefulShutdownCallback func(ctx context.Context) error

var (
	customShutdownCallbacks = list.New()
	gracefulShutdownCallbacks = make(map[string]GracefulShutdownCallback)
)

// AddCustomShutdownCallback 添加自定义关闭回调
// 注意：回调顺序不保证
func AddCustomShutdownCallback(callback func()) {
	customShutdownCallbacks.PushBack(callback)
}

// GetAllCustomShutdownCallbacks 获取所有自定义关闭回调
func GetAllCustomShutdownCallbacks() *list.List {
	return customShutdownCallbacks
}

// SetGracefulShutdownCallback 设置协议级别的优雅下线回调
// name: 协议名称，如 "grpc", "tri", "dubbo"
func SetGracefulShutdownCallback(name string, f GracefulShutdownCallback) {
	gracefulShutdownCallbacks[name] = f
}

// GetGracefulShutdownCallback 获取指定协议的优雅下线回调
func GetGracefulShutdownCallback(name string) (GracefulShutdownCallback, bool) {
	f, ok := gracefulShutdownCallbacks[name]
	return f, ok
}

// GetAllGracefulShutdownCallbacks 获取所有协议的优雅下线回调
func GetAllGracefulShutdownCallbacks() map[string]GracefulShutdownCallback {
	return gracefulShutdownCallbacks
}

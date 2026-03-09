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
	"sync"
)

// GracefulShutdownCallback is the callback for graceful shutdown
// name: protocol name such as "grpc", "tri", "dubbo"
// returns error if notify failed
type GracefulShutdownCallback func(ctx context.Context) error

var (
	customShutdownCallbacks     = list.New()
	gracefulShutdownCallbacksMu sync.RWMutex
	gracefulShutdownCallbacks   = make(map[string]GracefulShutdownCallback)
)

/**
 * AddCustomShutdownCallback
 * you should not make any assumption about the order.
 * For example, if you have more than one callbacks, and you wish the order is:
 * callback1()
 * callback2()
 * ...
 * callbackN()
 * Then you should put then together:
 * func callback() {
 *     callback1()
 *     callback2()
 *     ...
 *     callbackN()
 * }
 * I think the order of custom callbacks should be decided by the users.
 * Even though I can design a mechanism to support the ordered custom callbacks,
 * the benefit of that mechanism is low.
 * And it may introduce much complication for another users.
 */
// AddCustomShutdownCallback adds custom shutdown callback
func AddCustomShutdownCallback(callback func()) {
	customShutdownCallbacks.PushBack(callback)
}

// GetAllCustomShutdownCallbacks returns all custom shutdown callbacks
func GetAllCustomShutdownCallbacks() *list.List {
	return customShutdownCallbacks
}

// SetGracefulShutdownCallback sets protocol-level graceful shutdown callback
func SetGracefulShutdownCallback(name string, f GracefulShutdownCallback) {
	gracefulShutdownCallbacksMu.Lock()
	defer gracefulShutdownCallbacksMu.Unlock()
	gracefulShutdownCallbacks[name] = f
}

// GetGracefulShutdownCallback returns protocol's graceful shutdown callback
func GetGracefulShutdownCallback(name string) (GracefulShutdownCallback, bool) {
	gracefulShutdownCallbacksMu.RLock()
	defer gracefulShutdownCallbacksMu.RUnlock()
	f, ok := gracefulShutdownCallbacks[name]
	return f, ok
}

// GetAllGracefulShutdownCallbacks returns all protocol's graceful shutdown callbacks
func GetAllGracefulShutdownCallbacks() map[string]GracefulShutdownCallback {
	gracefulShutdownCallbacksMu.RLock()
	defer gracefulShutdownCallbacksMu.RUnlock()

	callbacks := make(map[string]GracefulShutdownCallback, len(gracefulShutdownCallbacks))
	for name, callback := range gracefulShutdownCallbacks {
		callbacks[name] = callback
	}

	return callbacks
}

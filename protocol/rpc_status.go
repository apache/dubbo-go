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

package protocol

import (
	"sync"
	"sync/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
)

var (
	methodStatistics sync.Map // url -> { methodName : RpcStatus}
)

// RpcStatus ...
type RpcStatus struct {
	active int32
}

// GetActive ...
func (rpc *RpcStatus) GetActive() int32 {
	return atomic.LoadInt32(&rpc.active)
}

// GetStatus ...
func GetStatus(url common.URL, methodName string) *RpcStatus {
	identifier := url.Key()
	methodMap, found := methodStatistics.Load(identifier)
	if !found {
		methodMap = &sync.Map{}
		methodStatistics.Store(identifier, methodMap)
	}

	methodActive := methodMap.(*sync.Map)
	rpcStatus, found := methodActive.Load(methodName)
	if !found {
		rpcStatus = &RpcStatus{}
		methodActive.Store(methodName, rpcStatus)
	}

	status := rpcStatus.(*RpcStatus)
	return status
}

// BeginCount ...
func BeginCount(url common.URL, methodName string) {
	beginCount0(GetStatus(url, methodName))
}

// EndCount ...
func EndCount(url common.URL, methodName string) {
	endCount0(GetStatus(url, methodName))
}

// private methods
func beginCount0(rpcStatus *RpcStatus) {
	atomic.AddInt32(&rpcStatus.active, 1)
}

func endCount0(rpcStatus *RpcStatus) {
	atomic.AddInt32(&rpcStatus.active, -1)
}

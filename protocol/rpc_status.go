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
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
)

var (
	methodStatistics sync.Map // url -> { methodName : RPCStatus}
	serviceStatistic sync.Map // url -> RPCStatus
)

type RPCStatus struct {
	active                        int32
	failed                        int32
	total                         int32
	totalElapsed                  int64
	failedElapsed                 int64
	maxElapsed                    int64
	failedMaxElapsed              int64
	succeededMaxElapsed           int64
	successiveRequestFailureCount int32
	lastRequestFailedTimestamp    int64
}

func (rpc *RPCStatus) GetActive() int32 {
	return atomic.LoadInt32(&rpc.active)
}

func (rpc *RPCStatus) GetFailed() int32 {
	return atomic.LoadInt32(&rpc.failed)
}

func (rpc *RPCStatus) GetTotal() int32 {
	return atomic.LoadInt32(&rpc.total)
}

func (rpc *RPCStatus) GetTotalElapsed() int64 {
	return atomic.LoadInt64(&rpc.totalElapsed)
}

func (rpc *RPCStatus) GetFailedElapsed() int64 {
	return atomic.LoadInt64(&rpc.failedElapsed)
}

func (rpc *RPCStatus) GetMaxElapsed() int64 {
	return atomic.LoadInt64(&rpc.maxElapsed)
}

func (rpc *RPCStatus) GetFailedMaxElapsed() int64 {
	return atomic.LoadInt64(&rpc.failedMaxElapsed)
}

func (rpc *RPCStatus) GetSucceededMaxElapsed() int64 {
	return atomic.LoadInt64(&rpc.succeededMaxElapsed)
}

func (rpc *RPCStatus) GetLastRequestFailedTimestamp() int64 {
	return atomic.LoadInt64(&rpc.lastRequestFailedTimestamp)
}

func (rpc *RPCStatus) GetSuccessiveRequestFailureCount() int32 {
	return atomic.LoadInt32(&rpc.successiveRequestFailureCount)
}

func GetURLStatus(url common.URL) *RPCStatus {
	rpcStatus, _ := serviceStatistic.LoadOrStore(url.Key(), &RPCStatus{})
	return rpcStatus.(*RPCStatus)
}

func GetMethodStatus(url common.URL, methodName string) *RPCStatus {
	identifier := url.Key()
	methodMap, found := methodStatistics.Load(identifier)
	if !found {
		methodMap = &sync.Map{}
		methodStatistics.Store(identifier, methodMap)
	}

	methodActive := methodMap.(*sync.Map)
	rpcStatus, found := methodActive.Load(methodName)
	if !found {
		rpcStatus = &RPCStatus{}
		methodActive.Store(methodName, rpcStatus)
	}

	status := rpcStatus.(*RPCStatus)
	return status
}

func BeginCount(url common.URL, methodName string) {
	beginCount0(GetURLStatus(url))
	beginCount0(GetMethodStatus(url, methodName))
}

func EndCount(url common.URL, methodName string, elapsed int64, succeeded bool) {
	endCount0(GetURLStatus(url), elapsed, succeeded)
	endCount0(GetMethodStatus(url, methodName), elapsed, succeeded)
}

// private methods
func beginCount0(rpcStatus *RPCStatus) {
	atomic.AddInt32(&rpcStatus.active, 1)
}

func endCount0(rpcStatus *RPCStatus, elapsed int64, succeeded bool) {
	atomic.AddInt32(&rpcStatus.active, -1)
	atomic.AddInt32(&rpcStatus.total, 1)
	atomic.AddInt64(&rpcStatus.totalElapsed, elapsed)

	if rpcStatus.maxElapsed < elapsed {
		atomic.StoreInt64(&rpcStatus.maxElapsed, elapsed)
	}
	if succeeded {
		if rpcStatus.succeededMaxElapsed < elapsed {
			atomic.StoreInt64(&rpcStatus.succeededMaxElapsed, elapsed)
		}
		atomic.StoreInt32(&rpcStatus.successiveRequestFailureCount, 0)
	} else {
		atomic.StoreInt64(&rpcStatus.lastRequestFailedTimestamp, time.Now().Unix())
		atomic.AddInt32(&rpcStatus.successiveRequestFailureCount, 1)
		atomic.AddInt32(&rpcStatus.failed, 1)
		atomic.AddInt64(&rpcStatus.failedElapsed, elapsed)
		if rpcStatus.failedMaxElapsed < elapsed {
			atomic.StoreInt64(&rpcStatus.failedMaxElapsed, elapsed)
		}
	}
}

func CurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

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
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
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
	serviceStateMap  sync.Map
)

func init() {
}

// RPCStatus is URL statistics.
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

// GetActive gets active.
func (rpc *RPCStatus) GetActive() int32 {
	return atomic.LoadInt32(&rpc.active)
}

// GetFailed gets failed.
func (rpc *RPCStatus) GetFailed() int32 {
	return atomic.LoadInt32(&rpc.failed)
}

// GetTotal gets total.
func (rpc *RPCStatus) GetTotal() int32 {
	return atomic.LoadInt32(&rpc.total)
}

// GetTotalElapsed gets total elapsed.
func (rpc *RPCStatus) GetTotalElapsed() int64 {
	return atomic.LoadInt64(&rpc.totalElapsed)
}

// GetFailedElapsed gets failed elapsed.
func (rpc *RPCStatus) GetFailedElapsed() int64 {
	return atomic.LoadInt64(&rpc.failedElapsed)
}

// GetMaxElapsed gets max elapsed.
func (rpc *RPCStatus) GetMaxElapsed() int64 {
	return atomic.LoadInt64(&rpc.maxElapsed)
}

// GetFailedMaxElapsed gets failed max elapsed.
func (rpc *RPCStatus) GetFailedMaxElapsed() int64 {
	return atomic.LoadInt64(&rpc.failedMaxElapsed)
}

// GetSucceededMaxElapsed gets succeeded max elapsed.
func (rpc *RPCStatus) GetSucceededMaxElapsed() int64 {
	return atomic.LoadInt64(&rpc.succeededMaxElapsed)
}

// GetLastRequestFailedTimestamp gets last request failed timestamp.
func (rpc *RPCStatus) GetLastRequestFailedTimestamp() int64 {
	return atomic.LoadInt64(&rpc.lastRequestFailedTimestamp)
}

// GetSuccessiveRequestFailureCount gets successive request failure count.
func (rpc *RPCStatus) GetSuccessiveRequestFailureCount() int32 {
	return atomic.LoadInt32(&rpc.successiveRequestFailureCount)
}

// GetURLStatus get URL RPC status.
func GetURLStatus(url *common.URL) *RPCStatus {
	rpcStatus, found := serviceStatistic.Load(url.Key())
	if !found {
		rpcStatus, _ = serviceStatistic.LoadOrStore(url.Key(), &RPCStatus{})
	}
	return rpcStatus.(*RPCStatus)
}

// GetMethodStatus get method RPC status.
func GetMethodStatus(url *common.URL, methodName string) *RPCStatus {
	identifier := url.Key()
	methodMap, found := methodStatistics.Load(identifier)
	if !found {
		methodMap, _ = methodStatistics.LoadOrStore(identifier, &sync.Map{})
	}

	methodActive := methodMap.(*sync.Map)
	rpcStatus, found := methodActive.Load(methodName)
	if !found {
		rpcStatus, _ = methodActive.LoadOrStore(methodName, &RPCStatus{})
	}

	status := rpcStatus.(*RPCStatus)
	return status
}

// BeginCount gets begin count.
func BeginCount(url *common.URL, methodName string) {
	beginCount0(GetURLStatus(url))
	beginCount0(GetMethodStatus(url, methodName))
}

// EndCount gets end count.
func EndCount(url *common.URL, methodName string, elapsed int64, succeeded bool) {
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
		atomic.StoreInt64(&rpcStatus.lastRequestFailedTimestamp, CurrentTimeMillis())
		atomic.AddInt32(&rpcStatus.successiveRequestFailureCount, 1)
		atomic.AddInt32(&rpcStatus.failed, 1)
		atomic.AddInt64(&rpcStatus.failedElapsed, elapsed)
		if rpcStatus.failedMaxElapsed < elapsed {
			atomic.StoreInt64(&rpcStatus.failedMaxElapsed, elapsed)
		}
	}
}

// CurrentTimeMillis get current timestamp
func CurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// Destroy is used to clean all status
func CleanAllStatus() {
	delete1 := func(key, _ interface{}) bool {
		methodStatistics.Delete(key)
		return true
	}
	methodStatistics.Range(delete1)
	delete2 := func(key, _ interface{}) bool {
		serviceStatistic.Delete(key)
		return true
	}
	serviceStatistic.Range(delete2)
	delete3 := func(_, value interface{}) bool {
		if v, ok := value.(*ServiceHealthState); ok {
			v.blackList.Range(func(key, value interface{}) bool {
				v.blackList.Delete(key)
				return true
			})
		}
		return true
	}
	serviceStateMap.Range(delete3)
}

// if the ip is changed in kubernetes, then the ip will not exist. So we should recycle the invoker.
// there are two ways :
// 1. we should know when the invoker is dropped and clear the data from blacklist
// 2. we add a counter to collect the retry times. After 512 times(just now) retry, we should clear it.
type invokerState struct {
	invoker    Invoker
	retryTimes int32
}

func newInvokeState(invoker Invoker) *invokerState {
	return &invokerState{
		invoker: invoker,
	}
}

func (s *invokerState) increateRetryTimes() {
	s.retryTimes++
}

// GetInvokerHealthyStatus get invoker's conn healthy status
func GetInvokerHealthyStatus(invoker Invoker) bool {
	if v, ok := serviceStateMap.Load(invoker.GetUrl().ServiceKey()); ok {
		if state, ok := v.(*ServiceHealthState); ok {
			_, found := state.blackList.Load(invoker.GetUrl().Key())
			return !found
		}
	}
	return true
}

// nolint
func GetAndRefreshState(url *common.URL) bool {
	if v, ok := serviceStateMap.Load(url.ServiceKey()); ok {
		if state, ok := v.(*ServiceHealthState); ok {
			return atomic.CompareAndSwapInt32(state.rebuildRoute, 1, 0)
		}
	}
	return false
}

type ServiceHealthState struct {
	serviceKey string
	//if some process in refresh
	refreshState *int32
	refresh      atomic.Value
	rebuildRoute *int32
	blackList    sync.Map // store unhealthy url blackList
}

func NewServiceState(serviceKey string) *ServiceHealthState {
	if v, ok := serviceStateMap.Load(serviceKey); ok {
		return v.(*ServiceHealthState)
	}
	serviceState := &ServiceHealthState{
		refreshState: new(int32),
		rebuildRoute: new(int32),
		serviceKey:   serviceKey,
	}
	serviceStateMap.Store(serviceKey, serviceState)
	return serviceState
}

func (s *ServiceHealthState) reset() {
	s.refresh.Store(false)
	atomic.StoreInt32(s.rebuildRoute, 0)
	s.blackList.Range(func(key, value interface{}) bool {
		s.blackList.Delete(key)
		return true
	})
}

func (s *ServiceHealthState) configNeedRefresh(needRefresh bool) {
	s.refresh.Store(needRefresh)
}

func (s *ServiceHealthState) needRefresh() bool {
	v := s.refresh.Load()
	if v == nil {
		return false
	}
	return v.(bool)
}

// SetInvokerUnhealthyStatus add target invoker to black list
func (s *ServiceHealthState) SetInvokerUnhealthyStatus(invoker Invoker) {
	s.configNeedRefresh(true)
	s.blackList.Store(invoker.GetUrl().Key(), newInvokeState(invoker))
	logger.Infof("Add invoker ip(%s) to black list for service(%s)", invoker.GetUrl().Location, invoker.GetUrl().ServiceKey())
	s.activeBlackListCacheDirty()
}

// RemoveInvokerUnhealthyStatus remove unhealthy status of target invoker from blacklist
func (s *ServiceHealthState) RemoveInvokerUnhealthyStatus(invoker Invoker) {
	s.blackList.Delete(invoker.GetUrl().Key())
	logger.Infof("Remove invoker ip(%s) from black list for service(%s)", invoker.GetUrl().Location, invoker.GetUrl().ServiceKey())
	s.activeBlackListCacheDirty()
}

// GetBlackListInvokers get at most size of blockSize invokers from black list
func (s *ServiceHealthState) GetBlackListInvokers(blockSize int) []*invokerState {
	resultIvks := make([]*invokerState, 0, blockSize)
	s.blackList.Range(func(k, v interface{}) bool {
		if v == nil {
			return true
		}
		resultIvks = append(resultIvks, v.(*invokerState))
		if len(resultIvks) == blockSize {
			return false
		}
		return true
	})
	return resultIvks
}

// RemoveUrlKeyUnhealthyStatus called when event of provider unregister, delete from black list
func (s *ServiceHealthState) RemoveUrlKeyUnhealthyStatus(key string) {
	if _, ok := s.blackList.Load(key); ok {
		s.blackList.Delete(key)
		s.activeBlackListCacheDirty()
	}
	logger.Info("Remove invoker key = ", key, " from black list")
}

func (s *ServiceHealthState) activeBlackListCacheDirty() {
	atomic.StoreInt32(s.rebuildRoute, 1)
}

// TryRefreshBlackList start 3 gr to check at most block=16 invokers in black list
// if target invoker is available, then remove it from black list
func (s *ServiceHealthState) TryRefreshBlackList() {
	if s.needRefresh() {
		go s.refreshBlackList()
	}
}

func (s *ServiceHealthState) refreshBlackList() {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("try to refresh black list failed: %s, %+v", s.serviceKey, r)
		}
	}()

	if atomic.CompareAndSwapInt32(s.refreshState, 0, 1) {
		defer func() {
			atomic.CompareAndSwapInt32(s.refreshState, 1, 0)
		}()

		ivkStates := s.GetBlackListInvokers(constant.DEFAULT_BLACK_LIST_RECOVER_BLOCK)
		logger.Debug("blackList len = ", len(ivkStates))
		if len(ivkStates) == 0 {
			logger.Infof("there is no data in black list, and will not refresh black list.")
			s.configNeedRefresh(false)
			return
		}
		wg := sync.WaitGroup{}
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(ivks []*invokerState, i int) {
				defer wg.Done()
				for j, _ := range ivks {
					if j%3-i == 0 && ivks[j].invoker.IsAvailable() {
						s.RemoveInvokerUnhealthyStatus(ivks[j].invoker)
					} else {
						ivks[j].increateRetryTimes()
					}
					// if the ip is changed in kubernetes, then the ip will not exist. So we should recycle the invoker.
					if ivks[j].retryTimes > constant.DEFAULT_BLACK_LIST_MAX_RETRY_TIMES {
						s.RemoveInvokerUnhealthyStatus(ivks[j].invoker)
					}
				}
			}(ivkStates, i)
		}
		wg.Wait()
	}
}

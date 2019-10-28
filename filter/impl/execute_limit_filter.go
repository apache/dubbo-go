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

package impl

import (
	"strconv"
	"sync"
	"sync/atomic"
)

import (
	"github.com/modern-go/concurrent"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	_ "github.com/apache/dubbo-go/filter/common/impl"
	"github.com/apache/dubbo-go/protocol"
)

const name = "execute"

func init() {
	extension.SetFilter(name, GetExecuteLimitFilter)
}

/**
 * The filter will limit the number of in-progress request and it's thread-safe.
 * example:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   execute.limit: 200 # the name of MethodServiceTpsLimiterImpl. if the value < 0, invocation will be ignored.
 *   execute.limit.rejected.handle: "default" # the name of rejected handler
 *   methods:
 *    - name: "GetUser"
 *      execute.limit: 20, # in this case, this configuration in service-level will be ignored.
 *    - name: "UpdateUser"
 *      execute.limit: -1, # If the rate<0, the method will be ignored
 *    - name: "DeleteUser"
 *      execute.limit.rejected.handle: "customHandler" # Using the custom handler to do something when the request was rejected.
 *    - name: "AddUser"
 * From the example, the configuration in service-level is 200, and the configuration of method GetUser is 20.
 * it means that, the GetUser will be counted separately.
 * The configuration of method UpdateUser is -1, so the invocation for it will not be counted.
 * So the method DeleteUser and method AddUser will be limited by service-level configuration.
 * Sometimes we want to do something, like log the request or return default value when the request is over limitation.
 * Then you can implement the RejectedExecutionHandler interface and register it by invoking SetRejectedExecutionHandler.
 */
type ExecuteLimitFilter struct {
	executeState *concurrent.Map
}

type ExecuteState struct {
	concurrentCount int64
}

func (ef *ExecuteLimitFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	methodConfigPrefix := "methods." + invocation.MethodName() + "."
	url := invoker.GetUrl()
	limitTarget := url.ServiceKey()
	limitRateConfig := constant.DEFAULT_EXECUTE_LIMIT

	methodLevelConfig := url.GetParam(methodConfigPrefix+constant.EXECUTE_LIMIT_KEY, "")
	if len(methodLevelConfig) > 0 {
		// we have the method-level configuration
		limitTarget = limitTarget + "#" + invocation.MethodName()
		limitRateConfig = methodLevelConfig
	} else {
		limitRateConfig = url.GetParam(constant.EXECUTE_LIMIT_KEY, constant.DEFAULT_EXECUTE_LIMIT)
	}

	limitRate, err := strconv.ParseInt(limitRateConfig, 0, 0)
	if err != nil {
		logger.Errorf("The configuration of execute.limit is invalid: %s", limitRateConfig)
		return &protocol.RPCResult{}
	}

	if limitRate < 0 {
		return invoker.Invoke(invocation)
	}

	state, _ := ef.executeState.LoadOrStore(limitTarget, &ExecuteState{
		concurrentCount: 0,
	})

	concurrentCount := state.(*ExecuteState).increase()
	defer state.(*ExecuteState).decrease()
	if concurrentCount > limitRate {
		logger.Errorf("The invocation was rejected due to over the execute limitation, url: %s ", url.String())
		rejectedHandlerConfig := url.GetParam(methodConfigPrefix+constant.EXECUTE_REJECTED_EXECUTION_HANDLER_KEY,
			url.GetParam(constant.EXECUTE_REJECTED_EXECUTION_HANDLER_KEY, constant.DEFAULT_KEY))
		return extension.GetRejectedExecutionHandler(rejectedHandlerConfig).RejectedExecution(url, invocation)
	}

	return invoker.Invoke(invocation)
}

func (ef *ExecuteLimitFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func (state *ExecuteState) increase() int64 {
	return atomic.AddInt64(&state.concurrentCount, 1)
}

func (state *ExecuteState) decrease() {
	atomic.AddInt64(&state.concurrentCount, -1)
}

var executeLimitOnce sync.Once
var executeLimitFilter *ExecuteLimitFilter

func GetExecuteLimitFilter() filter.Filter {
	executeLimitOnce.Do(func() {
		executeLimitFilter = &ExecuteLimitFilter{
			executeState: concurrent.NewMap(),
		}
	})
	return executeLimitFilter
}

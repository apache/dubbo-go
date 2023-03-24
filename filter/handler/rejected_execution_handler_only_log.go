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

package handler

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	// HandlerName handler name
	HandlerName = "log"
)

func init() {
	// this implementation is the the default implementation of RejectedExecutionHandler
	extension.SetRejectedExecutionHandler(HandlerName, GetOnlyLogRejectedExecutionHandler)
	extension.SetRejectedExecutionHandler(constant.DefaultKey, GetOnlyLogRejectedExecutionHandler)
}

var (
	onlyLogHandlerInstance *OnlyLogRejectedExecutionHandler
	onlyLogHandlerOnce     sync.Once
)

// OnlyLogRejectedExecutionHandler implements the RejectedExecutionHandler
/**
 * This implementation only logs the invocation info.
 * it always return en error inside the result.
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" # the name of limiter
 *   tps.limit.rejected.handler: "default" or "log"
 *   methods:
 *    - name: "GetUser"
 * OnlyLogRejectedExecutionHandler is designed to be singleton
 */
type OnlyLogRejectedExecutionHandler struct{}

// RejectedExecution will do nothing, it only log the invocation.
func (handler *OnlyLogRejectedExecutionHandler) RejectedExecution(url *common.URL,
	_ protocol.Invocation) protocol.Result {

	logger.Errorf("The invocation was rejected. url: %s", url.String())
	return &protocol.RPCResult{}
}

// GetOnlyLogRejectedExecutionHandler will return the instance of OnlyLogRejectedExecutionHandler
func GetOnlyLogRejectedExecutionHandler() filter.RejectedExecutionHandler {
	onlyLogHandlerOnce.Do(func() {
		onlyLogHandlerInstance = &OnlyLogRejectedExecutionHandler{}
	})
	return onlyLogHandlerInstance
}

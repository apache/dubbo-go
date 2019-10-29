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
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	filterCommon "github.com/apache/dubbo-go/filter/common"
	"github.com/apache/dubbo-go/protocol"
)

const HandlerName = "log"

func init() {
	extension.SetRejectedExecutionHandler(HandlerName, GetOnlyLogRejectedExecutionHandler)
	extension.SetRejectedExecutionHandler(constant.DEFAULT_KEY, GetOnlyLogRejectedExecutionHandler)
}

var onlyLogHandlerInstance *OnlyLogRejectedExecutionHandler
var onlyLogHandlerOnce sync.Once

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
 */
type OnlyLogRejectedExecutionHandler struct {
}

func (handler *OnlyLogRejectedExecutionHandler) RejectedExecution(url common.URL, invocation protocol.Invocation) protocol.Result {
	logger.Errorf("The invocation was rejected. url: %s", url.String())
	return &protocol.RPCResult{}
}

func GetOnlyLogRejectedExecutionHandler() filterCommon.RejectedExecutionHandler {
	onlyLogHandlerOnce.Do(func() {
		onlyLogHandlerInstance = &OnlyLogRejectedExecutionHandler{}
	})
	return onlyLogHandlerInstance
}

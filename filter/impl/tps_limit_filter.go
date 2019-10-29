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
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	_ "github.com/apache/dubbo-go/filter/common/impl"
	_ "github.com/apache/dubbo-go/filter/impl/tps/impl"
	"github.com/apache/dubbo-go/protocol"
)

const (
	TpsLimitFilterKey = "tps"
)

func init() {
	extension.SetFilter(TpsLimitFilterKey, GetTpsLimitFilter)
}

/**
 * if you wish to use the TpsLimiter, please add the configuration into your service provider configuration:
 * for example:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service", # it should be the name of limiter. if the value is 'default',
 *                                  # the MethodServiceTpsLimiterImpl will be used.
 *   tps.limit.rejected.handler: "default", # optional, or the name of the implementation
 *   if the value of 'tps.limiter' is nil or empty string, the tps filter will do nothing
 */
type TpsLimitFilter struct {
}

func (t TpsLimitFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	url := invoker.GetUrl()
	tpsLimiter := url.GetParam(constant.TPS_LIMITER_KEY, "")
	rejectedExeHandler := url.GetParam(constant.TPS_REJECTED_EXECUTION_HANDLER_KEY, constant.DEFAULT_KEY)
	if len(tpsLimiter) > 0 {
		allow := extension.GetTpsLimiter(tpsLimiter).IsAllowable(invoker.GetUrl(), invocation)
		if allow {
			return invoker.Invoke(invocation)
		}
		logger.Errorf("The invocation was rejected due to over the tps limitation, url: %s ", url.String())
		return extension.GetRejectedExecutionHandler(rejectedExeHandler).RejectedExecution(url, invocation)
	}
	return invoker.Invoke(invocation)
}

func (t TpsLimitFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func GetTpsLimitFilter() filter.Filter {
	return &TpsLimitFilter{}
}

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

package filter

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// RejectedExecutionHandler is the interface which defines the handler to handle exceptions from invoking filters.
// If the invocation cannot pass any validation in filter, like ExecuteLimitFilter and TpsLimitFilter,
// the implementation will be used. The common case is that sometimes you want to return the default
// value when the request was rejected.  Or you want to be warned if any request was rejected.
// In such situation, implement this interface and register it by invoking extension.SetRejectedExecutionHandler.

// RejectedExecutionHandler是一个接口，它定义了用于处理调用过滤器时产生的异常的处理器。
// 如果调用无法通过过滤器中的任何验证（例如ExecuteLimitFilter和TpsLimitFilter），
// 就会使用该接口的实现。常见的情况是，有时你希望在请求被拒绝时返回默认值，
// 或者在任何请求被拒绝时收到警告。
// 在这种情况下，实现此接口并通过调用extension.SetRejectedExecutionHandler进行注册即可。
//
// RejectedExecution method will be called if the invocation was rejected by some component.
type RejectedExecutionHandler interface {
	RejectedExecution(url *common.URL, invocation base.Invocation) result.Result
}

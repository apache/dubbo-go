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

package echo

import (
	"context"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	extension.SetFilter(constant.EchoFilterKey, func() filter.Filter {
		return &Filter{}
	})
}

// Filter health check
// RPCService need a Echo method in consumer, if you want to use Filter
// eg:
//		Echo func(ctx context.Context, arg interface{}, rsp *Xxx) error
type Filter struct{}

// Invoke response to the callers with its first argument.
func (f *Filter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking echo filter.")
	logger.Debugf("%v,%v", invocation.MethodName(), len(invocation.Arguments()))
	if invocation.MethodName() == constant.ECHO && len(invocation.Arguments()) == 1 {
		return &protocol.RPCResult{
			Rest:  invocation.Arguments()[0],
			Attrs: invocation.Attachments(),
		}
	}

	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *Filter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {

	return result
}

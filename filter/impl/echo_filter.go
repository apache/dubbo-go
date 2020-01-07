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
	"github.com/apache/dubbo-go/protocol"
)

const (
	ECHO = "echo"
)

func init() {
	extension.SetFilter(ECHO, GetFilter)
}

// RPCService need a Echo method in consumer, if you want to use EchoFilter
// eg:
//		Echo func(ctx context.Context, arg interface{}, rsp *Xxx) error
type EchoFilter struct{}

func (ef *EchoFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking echo filter.")
	logger.Debugf("%v,%v", invocation.MethodName(), len(invocation.Arguments()))
	if invocation.MethodName() == constant.ECHO && len(invocation.Arguments()) == 1 {
		return &protocol.RPCResult{
			Rest:  invocation.Arguments()[0],
			Attrs: invocation.Attachments(),
		}
	}

	return invoker.Invoke(invocation)
}

func (ef *EchoFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func GetFilter() filter.Filter {
	return &EchoFilter{}
}
